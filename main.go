package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"bytes"
	"time"

	net_http "net/http"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func main() {
	// Read URLs from input.txt
	content, err := os.ReadFile("input.txt")
	if err != nil {
		log.Fatalf("Failed to read input.txt: %v", err)
	}

	urls := strings.Split(strings.TrimSpace(string(content)), "\n")

	for _, urlStr := range urls {
		urlStr = strings.TrimSpace(urlStr)
		if urlStr == "" {
			continue
		}

		fmt.Printf("--- Processing: %s ---\n", urlStr)

		// Simple way to get post ID
		parts := strings.Split(urlStr, "-")
		if len(parts) == 0 {
			fmt.Println("Could not extract post ID")
			continue
		}
		postId := parts[len(parts)-1]
		fmt.Printf("Post ID: %s\n", postId)

		err := fetchGraphQLData(urlStr, postId)
		if err != nil {
			fmt.Printf("Error fetching data for %s: %v\n", urlStr, err)
		}
	}
}

func fetchGraphQLData(urlStr, postId string) error {
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_146),
		tls_client.WithProxyUrl("socks5://127.0.0.1:40000"),
		tls_client.WithNotFollowRedirects(),
		// We avoid using WithRandomTLSExtensionOrder as it sometimes breaks Cloudflare
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Payload that matches freedium's structure
	graphqlData := map[string]interface{}{
		"operationName": "FullPostQuery",
		"variables": map[string]interface{}{
			"postId":              postId,
			"postMeteringOptions": map[string]interface{}{},
		},
		"query": "query FullPostQuery($postId: ID!, $postMeteringOptions: PostMeteringOptions) { post(id: $postId) { __typename id title visibility clapCount readingTime isLocked previewContent { subtitle } creator { name } content(postMeteringOptions: $postMeteringOptions) { bodyModel { paragraphs { text type metadata { id } } } } } }",
	}

	jsonData, err := json.Marshal(graphqlData)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://medium.com/_/graphql", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1 (compatible; YandexMobileBot/3.0;")
	req.Header.Set("Accept", "multipart/mixed; deferSpec=20220824, application/json, application/json")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("X-Obvious-CID", "android")
	req.Header.Set("X-Xsrf-Token", "1")
	req.Header.Set("X-Client-Date", fmt.Sprintf("%d", time.Now().UnixMilli()))
	req.Header.Set("X-APOLLO-OPERATION-ID", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef") // Dummy hash
	req.Header.Set("X-APOLLO-OPERATION-NAME", "FullPostQuery")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Cache-Control", "public, max-age=-1")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status Code: %d\n", resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	// Parse JSON to extract content
	var responseData struct {
		Data struct {
			Post struct {
				Title   string `json:"title"`
				Content struct {
					BodyModel struct {
						Paragraphs []struct {
							Text     string `json:"text"`
							Type     string `json:"type"`
							Metadata struct {
								ID string `json:"id"`
							} `json:"metadata"`
						} `json:"paragraphs"`
					} `json:"bodyModel"`
				} `json:"content"`
			} `json:"post"`
		} `json:"data"`
	}

	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		return fmt.Errorf("failed to parse json: %w", err)
	}

	post := responseData.Data.Post
	if post.Title == "" {
		fmt.Printf("Response might be an error or block: %s\n", string(bodyBytes[:500]))
		return fmt.Errorf("no post data found")
	}

	// Create output folders
	outDir := fmt.Sprintf("output/%s", postId)
	mediaDir := fmt.Sprintf("%s/media", outDir)
	os.MkdirAll(mediaDir, 0755)

	var mdContent strings.Builder
	mdContent.WriteString(fmt.Sprintf("# %s\n\n", post.Title))

	for _, p := range post.Content.BodyModel.Paragraphs {
		switch p.Type {
		case "P": // paragraph
			mdContent.WriteString(fmt.Sprintf("%s\n\n", p.Text))
		case "H3": // h1
			mdContent.WriteString(fmt.Sprintf("## %s\n\n", p.Text))
		case "H4": // h2
			mdContent.WriteString(fmt.Sprintf("### %s\n\n", p.Text))
		case "BQ": // blockquote
			mdContent.WriteString(fmt.Sprintf("> %s\n\n", p.Text))
		case "PRE": // pre
			mdContent.WriteString(fmt.Sprintf("```\n%s\n```\n\n", p.Text))
		case "ULI": // li
			mdContent.WriteString(fmt.Sprintf("- %s\n", p.Text))
		case "OLI": // ol
			mdContent.WriteString(fmt.Sprintf("1. %s\n", p.Text))
		case "IMG": // image
			imgId := p.Metadata.ID
			if imgId != "" {
				imgUrl := fmt.Sprintf("https://miro.medium.com/max/1400/%s", imgId)
				fmt.Printf("Downloading image %s...\n", imgId)
				filename, err := downloadImage(imgUrl, mediaDir, imgId, client)
				if err == nil && filename != "" {
					altText := p.Text
					if altText == "" {
						altText = "Image"
					}
					mdContent.WriteString(fmt.Sprintf("![%s](media/%s)\n\n", altText, filename))
				} else {
					fmt.Printf("Failed to download image %s: %v\n", imgId, err)
				}
			}
		default:
			// Fallback
			mdContent.WriteString(fmt.Sprintf("%s\n\n", p.Text))
		}
	}

	outFileName := fmt.Sprintf("%s/%s.md", outDir, postId)
	err = os.WriteFile(outFileName, []byte(mdContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to save output: %w", err)
	}

	fmt.Printf("Successfully saved markdown to %s\n", outFileName)

	return nil
}

func downloadImage(imgUrl, destDir, imgId string, client tls_client.HttpClient) (string, error) {
	req, err := net_http.NewRequest(net_http.MethodGet, imgUrl, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Use standard http client to follow redirects
	stdClient := &net_http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := stdClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Figure out extension from content type, default to jpg
	ext := "jpg"
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "png") {
		ext = "png"
	} else if strings.Contains(contentType, "gif") {
		ext = "gif"
	} else if strings.Contains(contentType, "webp") {
		ext = "webp"
	}

	filename := imgId
	if !strings.Contains(imgId, ".") {
		filename = fmt.Sprintf("%s.%s", imgId, ext)
	}
	outPath := fmt.Sprintf("%s/%s", destDir, filename)

	err = os.WriteFile(outPath, bodyBytes, 0644)
	if err != nil {
		return "", err
	}

	return filename, nil
}
