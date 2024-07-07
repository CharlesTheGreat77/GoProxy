# GoProxy
Scrape proxies with a FAST concurrent proxy scraper written in Go! 😄


# Prerequisite
| Prerequisite | Version |
|--------------|---------|
| Go           |  <=1.22 |


# Install Go 🚀
MacOS
```
brew install golang
```
Windows 
```
https://go.dev/doc/install
```
Linux
```
apt install golang
```

# Install 
```
git clone https://github.com/CharlesTheGreat77/GoProxy
cd GoProxy/
go mod init main
go mod tidy
go build -o goproxy main.go
sudo mv /usr/local/bin
goproxy -h
```

# Demo 🫡
![demo](https://github.com/CharlesTheGreat77/GoProxy/assets/27988707/aabf821e-fa81-4b5e-847f-dacfe0518833)


# Usage 👀
```
Usage of ./main:
  -max int
    	specify maximum number of proxies (default 10)
  -output string
    	specify output file (default "proxies.txt")
  -scrape string
    	specify option to scrape from [sslproxies, proxyscrape] (default "proxyscrape")
```

# Default 
```
./main
```
- Validates 10 proxies from proxyscrape by default and saves to proxies.txt
