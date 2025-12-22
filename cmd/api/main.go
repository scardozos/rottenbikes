package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/scardozos/rottenbikes/cmd/api/email"
	"github.com/scardozos/rottenbikes/cmd/api/httpserver"
	"github.com/scardozos/rottenbikes/internal/domain"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	// TODO: load from env/config
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		user := getEnv("DB_USER", "rottenbikes")
		pass := getEnv("DB_PASSWORD", "rottenbikes")
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		dbname := getEnv("DB_NAME", "rottenbikes")
		sslmode := getEnv("DB_SSLMODE", "disable")
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, dbname, sslmode)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := domain.NewStore(db)
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	// Initialize Email Sender
	var sender email.EmailSender = &email.NoopSender{}
	mailtrapToken := email.GetToken("MAILTRAP")
	if mailtrapToken != "" {
		sender = &email.MailtrapSender{
			Token:     mailtrapToken,
			FromEmail: getEnv("EMAIL_FROM_ADDRESS", "hello@rottenbik.es"),
			FromName:  getEnv("EMAIL_FROM_NAME", "RottenBikes"),
		}
		log.Printf("Using Mailtrap email sender (token found)")
	} else {
		log.Printf("Using Noop email sender (EMAIL_SENDER_TOKEN_MAILTRAP not set)")
	}

	srv, err := httpserver.New(store, sender, ":"+port)
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
