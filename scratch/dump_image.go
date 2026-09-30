package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	conn, err := pgx.Connect(context.Background(), "postgres://ledger:ledger@localhost:25432/ledger?sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	var data []byte
	err = conn.QueryRow(context.Background(), "SELECT file_bytes FROM statement_uploads ORDER BY created_at DESC LIMIT 1").Scan(&data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile("scratch/latest_upload.png", data, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Write failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Saved to scratch/latest_upload.png")
}
