package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

type Config struct {
	DBUser           string `json:"DB_USER"`
	DBHost           string `json:"DB_HOST"`
	DBPort           string `json:"DB_PORT"`
	DBName           string `json:"DB_NAME"`
	DBSSLMode        string `json:"DB_SSLMODE"`
	APIPort          string `json:"API_PORT"`
	MetricsPort      string `json:"METRICS_PORT"`
	EmailSenderType  string `json:"EMAIL_SENDER_TYPE"`
	MailtrapTokenSet bool   `json:"MAILTRAP_TOKEN_SET"`
	EmailFromAddress string `json:"EMAIL_FROM_ADDRESS"`
	EmailFromName    string `json:"EMAIL_FROM_NAME"`
}

func main() {
	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}
	// Default to info level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Load Config
	cfg := Config{
		DBUser:           getEnv("DB_USER", "rottenbikes"),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBName:           getEnv("DB_NAME", "rottenbikes"),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
		APIPort:          getEnv("API_PORT", "8080"),
		MetricsPort:      getEnv("METRICS_PORT", "9091"),
		EmailSenderType:  "noop",
		EmailFromAddress: getEnv("EMAIL_FROM_ADDRESS", "hello@rottenbik.es"),
		EmailFromName:    getEnv("EMAIL_FROM_NAME", "RottenBikes"),
	}
	if os.Getenv("API_PORT") != "" {
		cfg.APIPort = os.Getenv("API_PORT")
	}

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
		log.Fatal().Err(err).Msg("failed to open db")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("failed to ping db")
	}

	store := domain.NewStore(db)
	port := cfg.APIPort

	// Initialize Email Sender
	var sender email.EmailSender = &email.NoopSender{}
	mailtrapToken := email.GetToken("MAILTRAP")
	if mailtrapToken != "" {
		cfg.EmailSenderType = "mailtrap"
		cfg.MailtrapTokenSet = true
		sender = &email.MailtrapSender{
			Token:     mailtrapToken,
			FromEmail: cfg.EmailFromAddress,
			FromName:  cfg.EmailFromName,
		}
	} else {
		cfg.EmailSenderType = "noop"
		cfg.MailtrapTokenSet = false
	}

	// Log Startup Config
	log.Info().Interface("config", cfg).Msg("starting service")

	srv, err := httpserver.New(store, sender, ":"+port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
	}
	// Metrics Server
	metricsPort := cfg.MetricsPort
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: metricsMux,
	}

	// Run server in background.
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	go func() {
		log.Info().Msgf("Metrics server listening on %s", metricsSrv.Addr)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("metrics server error")
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
	if err := metricsSrv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("metrics server shutdown failed")
	}
}
