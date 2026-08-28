package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	graphqlData := []byte(`{"operationName":"FullPostQuery","variables":{"postId":"0df694172294","postMeteringOptions":{}},"query":"query FullPostQuery($postId: ID!, $postMeteringOptions: PostMeteringOptions) { post(id: $postId) { __typename id title } }"}`)

	req, _ := http.NewRequest(http.MethodPost, "https://medium.com/_/graphql", bytes.NewBuffer(graphqlData))
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1 (compatible; YandexMobileBot/3.0;")
	req.Header.Set("Accept", "multipart/mixed; deferSpec=20220824, application/json, application/json")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("X-Obvious-CID", "android")
	req.Header.Set("X-Xsrf-Token", "1")
	req.Header.Set("X-Client-Date", fmt.Sprintf("%d", time.Now().UnixMilli()))
	req.Header.Set("X-APOLLO-OPERATION-ID", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	req.Header.Set("X-APOLLO-OPERATION-NAME", "FullPostQuery")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "Keep-Alive")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Println("Body:", string(bodyBytes))
}
