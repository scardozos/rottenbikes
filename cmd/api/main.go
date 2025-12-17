package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/scardozos/rottenbikes/cmd/api/httpserver"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func main() {
	// TODO: load from env/config
	dsn := "postgres://rottenbikes:rottenbikes@localhost:5432/rottenbikes?sslmode=disable"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := domain.NewStore(db)
	srv, err := httpserver.New(store, ":8080")
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Run server in background.
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
