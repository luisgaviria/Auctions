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
	"time"
)

// Cloudflare Browser Rendering REST API endpoints.
// /crawl starts a new crawl job; /crawl/{id} polls for results.
// The REST API is always asynchronous — there is no synchronous HTML response.
const cfCrawlEndpoint = "https://api.cloudflare.com/client/v4/accounts/%s/browser-rendering/crawl"
const cfJobEndpoint = "https://api.cloudflare.com/client/v4/accounts/%s/browser-rendering/crawl/%s"

// NOTE: The CF REST API does not support browser automation actions (click,
// wait, type). Those require a Workers Binding with Puppeteer/Playwright.
// CFetch is therefore a pure "render this URL and return HTML" utility.

type cfCrawlRequest struct {
	URL         string         `json:"url"`
	Formats     []string       `json:"formats"`               // ["html"]
	Depth       int            `json:"depth"`                 // 0 = target URL only, no link following
	GotoOptions *cfGotoOptions `json:"gotoOptions,omitempty"` // navigation options
}

type cfGotoOptions struct {
	WaitUntil string `json:"waitUntil"` // "networkidle0" waits for all AJAX to finish
	Timeout   int    `json:"timeout"`   // ms; 0 = use CF default
}

// cfCrawlResponse is the immediate POST response.
//
// The CF API has varied between returning `result` as a plain string job-ID
// and returning it as an object {"id": "..."}. We use json.RawMessage so we
// can handle both formats without a hard unmarshal failure.
type cfCrawlResponse struct {
	Result  json.RawMessage `json:"result"`
	Success bool            `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// extractJobID attempts to read a job ID from a cfCrawlResponse.Result.
// It tries a plain string first, then falls back to {"id": "..."}.
func extractJobID(raw json.RawMessage) (string, error) {
	// Format 1: "result": "abc-123"  (original CF spec)
	var id string
	if err := json.Unmarshal(raw, &id); err == nil && id != "" {
		return id, nil
	}
	// Format 2: "result": {"id": "abc-123"}  (2026 CF API)
	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.ID != "" {
		return obj.ID, nil
	}
	return "", fmt.Errorf("cfetch: cannot extract job ID from result: %s", string(raw))
}

// cfJobResponse is the poll response returned by GET /crawl/{id}.
type cfJobResponse struct {
	Result struct {
		Status  string `json:"status"` // "running"|"completed"|"errored"|"cancelled_*"
		Records []struct {
			URL    string `json:"url"`
			HTML   string `json:"html"`
			Status int    `json:"status"` // HTTP status of the fetched page
		} `json:"records"`
	} `json:"result"`
}

// CFetch fetches targetURL via Cloudflare Browser Rendering and returns the
// fully-rendered HTML of that page. It posts a crawl job and polls until
// the job completes or the context / 150-second deadline is reached.
//
// Reads CF_ACCOUNT_ID and CF_API_TOKEN from the environment.
func CFetch(ctx context.Context, targetURL string) (string, error) {
	accountID := os.Getenv("CF_ACCOUNT_ID")
	token := os.Getenv("CF_API_TOKEN")
	if accountID == "" || token == "" {
		return "", fmt.Errorf("cfetch: CF_ACCOUNT_ID or CF_API_TOKEN not set in environment")
	}

	payload := cfCrawlRequest{
		URL:     targetURL,
		Formats: []string{"html"},
		Depth:   1,
		GotoOptions: &cfGotoOptions{
			WaitUntil: "networkidle0",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("cfetch: marshal: %w", err)
	}

	log.Printf("[cfetch] POST /crawl url=%s", targetURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf(cfCrawlEndpoint, accountID), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("cfetch: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cfetch: http post: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	log.Printf("[cfetch] POST response status=%d body=%.300s", resp.StatusCode, string(raw))

	var result cfCrawlResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("cfetch: unmarshal response: %w", err)
	}
	if !result.Success {
		msg := "unknown CF error"
		if len(result.Errors) > 0 {
			msg = result.Errors[0].Message
		}
		log.Printf("[cfetch] FAIL url=%s err=%s", targetURL, msg)
		return "", fmt.Errorf("cfetch: api error for %s: %s", targetURL, msg)
	}

	jobID, err := extractJobID(result.Result)
	if err != nil {
		return "", err
	}
	log.Printf("[cfetch] ASYNC url=%s job_id=%s — polling", targetURL, jobID)
	return pollCFJob(ctx, accountID, token, jobID)
}

// pollCFJob polls GET /crawl/{id} until the job status is "completed",
// an error status is returned, or the context / 150-second deadline fires.
func pollCFJob(ctx context.Context, accountID, token, jobID string) (string, error) {
	pollURL := fmt.Sprintf(cfJobEndpoint, accountID, jobID)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	deadline := time.After(150 * time.Second)
	client := &http.Client{Timeout: 15 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("cf job %s: %w", jobID, ctx.Err())
		case <-deadline:
			return "", fmt.Errorf("cf job %s timed out after 150s", jobID)
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
			if err != nil {
				continue
			}
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("[cfetch] poll error job_id=%s err=%v — retrying", jobID, err)
				continue // transient — keep polling
			}

			var jr cfJobResponse
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			json.Unmarshal(body, &jr) //nolint:errcheck — parse failure treated as "running"

			log.Printf("[cfetch] poll job_id=%s status=%s records=%d",
				jobID, jr.Result.Status, len(jr.Result.Records))

			switch jr.Result.Status {
			case "completed":
				if len(jr.Result.Records) == 0 {
					return "", fmt.Errorf("cf job %s completed with no records", jobID)
				}
				html := jr.Result.Records[0].HTML
				log.Printf("[cfetch] OK job_id=%s html_bytes=%d", jobID, len(html))
				return html, nil
			case "errored":
				return "", fmt.Errorf("cf job %s errored", jobID)
			case "cancelled_due_to_timeout":
				return "", fmt.Errorf("cf job %s cancelled: timeout", jobID)
			case "cancelled_due_to_limits":
				return "", fmt.Errorf("cf job %s cancelled: limits exceeded", jobID)
			case "cancelled_by_user":
				return "", fmt.Errorf("cf job %s cancelled by user", jobID)
			}
			// "running" | "" — keep waiting
		}
	}
}
