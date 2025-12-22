package main

import (
	"log"
	"net/http"
	"os"
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

		// Check if file exists in ui/dist
		fullPath := "./ui/dist" + path
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

	// Inject environment variables as a script tag
	// We only inject variables prefixed with EXPO_PUBLIC_ for security
	envScript := "<script>\n"
	envScript += "  window.EXPO_PUBLIC_API_URL = " + quote(os.Getenv("EXPO_PUBLIC_API_URL")) + ";\n"
	envScript += "  window.EXPO_PUBLIC_HCAPTCHA_SITEKEY = " + quote(os.Getenv("EXPO_PUBLIC_HCAPTCHA_SITEKEY")) + ";\n"
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
