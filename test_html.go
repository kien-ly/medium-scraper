package main

import (
	"fmt"
	"io"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func main() {
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_120),
	}
	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)

	req, _ := http.NewRequest(http.MethodGet, "https://medium.com/@neilb_86943/claude-skills-death-to-manual-exploratory-data-analysis-0df694172294", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Status:", resp.StatusCode)

	if strings.Contains(string(body), "window.__APOLLO_STATE__") {
		fmt.Println("Found APOLLO_STATE!")
	} else {
		fmt.Println("Length:", len(body))
	}
}
