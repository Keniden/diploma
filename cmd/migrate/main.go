// Command migrate applies goose SQL migrations from migrations/ to the
// configured PostgreSQL database.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	url := flag.String("url", os.Getenv("POSTGRES_DSN"), "PostgreSQL DSN")
	flag.Parse()
	if *url == "" {
		*url = "postgres://postgres:postgres@localhost:5432/last_mile?sslmode=disable"
	}

	if err := run(context.Background(), *url); err != nil {
		log.Fatalf("migrate: %v", err)
	}
}

func run(ctx context.Context, url string) error {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("up: %w", err)
	}
	log.Printf("goose: migrations applied")
	return nil
}

var _ = stdlib.GetDefaultDriver // ensure pgx stdlib driver registered
