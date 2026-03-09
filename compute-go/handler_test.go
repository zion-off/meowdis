package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func startStorageServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Setenv("STORAGE_ENDPOINT", server.URL)
	t.Setenv("STORAGE_TOKEN", "test")
	return server
}

func writeStorageResults(w http.ResponseWriter, results any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
}

func TestAuthMiddleware(t *testing.T) {
	handlerFn := authMiddleware(handler)
	os.Setenv("AUTH_TOKEN", "secret")

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("[]"))
	rec := httptest.NewRecorder()
	handlerFn(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("[]"))
	req.Header.Set("Authorization", "Bearer wrong")
	rec = httptest.NewRecorder()
	handlerFn(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerNonPost(t *testing.T) {
	handlerFn := authMiddleware(handler)
	os.Setenv("AUTH_TOKEN", "secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handlerFn(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandlerSingleCommand(t *testing.T) {
	handlerFn := authMiddleware(handler)
	os.Setenv("AUTH_TOKEN", "secret")
	server := startStorageServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeStorageResults(w, []any{})
	})
	defer server.Close()

	body, _ := json.Marshal([]any{"PING"})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handlerFn(rec, req)

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp["result"] != "PONG" {
		t.Fatalf("expected PONG, got %v", resp["result"])
	}
}

func TestHandlerNumericArgs(t *testing.T) {
	handlerFn := authMiddleware(handler)
	os.Setenv("AUTH_TOKEN", "secret")
	server := startStorageServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeStorageResults(w, [][]map[string]any{
			{},
			{},
			{{"value": "a"}, {"value": "b"}},
		})
	})
	defer server.Close()

	body, _ := json.Marshal([]any{"LRANGE", "k", 0, -1})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handlerFn(rec, req)

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	result := resp["result"].([]any)
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestHandlerPipeline(t *testing.T) {
	handlerFn := authMiddleware(handler)
	os.Setenv("AUTH_TOKEN", "secret")
	server := startStorageServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeStorageResults(w, [][][]map[string]any{
			{},
			{},
		})
	})
	defer server.Close()

	body, _ := json.Marshal([][]any{{"PING"}, {"PING", "hi"}})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handlerFn(rec, req)

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	result := resp["result"].([]any)
	if len(result) != 2 || result[0] != "PONG" || result[1] != "hi" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestHandlerInvalidJSON(t *testing.T) {
	handlerFn := authMiddleware(handler)
	os.Setenv("AUTH_TOKEN", "secret")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not-json"))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	handlerFn(rec, req)

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp["error"] != "ERR invalid request body" {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestCoerceStringSlice(t *testing.T) {
	values, ok := coerceStringSlice([]any{"a", json.Number("1"), float64(2)})
	if !ok {
		t.Fatalf("expected ok")
	}
	if len(values) != 3 || values[0] != "a" || values[1] != "1" || values[2] != "2" {
		t.Fatalf("unexpected values: %v", values)
	}

	_, ok = coerceStringSlice([]any{"a", true})
	if ok {
		t.Fatalf("expected ok=false for bool")
	}
}
