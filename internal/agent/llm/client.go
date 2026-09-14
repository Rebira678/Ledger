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
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Chat sends a single user message with a system prompt and returns the
// assistant's reply. Errors are wrapped with context at this boundary.
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	if !c.Configured() {
		return "", ErrNotConfigured
	}
	reqBody, err := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    []chatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Temperature: 0.3, // grounded summaries; low creativity by design
		MaxTokens:   400,
	})
	if err != nil {
		return "", fmt.Errorf("llm: encoding request: %w", err)
	}
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

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("llm: decoding response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || parsed.Error != nil {
		msg := ""
		if parsed.Error != nil {
			msg = parsed.Error.Message
		}
		return "", fmt.Errorf("llm: provider returned status %d: %s", resp.StatusCode, msg)
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
