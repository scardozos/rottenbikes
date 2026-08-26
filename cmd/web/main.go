package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") == "local" || os.Getenv("ENV") == "" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02T15:04:05Z07:00"})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Serve static files from ui/dist, but intercept index.html to inject env vars
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}

		if path == "/" || path == "/index.html" {
			serveIndex(w)
			return
		}

		// Clean the path to prevent directory traversal
		cleanedPath := filepath.Clean(path)
		fullPath := filepath.Join("ui/dist", cleanedPath)

		// Verify that the requested file path stays inside ui/dist
		rel, err := filepath.Rel("ui/dist", fullPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Check if file exists in ui/dist
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// Fallback to index.html for SPA routing
			serveIndex(w)
			return
		}

		http.ServeFile(w, r, fullPath)
	})

	log.Info().Str("port", port).Msg("Web UI server listening")
	webSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := webSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Web UI server failed to start")
	}
}

func serveIndex(w http.ResponseWriter) {
	data, err := os.ReadFile("./ui/dist/index.html")
	if err != nil {
		http.Error(w, "Could not read index.html", http.StatusInternalServerError)
		return
	}

	apiUrl := os.Getenv("EXPO_PUBLIC_API_URL")
	sitekey := os.Getenv("EXPO_PUBLIC_HCAPTCHA_SITEKEY")

	apiUrlBytes, _ := json.Marshal(apiUrl)
	sitekeyBytes, _ := json.Marshal(sitekey)

	// Inject environment variables as a script tag
	// We only inject variables prefixed with EXPO_PUBLIC_ for security
	envScript := "<script>\n"
	envScript += "  window.EXPO_PUBLIC_API_URL = " + string(apiUrlBytes) + ";\n"
	envScript += "  window.EXPO_PUBLIC_HCAPTCHA_SITEKEY = " + string(sitekeyBytes) + ";\n"
	envScript += "</script>\n"

	html := string(data)
	// Inject before </head>
	replacement := envScript + "</head>"
	html = strings.Replace(html, "</head>", replacement, 1)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
