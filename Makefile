.PHONY: build test test-unit test-integration lint run up down clean tidy

# Build both binaries
build:
	go build -o bin/ledger-server ./cmd/ledger-server
	go build -o bin/ledger-worker ./cmd/ledger-worker

# All tests (unit + integration against real Postgres via testcontainers)
test: test-unit test-integration

test-unit:
	go test ./... -run 'TestUnit_' -count=1

test-integration:
	go test ./internal/... -count=1

lint:
	gofmt -l ./cmd ./internal
	go vet ./...

tidy:
	go mod tidy

# Run server locally against Postgres started by docker compose
run:
	go run ./cmd/ledger-server

# Full local stack: postgres + server
up:
	docker compose up --build

down:
	docker compose down -v

clean:
	rm -rf bin
