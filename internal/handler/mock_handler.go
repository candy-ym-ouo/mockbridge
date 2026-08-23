package handler

import (
	"io"
	"mockbridge/internal/middleware"
	"mockbridge/internal/service"
	"net"
	"net/http"
	"strings"
)

type MockHandler struct {
	mocks   *service.MockService
	records *service.RecordService
}

func NewMockHandler(m *service.MockService, r *service.RecordService) *MockHandler {
	return &MockHandler{mocks: m, records: r}
}
func (h *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/mock/_ping" {
		writeJSON(w, 200, 0, "pong", nil)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/mock/") {
		writeJSON(w, 404, 40400, "not found", nil)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if err != nil {
		writeJSON(w, 400, 40000, "cannot read request body", nil)
		return
	}
	if len(body) > 1<<20 {
		writeJSON(w, 400, 40000, "request body exceeds 1MB", nil)
		return
	}
	response := h.mocks.Process(r.Context(), service.MockRequest{RequestID: middleware.RequestIDFrom(r.Context()), Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), RawQuery: r.URL.RawQuery, Headers: r.Header.Clone(), Body: body, ClientIP: clientIP(r)})
	for key, values := range response.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.Status)
	_, _ = w.Write(response.Body)
	if response.Recordable {
		h.records.SubmitRequest(r.Context(), response.Record)
	}
}
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
