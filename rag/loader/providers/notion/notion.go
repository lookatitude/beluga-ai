// Package notion provides a DocumentLoader that loads pages from Notion via
// its API. It implements the loader.DocumentLoader interface using direct HTTP
// calls to the Notion API.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/notion"
//
//	l, err := loader.New("notion", config.ProviderConfig{
//	    APIKey: "ntn_...",
//	})
//	docs, err := l.Load(ctx, "page-id-here")
package notion

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
)

const defaultBaseURL = "https://api.notion.com"

func init() {
	loader.Register("notion", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
		return New(cfg)
	})
}

// Loader loads pages from the Notion API.
type Loader struct {
	client *httpclient.Client
}

// Compile-time interface check.
var _ loader.DocumentLoader = (*Loader)(nil)

// New creates a new Notion document loader.
func New(cfg config.ProviderConfig) (*Loader, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("notion: API key (integration token) is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := httpclient.New(
		httpclient.WithBaseURL(baseURL),
		httpclient.WithBearerToken(cfg.APIKey),
		httpclient.WithHeader("Notion-Version", "2022-06-28"),
		httpclient.WithTimeout(timeout),
	)

	return &Loader{client: client}, nil
}

// blockChildren is the Notion API response for block children.
type blockChildren struct {
	Results []block `json:"results"`
	HasMore bool    `json:"has_more"`
}

// block represents a single Notion block.
type block struct {
	ID   string `json:"id"`
	Type string `json:"type"`

	Paragraph     *richTextBlock `json:"paragraph,omitempty"`
	Heading1      *richTextBlock `json:"heading_1,omitempty"`
	Heading2      *richTextBlock `json:"heading_2,omitempty"`
	Heading3      *richTextBlock `json:"heading_3,omitempty"`
	BulletedList  *richTextBlock `json:"bulleted_list_item,omitempty"`
	NumberedList  *richTextBlock `json:"numbered_list_item,omitempty"`
	Toggle        *richTextBlock `json:"toggle,omitempty"`
	Quote         *richTextBlock `json:"quote,omitempty"`
	Callout       *richTextBlock `json:"callout,omitempty"`
	Code          *codeBlock     `json:"code,omitempty"`
}

// richTextBlock contains rich text content.
type richTextBlock struct {
	RichText []richText `json:"rich_text"`
}

// codeBlock contains code content.
type codeBlock struct {
	RichText []richText `json:"rich_text"`
	Language string     `json:"language"`
}

// richText represents a rich text element.
type richText struct {
	PlainText string `json:"plain_text"`
}

// pageResponse is the Notion page retrieval response.
type pageResponse struct {
	ID         string `json:"id"`
	Properties map[string]property `json:"properties"`
}

// property represents a Notion page property.
type property struct {
	Type  string `json:"type"`
	Title []richText `json:"title,omitempty"`
}

// Load fetches a Notion page's blocks and returns them as a document.
func (l *Loader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if source == "" {
		return nil, fmt.Errorf("notion: page ID is required")
	}

	pageID := strings.ReplaceAll(source, "-", "")

	// Fetch page metadata for title.
	page, err := httpclient.DoJSON[pageResponse](ctx, l.client, "GET",
		fmt.Sprintf("/v1/pages/%s", pageID), nil)
	if err != nil {
		return nil, fmt.Errorf("notion: fetch page %q: %w", pageID, err)
	}

	title := extractTitle(page)

	// Fetch block children.
	blocks, err := httpclient.DoJSON[blockChildren](ctx, l.client, "GET",
		fmt.Sprintf("/v1/blocks/%s/children?page_size=100", pageID), nil)
	if err != nil {
		return nil, fmt.Errorf("notion: fetch blocks %q: %w", pageID, err)
	}

	content := extractContent(blocks.Results)
	if content == "" {
		return nil, nil
	}

	meta := map[string]any{
		"source":  source,
		"loader":  "notion",
		"page_id": page.ID,
		"title":   title,
	}

	return []schema.Document{{
		ID:       page.ID,
		Content:  content,
		Metadata: meta,
	}}, nil
}

// extractTitle gets the page title from properties.
func extractTitle(page pageResponse) string {
	for _, prop := range page.Properties {
		if prop.Type == "title" && len(prop.Title) > 0 {
			return prop.Title[0].PlainText
		}
	}
	return ""
}

// extractContent converts Notion blocks to plain text.
func extractContent(blocks []block) string {
	var parts []string
	for _, b := range blocks {
		text := blockText(b)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

// blockText extracts plain text from a block.
func blockText(b block) string {
	var rtb *richTextBlock
	switch b.Type {
	case "paragraph":
		rtb = b.Paragraph
	case "heading_1":
		rtb = b.Heading1
	case "heading_2":
		rtb = b.Heading2
	case "heading_3":
		rtb = b.Heading3
	case "bulleted_list_item":
		rtb = b.BulletedList
	case "numbered_list_item":
		rtb = b.NumberedList
	case "toggle":
		rtb = b.Toggle
	case "quote":
		rtb = b.Quote
	case "callout":
		rtb = b.Callout
	case "code":
		if b.Code != nil {
			return richTextToPlain(b.Code.RichText)
		}
	}
	if rtb != nil {
		return richTextToPlain(rtb.RichText)
	}
	return ""
}

// richTextToPlain concatenates rich text elements into plain text.
func richTextToPlain(rts []richText) string {
	var sb strings.Builder
	for _, rt := range rts {
		sb.WriteString(rt.PlainText)
	}
	return sb.String()
}
