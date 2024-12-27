package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ScrapeOption struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// default scraping options
func defaultScrapeOptions() map[string]*ScrapeOption {
	return map[string]*ScrapeOption{
		// they both can easily be scraped with the regex used.. no additional processing..
		"proxyscrape":   {Name: "proxyscrape", URL: "https://api.proxyscrape.com/v2/?request=getproxies&protocol=http&timeout=10000&country=all&ssl=all&anonymity=all"},
		"sslproxies":    {Name: "sslproxies", URL: "https://sslproxies.org"},
		"freeproxylist": {Name: "freeproxylist", URL: "https://free-proxy-list.net"},
	}
}

var maxGoRoutines int = 15
var guard chan struct{} = make(chan struct{}, maxGoRoutines)

// scrape from proxy option;
// return slice of proxies and errors
func fetchProxies(scrape_url string) ([]string, error) {
	var err error
	// scrape from proxyscrape
	req, err := http.NewRequest("GET", scrape_url, nil)
	if err != nil {
		err = errors.New("[-] Error configuring request with url")
		return nil, err
	}
	// add/change user agent as necessary
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	client := http.DefaultClient
	res, err := client.Do(req)

	if err != nil {
		err = errors.New("[-] Error Connecting")
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		err = errors.New("[*] Read error")
		return nil, err
	}
	// grab ip:port in html body with regexp
	re := regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}:\d{1,5}`)
	proxies := re.FindAllString(string(body), -1)
	return proxies, err
}

// validate all proxies until max proxies is met
func checkProxy(ctx context.Context, proxy string, validProxies *[]string, counter *int, maxProxies int, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() { <-guard }()
	var proxyUrl *url.URL
	// check if context is cancelled
	select {
	case <-ctx.Done():
		return
	default:
		var err error
		proxyUrl, err = url.Parse("http://" + proxy)
		if err != nil {
			fmt.Printf("[-] Error setting proxy..\nError: %v", err)
			return
		}
	}
	transport := &http.Transport{Proxy: http.ProxyURL(proxyUrl), DialContext: (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 1 * time.Second}).DialContext}
	client := http.Client{Transport: transport}
	req, err := http.NewRequest("GET", "https://google.com/", nil)
	if err != nil {
		return
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// seperate context for request to close connections after max is hit
	reqCtx, reqCancel := context.WithCancel(ctx)
	defer reqCancel()
	req = req.WithContext(reqCtx)
	res, err := client.Do(req)
	if err != nil {
		if e, ok := err.(*url.Error); ok {
			// seem to be hitting some google captchas for some proxies, use that as the proxy being valid
			if e.Err.Error() == "stopped after 10 redirects" || strings.Contains(err.Error(), "Get \"https://www.google.com/sorry/") {
				if *counter <= maxProxies-1 {
					fmt.Printf("[*] Valid Proxy: http://%v\n", proxy)
					*validProxies = append(*validProxies, proxy)
					*counter++
				}
			}
		}
		return
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		if *counter <= maxProxies-1 {
			fmt.Printf("[*] Valid Proxy: %v\n", proxy)
			*validProxies = append(*validProxies, proxy)
			*counter++
		}
	}
}

func main() {
	scrape := flag.String("scrape", "proxyscrape", "specify option to scrape from [sslproxies, proxyscrape]")
	maxProxies := flag.Int("max", 10, "specify maximum number of proxies")
	output := flag.String("output", "proxies.txt", "specify output file")
	flag.Parse()

	defaultOption := defaultScrapeOptions()
	selectedOption, ok := defaultOption[*scrape]
	if !ok {
		fmt.Printf("Invalid Scrape option: %s\n", *scrape)
		return
	}

	var wg sync.WaitGroup
	var counter int
	var validProxies []string
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("[*] Fetching proxies")
	start := time.Now()
	var proxies, err = fetchProxies(selectedOption.URL)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("[#] Proxies Found: %v\n", len(proxies))
	for _, proxy := range proxies {
		guard <- struct{}{}
		wg.Add(1)
		go checkProxy(ctx, string(proxy), &validProxies, &counter, *maxProxies, &wg)
		if counter >= *maxProxies {
			cancel()
			break
		}
	}

	wg.Wait()

	file, err := os.OpenFile(*output, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[-] Error: %v\n", err)
		return
	}
	defer file.Close()
	fmt.Printf("[*] Saving to proxies.txt\n\n")
	for _, proxy := range validProxies {
		if _, err := file.WriteString("http://" + proxy + "\n"); err != nil {
			fmt.Println("[-] Error appending to file")
		}
	}
	duration := time.Since(start)
	fmt.Printf("[*] Execution Time: %v\n", duration)
}
