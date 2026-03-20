package sites

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s"

// maxHTMLBytes is the maximum number of HTML bytes sent to Gemini.
// Gemini 1.5 Flash has a large context window, but trimming keeps costs low.
const maxHTMLBytes = 200_000

// geminiRequest is the JSON body for the Gemini generateContent API.
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// geminiResponse is the minimal subset of the generateContent response we use.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// rescuedAuction is the JSON shape Gemini is asked to return per auction.
type rescuedAuction struct {
	Address string `json:"address"`
	City    string `json:"city"`
	Date    string `json:"date"`
	Time    string `json:"time"`
	Deposit string `json:"deposit"`
	Status  string `json:"status"`
	Link    string `json:"link"`
}

// RescueWithAI sends raw HTML to Gemini 1.5 Flash and asks it to extract
// all real-estate auctions. Returns a []Auction on success, or an error if
// Gemini is unavailable or returns no usable data.
//
// Reads GEMINI_API_KEY from the environment.
func RescueWithAI(ctx context.Context, html string) ([]Auction, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("gemini: GEMINI_API_KEY not set")
	}

	// Truncate HTML to keep prompt size manageable.
	if len(html) > maxHTMLBytes {
		html = html[:maxHTMLBytes]
	}

	prompt := `Act as a web scraper. Extract all real estate auctions from the HTML below.
Return ONLY a valid JSON array (no markdown, no explanation) where each element has these keys:
  "address", "city", "date", "time", "deposit", "status", "link"
Use empty string "" for any field that is missing or unknown.
For "date", use the format "Mon DD, YYYY" if possible.

HTML:
` + html

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	url := fmt.Sprintf(geminiEndpoint, apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: http post: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: api returned %d: %s", resp.StatusCode, string(raw))
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(raw, &gemResp); err != nil {
		return nil, fmt.Errorf("gemini: unmarshal response: %w", err)
	}
	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini: no candidates in response")
	}

	text := gemResp.Candidates[0].Content.Parts[0].Text
	text = strings.TrimSpace(text)

	// Strip any accidental markdown code fences.
	if strings.HasPrefix(text, "```") {
		first := strings.Index(text, "\n")
		last := strings.LastIndex(text, "```")
		if first != -1 && last > first {
			text = strings.TrimSpace(text[first+1 : last])
		}
	}

	var rescued []rescuedAuction
	if err := json.Unmarshal([]byte(text), &rescued); err != nil {
		log.Printf("[gemini] JSON parse error: %v — raw: %.200s", err, text)
		return nil, fmt.Errorf("gemini: parse extracted JSON: %w", err)
	}

	auctions := make([]Auction, 0, len(rescued))
	for _, r := range rescued {
		if r.Address == "" {
			continue // skip rows with no address
		}
		auctions = append(auctions, Auction{
			Street:  r.Address,
			City:    r.City,
			Date:    r.Date,
			Time:    r.Time,
			Deposit: r.Deposit,
			Status:  r.Status,
			Url:     r.Link,
		})
	}

	log.Printf("[gemini] extracted %d auctions from HTML (%d bytes)", len(auctions), len(html))
	return auctions, nil
}
