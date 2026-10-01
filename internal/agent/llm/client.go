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
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/rebira678/ledger/internal/domain"
)

// ErrNotConfigured is returned when LEDGER_LLM_API_KEY is empty.
var ErrNotConfigured = errors.New("llm: not configured (set LEDGER_LLM_API_KEY)")

// Client calls an OpenAI-compatible chat-completions endpoint.
type Client struct {
	apiKey       string
	baseURL      string
	model        string
	timeout      time.Duration
	httpClient   *http.Client
	visionClient *http.Client // longer timeout for image payloads
}

// New builds a Client. apiKey may be empty → ErrNotConfigured on use.
func New(apiKey, baseURL, model string, timeout time.Duration) *Client {
	visionTimeout := timeout * 4 // 4x base timeout for vision requests
	if visionTimeout < 120*time.Second {
		visionTimeout = 120 * time.Second
	}
	return &Client{
		apiKey: apiKey, baseURL: baseURL, model: model, timeout: timeout,
		httpClient:   &http.Client{Timeout: timeout},
		visionClient: &http.Client{Timeout: visionTimeout},
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

	respStr, err := c.doReqWithClient(ctx, reqBody, c.visionClient)
	if err != nil {
		return nil, err
	}

	fmt.Printf("DEBUG ParseReceipt: mimeType=%s base64len=%d\n", mimeType, len(base64Image))
	fmt.Printf("DEBUG ParseReceipt: raw LLM response=%q\n", respStr)

	// Clean up potential markdown formatting from the response
	respStr = strings.TrimPrefix(respStr, "```json")
	respStr = strings.TrimPrefix(respStr, "```")
	respStr = strings.TrimSuffix(respStr, "```")
	respStr = strings.TrimSpace(respStr)

	fmt.Printf("DEBUG ParseReceipt: cleaned response=%q\n", respStr)

	var data ReceiptData
	if err := json.Unmarshal([]byte(respStr), &data); err != nil {
		return nil, fmt.Errorf("llm: failed to decode receipt JSON (%w): raw text: %q", err, respStr)
	}
	fmt.Printf("DEBUG ParseReceipt: parsed data=%+v\n", data)
	return &data, nil
}

// ParseSMS sends an unmatched SMS message to the LLM to extract transaction details.
func (c *Client) ParseSMS(ctx context.Context, senderID string, body string) (*domain.ParsedTransaction, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}

	systemPrompt := `You are an expert financial transaction parser. You will be given a bank or telecom SMS message that could not be parsed by normal scripts.
Your task is to extract the following information:
- amount: (float) the amount of money in the transaction. Must be positive.
- currency: (string) the currency (e.g. "ETB", "Birr", "USD"). Default to "ETB" if unknown.
- direction: (string) exactly "credit" if the user received/deposited money, or "debit" if the user sent/spent money.
- counterparty: (string) the name of the person or business the user sent money to or received money from.
- reference: (string) the transaction reference or receipt number.
- balance_units: (integer) the units of the remaining balance (e.g., for 12.34, units=12). Output 0 if not present.
- balance_cents: (integer) the cents of the remaining balance (e.g., for 12.34, cents=34). Output 0 if not present.

Return ONLY a valid JSON object matching this schema. No markdown formatting.
Schema: {"amount": 123.45, "currency": "ETB", "direction": "debit", "counterparty": "Alemayehu", "reference": "TXN123", "balance_units": 450, "balance_cents": 50}`

	userPrompt := fmt.Sprintf("Sender: %s\nMessage: %s", senderID, body)

	respStr, err := c.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	respStr = strings.TrimPrefix(respStr, "```json")
	respStr = strings.TrimPrefix(respStr, "```")
	respStr = strings.TrimSuffix(respStr, "```")
	respStr = strings.TrimSpace(respStr)

	var data struct {
		Amount       float64 `json:"amount"`
		Currency     string  `json:"currency"`
		Direction    string  `json:"direction"`
		Counterparty string  `json:"counterparty"`
		Reference    string  `json:"reference"`
		BalanceUnits int64   `json:"balance_units"`
		BalanceCents int64   `json:"balance_cents"`
	}
	if err := json.Unmarshal([]byte(respStr), &data); err != nil {
		return nil, fmt.Errorf("llm: failed to decode sms JSON (%w): raw text: %q", err, respStr)
	}

	money, _ := domain.ParseMoney(fmt.Sprintf("%.2f", data.Amount))
	dir := domain.DirectionDebit
	if strings.ToLower(data.Direction) == "credit" {
		dir = domain.DirectionCredit
	}
	
	var bal *domain.Money
	if data.BalanceUnits > 0 || data.BalanceCents > 0 {
		bal = &domain.Money{Units: data.BalanceUnits, Cents: data.BalanceCents}
	}

	return &domain.ParsedTransaction{
		Amount:       money,
		Currency:     data.Currency,
		Direction:    dir,
		Counterparty: data.Counterparty,
		Reference:    data.Reference,
		Balance:      bal,
	}, nil
}

func (c *Client) doReq(ctx context.Context, reqBody []byte) (string, error) {
	return c.doReqWithClient(ctx, reqBody, c.httpClient)
}

func (c *Client) doReqWithClient(ctx context.Context, reqBody []byte, client *http.Client) (string, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 5s, 15s, 30s + jitter
			backoff := time.Duration(5<<uint(attempt-1)) * time.Second
			jitter := time.Duration(rand.Int63n(int64(2 * time.Second)))
			select {
			case <-time.After(backoff + jitter):
			case <-ctx.Done():
				return "", ctx.Err()
			}
			fmt.Printf("DEBUG llm: retrying request (attempt %d/%d)\n", attempt+1, maxRetries+1)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			stringTrimSuffix(c.baseURL, "/")+"/chat/completions", bytes.NewReader(reqBody))
		if err != nil {
			return "", fmt.Errorf("llm: building request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("llm: calling provider: %w", err)
			continue // retry on network errors
		}

		// Read the full body first
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		resp.Body.Close()
		bodyBytes := buf.Bytes()

		// Retry on 429 (rate limit) and 503 (overloaded)
		if resp.StatusCode == 429 || resp.StatusCode == 503 {
			lastErr = fmt.Errorf("llm: provider returned status %d (will retry)", resp.StatusCode)
			continue
		}

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
			return "", fmt.Errorf("llm: provider returned status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var parsed chatResponse
		fmt.Printf("DEBUG Chat raw response: %s\n", string(bodyBytes))
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

	return "", fmt.Errorf("llm: all %d retries exhausted: %w", maxRetries+1, lastErr)
}

func stringTrimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}
