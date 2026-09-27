package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testIndex = `<!DOCTYPE html><html><head><title>RottenBikes</title></head>` +
	`<body><script src="/_expo/static/js/web/index-new.js" defer></script></body></html>`

// newTestDist builds a dist directory shaped like `expo export -p web`.
func newTestDist(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"index.html":                        testIndex,
		"favicon.ico":                       "icon",
		"metadata.json":                     "{}",
		"_expo/static/js/web/index-new.js":  "console.log('new build')",
		"assets/icons/back-icon.35ba0e.png": "png",
	}
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestHandler(t *testing.T) {
	t.Setenv("EXPO_PUBLIC_API_URL", "https://api.example.com")
	t.Setenv("EXPO_PUBLIC_HCAPTCHA_SITEKEY", "site-key")
	h := newHandler(newTestDist(t))

	cases := []struct {
		name        string
		path        string
		status      int
		cache       string
		contentType string // prefix
		body        string // substring
	}{
		{"index", "/", 200, cacheNoCache, "text/html", "index-new.js"},
		{"index by name", "/index.html", 200, cacheNoCache, "text/html", "index-new.js"},
		{"app route falls back to index", "/confirm/abc123", 200, cacheNoCache, "text/html", "index-new.js"},
		{"directory falls back to index, no listing", "/_expo/static/js/web/", 200, cacheNoCache, "text/html", "index-new.js"},
		{"current bundle is immutable", "/_expo/static/js/web/index-new.js", 200, cacheImmutable, "text/javascript", "new build"},
		{"hashed asset is immutable", "/assets/icons/back-icon.35ba0e.png", 200, cacheImmutable, "image/png", "png"},
		{"unhashed file is revalidated", "/favicon.ico", 200, cacheNoCache, "", "icon"},
		{"unhashed json is revalidated", "/metadata.json", 200, cacheNoCache, "application/json", "{}"},
		// The regression: a bundle from another build must be a non-cacheable
		// 404, never index.html with 200 (which Cloudflare cached as the JS).
		{"bundle from another build is a 404", "/_expo/static/js/web/index-old.js", 404, cacheNoStore, "text/plain", "Not Found"},
		{"missing asset is a 404", "/assets/icons/missing.png", 404, cacheNoStore, "text/plain", "Not Found"},
		{"missing top-level file is a 404", "/robots.txt", 404, cacheNoStore, "text/plain", "Not Found"},
		{"traversal stays inside dist", "/../../etc/passwd", 200, cacheNoCache, "text/html", "index-new.js"},
		{"healthz", "/healthz", 200, "", "", "ok"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := get(t, h, c.path)
			if w.Code != c.status {
				t.Fatalf("status %d, want %d (body %q)", w.Code, c.status, w.Body.String())
			}
			if got := w.Header().Get("Cache-Control"); got != c.cache {
				t.Errorf("Cache-Control %q, want %q", got, c.cache)
			}
			if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, c.contentType) {
				t.Errorf("Content-Type %q, want prefix %q", got, c.contentType)
			}
			if !strings.Contains(w.Body.String(), c.body) {
				t.Errorf("body %q does not contain %q", w.Body.String(), c.body)
			}
			if c.status == 404 && strings.Contains(w.Body.String(), "<html") {
				t.Error("a missing file must not be answered with HTML")
			}
		})
	}
}

func TestIndexInjectsPublicSettings(t *testing.T) {
	t.Setenv("EXPO_PUBLIC_API_URL", `https://api.example.com/"quoted"`)
	t.Setenv("EXPO_PUBLIC_HCAPTCHA_SITEKEY", "site-key")
	w := get(t, newHandler(newTestDist(t)), "/")

	body := w.Body.String()
	for _, want := range []string{
		`window.EXPO_PUBLIC_API_URL = "https://api.example.com/\"quoted\"";`,
		`window.EXPO_PUBLIC_HCAPTCHA_SITEKEY = "site-key";`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index.html missing %s:\n%s", want, body)
		}
	}
	if strings.Index(body, "window.EXPO_PUBLIC_API_URL") > strings.Index(body, "</head>") {
		t.Error("settings must be injected before </head>")
	}
}

func TestMissingIndexIsNotCached(t *testing.T) {
	w := get(t, newHandler(t.TempDir()), "/")
	if w.Code != http.StatusInternalServerError || w.Header().Get("Cache-Control") != cacheNoStore {
		t.Errorf("got %d with Cache-Control %q", w.Code, w.Header().Get("Cache-Control"))
	}
}
