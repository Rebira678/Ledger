package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rebira678/ledger/internal/agent/llm"
)

func main() {
	b, err := os.ReadFile(".env")
	if err != nil {
		log.Fatal(err)
	}
	
	var apiKey, baseURL, model string
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "LEDGER_LLM_API_KEY=") {
			apiKey = strings.TrimPrefix(line, "LEDGER_LLM_API_KEY=")
		}
		if strings.HasPrefix(line, "LEDGER_LLM_BASE_URL=") {
			baseURL = strings.TrimPrefix(line, "LEDGER_LLM_BASE_URL=")
		}
		if strings.HasPrefix(line, "LEDGER_LLM_MODEL=") {
			model = strings.TrimPrefix(line, "LEDGER_LLM_MODEL=")
		}
	}

	client := llm.New(apiKey, baseURL, model, 30*time.Second)

	fmt.Println("Testing Gemini LLM Client ParseSMS...")

	senderID := "CBE"
	body := "Your new account balance is ETB 10,500.25 after you transferred 1500 to Abebe Kebede on Oct 1. Txn Ref: TRN-998877"

	ctx := context.Background()
	parsed, err := client.ParseSMS(ctx, senderID, body)
	if err != nil {
		log.Fatalf("ParseSMS failed: %v", err)
	}

	fmt.Printf("Success! Parsed Transaction:\n")
	fmt.Printf("Amount: %s\n", parsed.Amount.String())
	fmt.Printf("Currency: %s\n", parsed.Currency)
	fmt.Printf("Direction: %s\n", parsed.Direction)
	fmt.Printf("Counterparty: %s\n", parsed.Counterparty)
	fmt.Printf("Reference: %s\n", parsed.Reference)
	if parsed.Balance != nil {
		fmt.Printf("Balance: %s\n", parsed.Balance.String())
	}
}
