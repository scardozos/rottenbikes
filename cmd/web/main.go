package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const distDir = "ui/dist"

// Cache policies. Cloudflare sits in front of the UI and follows these.
const (
	// index.html references the current build's hashed bundles, so it must
	// be revalidated on every load; a stale copy points at bundles that no
	// longer exist after a deploy.
	cacheNoCache = "no-cache"
	// Hashed build output never changes under the same name.
	cacheImmutable = "public, max-age=31536000, immutable"
	// Missing files must never be cached: during a rollout a new page can ask
	// an old pod for a new bundle, and a cached error would outlive it.
	cacheNoStore = "no-store"
)

// Directories whose file names contain a content hash (expo export).
var hashedPrefixes = []string{"/_expo/static/", "/assets/"}

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

	log.Info().Str("port", port).Msg("Web UI server listening")
	webSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           newHandler(distDir),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := webSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Web UI server failed to start")
	}
}

// newHandler serves the exported web UI from dir, injecting the
// EXPO_PUBLIC_* settings into index.html.
//
//   - Existing files are served as-is: hashed build output is cached forever,
//     anything else is revalidated.
//   - A missing file (a path with an extension, e.g. a bundle from another
//     build) is a 404 that nothing caches. It is never answered with
//     index.html: a 200 HTML response under a .js URL is what a CDN caches
//     and serves as the bundle, breaking the app for everyone.
//   - Any other path is an app route (e.g. /confirm/<token>) and gets
//     index.html (SPA fallback).
func newHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path

		if p == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}

		if p == "/" || p == "/index.html" {
			serveIndex(w, dir)
			return
		}

		// Clean the path to prevent directory traversal
		fullPath := filepath.Join(dir, filepath.Clean("/"+p))

		// Verify that the requested file path stays inside dir
		rel, err := filepath.Rel(dir, fullPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			if isHashed(p) {
				w.Header().Set("Cache-Control", cacheImmutable)
			} else {
				w.Header().Set("Cache-Control", cacheNoCache)
			}
			http.ServeFile(w, r, fullPath)
			return
		}

		if path.Ext(p) != "" {
			w.Header().Set("Cache-Control", cacheNoStore)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// Fallback to index.html for SPA routing
		serveIndex(w, dir)
	})
}

func isHashed(p string) bool {
	for _, prefix := range hashedPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

func serveIndex(w http.ResponseWriter, dir string) {
	data, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		w.Header().Set("Cache-Control", cacheNoStore)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", cacheNoCache)
	_, _ = w.Write([]byte(html))
}
