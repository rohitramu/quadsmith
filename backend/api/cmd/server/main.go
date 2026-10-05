package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/jackc/pgx/v5/pgxpool"
	
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
	"quadsmith/api/internal/server"
	"quadsmith/api/internal/store"
)

func main() {
	// 1. Connect to PostgreSQL
	// In MVP, fallback to a local default if DATABASE_URL is missing
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@localhost:5432/quadsmith"
	}

	pool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// 2. Initialize Store (with CEL Engine)
	compStore, err := store.NewComponentStore(pool)
	if err != nil {
		log.Fatalf("Failed to initialize component store: %v\n", err)
	}
	buildStore := store.NewBuildStore(pool)

	// 3. Initialize ConnectRPC Service Handler
	qsServer := server.NewQuadsmithServer(compStore, buildStore)

	mux := http.NewServeMux()
	path, handler := quadsmithconnect.NewQuadsmithAPIHandler(qsServer)
	mux.Handle(path, handler)

	// 4. Start HTTP/2 Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting Quadsmith API Server on port %s...\n", port)
	
	// Use h2c so we can support HTTP/2 without TLS for local development
	err = http.ListenAndServe(
		":"+port,
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
