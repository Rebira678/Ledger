// Package llm provides a real OpenAI-compatible Chat Completions client (D-07)
// used for the weekly narrative and clarifying-question phrasing (FR-5.4).
// If no API key is configured the client reports ErrNotConfigured — callers
// degrade explicitly rather than fabricating content.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrNotConfigured is returned when LEDGER_LLM_API_KEY is empty.
var ErrNotConfigured = errors.New("llm: not configured (set LEDGER_LLM_API_KEY)")

// Client calls an OpenAI-compatible chat-completions endpoint.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	timeout    time.Duration
	httpClient *http.Client
}

// New builds a Client. apiKey may be empty → ErrNotConfigured on use.
func New(apiKey, baseURL, model string, timeout time.Duration) *Client {
	return &Client{
		apiKey: apiKey, baseURL: baseURL, model: model, timeout: timeout,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Configured reports whether a key is present.
func (c *Client) Configured() bool { return c != nil && c.apiKey != "" }

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Chat sends a single user message with a system prompt and returns the assistant's reply.
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	if !c.Configured() {
		return "", ErrNotConfigured
	}
	reqBody, err := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Temperature: 0.3,
		MaxTokens:   400,
	})
	if err != nil {
		return "", fmt.Errorf("llm: encoding request: %w", err)
	}
	return c.doReq(ctx, reqBody)
}

// ReceiptData represents the extracted fields from a receipt image.
type ReceiptData struct {
	Merchant  string  `json:"merchant"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Date      string  `json:"date"`
	Direction string  `json:"direction"`
	Category  string  `json:"category"`
}

// ParseReceipt sends an image to the LLM to extract receipt details.
func (c *Client) ParseReceipt(ctx context.Context, base64Image string, mimeType string) (*ReceiptData, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}

	content := []map[string]any{
		{"type": "text", "text": "Extract the merchant name, total amount, currency (e.g. ETB, USD), date (YYYY-MM-DD), direction (must be either \"credit\" if the user received money, or \"debit\" if the user paid/sent money), and a category (choose one of: Groceries, Dining, Utilities, Transport, Income, Transfer, Other). Return ONLY a valid JSON object matching this schema: {\"merchant\": \"...\", \"amount\": 123.45, \"currency\": \"...\", \"date\": \"...\", \"direction\": \"debit\", \"category\": \"Dining\"}. No markdown formatting."},
		{
			"type": "image_url",
			"image_url": map[string]string{
				"url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image),
			},
		},
	}

	reqBody, err := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    []chatMessage{{Role: "user", Content: content}},
		Temperature: 0.1,
		MaxTokens:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("llm: encoding request: %w", err)
	}

	respStr, err := c.doReq(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	// Clean up potential markdown formatting from the response
	respStr = strings.TrimPrefix(respStr, "```json")
	respStr = strings.TrimPrefix(respStr, "```")
	respStr = strings.TrimSuffix(respStr, "```")
	respStr = strings.TrimSpace(respStr)

	var data ReceiptData
	if err := json.Unmarshal([]byte(respStr), &data); err != nil {
		return nil, fmt.Errorf("llm: failed to decode receipt JSON (%w): raw text: %q", err, respStr)
	}
	return &data, nil
}

func (c *Client) doReq(ctx context.Context, reqBody []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		stringTrimSuffix(c.baseURL, "/")+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("llm: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: calling provider: %w", err)
	}
	defer resp.Body.Close()

	// Read the full body first so we can surface raw API errors (e.g. Gemini 503 arrays)
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	bodyBytes := buf.Bytes()

	if resp.StatusCode != http.StatusOK {
		// Attempt to parse standard OpenAI error object
		var errObj struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error,omitempty"`
		}
		_ = json.Unmarshal(bodyBytes, &errObj)
		if errObj.Error != nil && errObj.Error.Message != "" {
			return "", fmt.Errorf("llm: provider returned status %d: %s", resp.StatusCode, errObj.Error.Message)
		}
		// Fallback to raw body for unexpected error shapes (like arrays)
		return "", fmt.Errorf("llm: provider returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var parsed chatResponse
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return "", fmt.Errorf("llm: decoding response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("llm: provider returned error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("llm: provider returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

func stringTrimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}
