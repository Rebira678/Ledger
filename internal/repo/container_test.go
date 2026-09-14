package repo_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// postgresContainer starts a real PostgreSQL 16 container and returns its DSN.
func postgresContainer(ctx context.Context) (string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "ledger",
			"POSTGRES_PASSWORD": "ledger",
			"POSTGRES_DB":       "ledger_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", fmt.Errorf("starting postgres container: %w", err)
	}
	host, err := c.Host(ctx)
	if err != nil {
		return "", err
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return "", err
	}
	_ = os.Setenv("LEDGER_TESTCONTAINER", c.GetContainerID())
	return fmt.Sprintf("postgres://ledger:ledger@%s:%s/ledger_test?sslmode=disable", host, port.Port()), nil
}
