package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestUI_ServesIndex guards the go:embed wiring: if the static assets ever
// stop being embedded (e.g. an accidental package rename or embed pattern
// typo), this fails loudly instead of silently shipping a blank /ui/.
func TestUI_ServesIndex(t *testing.T) {
	m := &stubManager{}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html content type, got %s", ct)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("agentic-sim")) {
		t.Fatalf("expected index.html marker in body, got %s", rec.Body.String())
	}
}

func TestUI_ServesStaticAsset(t *testing.T) {
	m := &stubManager{}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/ui/app.js", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUI_UnknownAssetNotFound(t *testing.T) {
	m := &stubManager{}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/ui/nope.js", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUI_BareRedirectsToTrailingSlash(t *testing.T) {
	m := &stubManager{}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/ui", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/ui/" {
		t.Fatalf("expected redirect to /ui/, got %s", loc)
	}
}
