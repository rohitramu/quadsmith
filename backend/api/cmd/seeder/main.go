package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func genUUID() string {
	u, _ := uuid.NewV7()
	return u.String()
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/quadsmith?sslmode=disable"
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	tx, err := db.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to begin tx: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	// Clean tables? The tables are fresh, but let's just insert.
	// Actually we should truncate to make it idempotent
	tables := []string{
		"builds", "motors", "frames", "batteries", "escs", "flight_controllers",
		"receivers", "video_transmitters", "antennas", "cameras", "propellers",
	}
	for _, table := range tables {
		tx.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE;")
	}

	fmt.Println("Seeding Full Database from JSON...")
	seedData(ctx, tx)

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to commit tx: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Seed complete!")
}
