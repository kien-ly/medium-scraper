# Medium Scraper (Go)

A high-performance Medium article scraper built in Go. It securely fetches Medium articles via their private GraphQL API, bypassing strict Cloudflare bot-protection and local ISP SNI blocks, and exports the data into clean, formatted Markdown files alongside their local images.

## How it Works (The Architecture & Bypass Strategy)

Medium enforces heavy bot-protection via Cloudflare which drops TLS handshakes (`EOF` or `SSL_ERROR_SYSCALL`) if it detects automated scrapers or if local ISPs intercept the SNI (Server Name Indication). 

To bypass this reliably, this scraper uses a multi-layered approach:

1. **Golang + TLS Fingerprinting (`bogdanfinn/tls-client`)**: 
   Standard HTTP clients (like Python's `requests` or Go's `net/http`) have obvious TLS fingerprints that Cloudflare instantly blocks. We use `tls-client` to spoof a legitimate `Chrome_146` TLS handshake.
2. **Search Engine Bot Spoofing**: 
   Medium specifically whitelists certain web crawlers. We spoof the `User-Agent` as `YandexMobileBot/3.0` and attach required Android Apollo GraphQL headers (`X-Obvious-CID`, `X-APOLLO-OPERATION-ID`) to get classified as a legitimate crawler.
3. **ISP SNI Bypass via Cloudflare WARP**: 
   In regions where Medium is blocked at the ISP level (e.g., Vietnam), the raw IP connection is forcefully dropped. We bypass this by routing the `tls-client` traffic through a local SOCKS5 proxy (`socks5://127.0.0.1:40000`) powered by Cloudflare WARP (`warp-cli`).
4. **Native Image Fetching**: 
   Images hosted on `miro.medium.com` are dynamically extracted from the GraphQL JSON (`Metadata.ID`). The scraper downloads these images concurrently using Go's standard `net/http` client (to natively follow redirects) and places them in a local `media/` folder.

## Prerequisites

1. **Go (Golang)**: Make sure you have Go installed (`brew install go`).
2. **Cloudflare WARP Proxy**: You must have `warp-cli` installed and running in proxy mode to bypass the SNI block.

### Setting up WARP Proxy
Run the following commands in your terminal to expose WARP as a local SOCKS5 proxy:
```bash
warp-cli mode proxy
warp-cli proxy port 40000
warp-cli connect
```

## Usage

1. Put the Medium URLs you want to scrape inside `input.txt` (one URL per line).
2. Run the Go scraper:
```bash
go run main.go
```

## Output Structure

The scraper will create an `output/` directory and organize each post into its own folder using the Post ID:

```text
output/
└── 0afd92a74b6f/
    ├── 0afd92a74b6f.md        # The parsed Markdown article
    └── media/
        ├── 1*Bki-2V42iY6.png  # Downloaded image
        └── 1*xZMoHnB5YKM.png  # Downloaded image
```

Images are automatically linked inside the Markdown file relative to the `media/` folder.
