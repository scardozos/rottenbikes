package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Serve static files from ui/dist, but intercept index.html to inject env vars
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
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

	log.Printf("Web UI server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
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

	// Inject environment variables as a script tag
	// We only inject variables prefixed with EXPO_PUBLIC_ for security
	envScript := "<script>\n"
	envScript += "  window.EXPO_PUBLIC_API_URL = " + quote(apiUrl) + ";\n"
	envScript += "  window.EXPO_PUBLIC_HCAPTCHA_SITEKEY = " + quote(sitekey) + ";\n"
	envScript += "</script>\n"

	html := string(data)
	// Inject before </head>
	replacement := envScript + "</head>"
	html = strings.Replace(html, "</head>", replacement, 1)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func quote(s string) string {
	if s == "" {
		return "undefined"
	}
	return "'" + s + "'"
}
