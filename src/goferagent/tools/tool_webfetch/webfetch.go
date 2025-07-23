package tool_webfetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/elee1766/gofer/src/agent"
	"github.com/elee1766/gofer/src/goferagent/toolsutil"
)

// Tool name constant
const Name = "web_fetch"

const webFetchPrompt = `Fetches content from a URL and returns it in the specified format.

WHEN TO USE THIS TOOL:
- Use when you need to download content from a URL
- Helpful for retrieving documentation, API responses, or web content
- Useful for getting external information to assist with tasks

HOW TO USE:
- Provide the URL to fetch content from
- Specify the desired output format (text, markdown, or html)
- Optionally set a timeout for the request

FEATURES:
- Supports three output formats: text, markdown, and html
- Automatically handles HTTP redirects
- Sets reasonable timeouts to prevent hanging
- Validates input parameters before making requests

LIMITATIONS:
- Maximum response size is 5MB
- Only supports HTTP and HTTPS protocols
- Cannot handle authentication or cookies
- Some websites may block automated requests

TIPS:
- Use text format for plain text content or simple API responses
- Use markdown format for content that should be rendered with formatting
- Use html format when you need the raw HTML structure
- Set appropriate timeouts for potentially slow websites`

// WebFetchInput represents the parameters for web_fetch
type WebFetchInput struct {
	URL     string `json:"url" required:"true" description:"The URL to fetch content from"`
	Format  string `json:"format" required:"true" description:"The format to return the content in (text, markdown, or html)"`
	Timeout int    `json:"timeout,omitempty" description:"Optional timeout in seconds (max 120, default 30)"`
}

// WebFetchOutput represents the response from web_fetch
type WebFetchOutput struct {
	Content     string            `json:"content" description:"The fetched content in the requested format"`
	StatusCode  int               `json:"status_code" description:"HTTP status code of the response"`
	Headers     map[string]string `json:"headers,omitempty" description:"Selected HTTP headers from the response"`
	URL         string            `json:"url" description:"The final URL after any redirects"`
	ContentType string            `json:"content_type,omitempty" description:"Content-Type header from the response"`
}

// Tool returns the web_fetch tool definition using GenericTool
func Tool() (agent.Tool, error) {
	return agent.NewGenericTool(Name, webFetchPrompt, webFetchHandler)
}

// Legacy types for backward compatibility
type Params = WebFetchInput
type Response = WebFetchOutput

// webFetchHandler creates a type-safe handler for the web_fetch tool
func webFetchHandler(ctx context.Context, input WebFetchInput) (WebFetchOutput, error) {
	// Check for cancellation
	select {
	case <-ctx.Done():
		return WebFetchOutput{}, fmt.Errorf("operation cancelled")
	default:
	}

	// Validate URL
	if input.URL == "" {
		return WebFetchOutput{}, fmt.Errorf("URL parameter is required")
	}

	// Validate format
	format := strings.ToLower(input.Format)
	if format != "text" && format != "markdown" && format != "html" {
		return WebFetchOutput{}, fmt.Errorf("format must be one of: text, markdown, html")
	}

	// Validate URL scheme
	if !strings.HasPrefix(input.URL, "http://") && !strings.HasPrefix(input.URL, "https://") {
		return WebFetchOutput{}, fmt.Errorf("URL must start with http:// or https://")
	}

	// Set timeout with max limit
	if input.Timeout <= 0 {
		input.Timeout = 30
	} else if input.Timeout > 120 {
		input.Timeout = 120 // Max 2 minutes
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(input.Timeout) * time.Second,
		// Follow redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", input.URL, nil)
	if err != nil {
		return WebFetchOutput{}, fmt.Errorf("failed to create request: %v", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "gofer/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Check for cancellation before request
	select {
	case <-ctx.Done():
		return WebFetchOutput{}, fmt.Errorf("operation cancelled")
	default:
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return WebFetchOutput{}, fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return WebFetchOutput{}, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	// Read response body with size limit
	const maxSize = 5 * 1024 * 1024 // 5MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return WebFetchOutput{}, fmt.Errorf("failed to read response: %v", err)
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")

	// Process content based on format
	var processedContent string
	switch format {
	case "text":
		if strings.Contains(contentType, "text/html") {
			// Extract text from HTML
			text, err := extractTextFromHTML(content)
			if err != nil {
				toolsutil.GetLogger().Warn("Failed to extract text from HTML, returning raw content", "error", err)
				processedContent = content
			} else {
				processedContent = text
			}
		} else {
			processedContent = content
		}

	case "markdown":
		if strings.Contains(contentType, "text/html") {
			// Convert HTML to Markdown
			markdown, err := convertHTMLToMarkdown(content)
			if err != nil {
				toolsutil.GetLogger().Warn("Failed to convert HTML to Markdown, wrapping in code block", "error", err)
				processedContent = "```html\n" + content + "\n```"
			} else {
				processedContent = markdown
			}
		} else if strings.Contains(contentType, "application/json") {
			// Wrap JSON in code block
			processedContent = "```json\n" + content + "\n```"
		} else {
			// Wrap other content types in code block
			processedContent = "```\n" + content + "\n```"
		}

	case "html":
		processedContent = content

	default:
		processedContent = content
	}

	// Extract selected headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 && (key == "Content-Type" || key == "Content-Length" || key == "Last-Modified") {
			headers[key] = values[0]
		}
	}

	toolsutil.GetLogger().Info("Fetched web content",
		"url", input.URL,
		"status", resp.StatusCode,
		"size", len(body),
		"format", format,
	)

	return WebFetchOutput{
		Content:     processedContent,
		StatusCode:  resp.StatusCode,
		Headers:     headers,
		URL:         resp.Request.URL.String(),
		ContentType: contentType,
	}, nil
}

// extractTextFromHTML extracts plain text from HTML content
func extractTextFromHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Remove script and style tags
	doc.Find("script, style").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// Get text content
	text := doc.Text()

	// Clean up whitespace
	lines := strings.Split(text, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}

	return strings.Join(cleanedLines, "\n"), nil
}

// convertHTMLToMarkdown converts HTML content to Markdown
func convertHTMLToMarkdown(html string) (string, error) {
	// Configure the converter
	converter := md.NewConverter("", true, nil)

	// Convert HTML to Markdown
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to Markdown: %w", err)
	}

	// Clean up the markdown
	markdown = strings.TrimSpace(markdown)

	// Remove excessive newlines
	markdown = strings.ReplaceAll(markdown, "\n\n\n", "\n\n")

	return markdown, nil
}
