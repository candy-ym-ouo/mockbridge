package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChainRequestIDCorsAndRecover(t *testing.T) {
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFrom(r.Context()) == "" {
			t.Error("missing request id")
		}
		panic("boom")
	}), slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 500 || rr.Header().Get("X-Request-ID") == "" || rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("code=%d headers=%v", rr.Code, rr.Header())
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodOptions, "/x", nil))
	if rr.Code != 204 {
		t.Fatalf("options=%d", rr.Code)
	}
}
