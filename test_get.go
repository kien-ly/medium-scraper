package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func main() {
	baseURL := "https://medium.com/_/graphql"
	variables := `{"postId":"0df694172294","postMeteringOptions":{}}`
	query := `query FullPostQuery($postId: ID!, $postMeteringOptions: PostMeteringOptions) { post(id: $postId) { __typename id title } }`

	fullURL := fmt.Sprintf("%s?operationName=FullPostQuery&variables=%s&query=%s", baseURL, url.QueryEscape(variables), url.QueryEscape(query))

	req, _ := http.NewRequest(http.MethodGet, fullURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Body:", string(body)[:200])
}
