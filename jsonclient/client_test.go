package jsonclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func TestClient_Get_Success(t *testing.T) {
	// Make a test server with httptest.Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua == "" {
			t.Error("expected User-Agent header to be set")
		}

		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/users/1" {
			t.Errorf("expected path /users/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(User{ID: 1, Name: "Alice", Email: "alice@example.com"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := client.Get(ctx, "/users/1", &user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != 1 || user.Name != "Alice" {
		t.Errorf("unexpected user: %+v", user)
	}
}

func TestClient_Post_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(User{ID: 42, Name: req.Name, Email: req.Email})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	req := CreateUserRequest{Name: "Bob", Email: "bob@example.com"}
	var user User
	err := client.Post(ctx, "/users", &req, &user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != 42 || user.Name != "Bob" {
		t.Errorf("unexpected user: %+v", user)
	}
}

func TestClient_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "user not found"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	var resp ErrorResponse
	err := client.Get(ctx, "/users/999", &resp)

	respErr, ok := err.(*ResponseError)
	if !ok {
		t.Fatalf("expected ResponseError, got %T: %v", err, err)
	}

	if respErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", respErr.StatusCode)
	}

	if len(respErr.Body) == 0 {
		t.Error("expected error body to be preserved")
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := client.Get(ctx, "/slow", nil)
	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}
}

func TestClient_CustomUserAgent(t *testing.T) {
	expectedUA := "MyApp/2.0"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != expectedUA {
			t.Errorf("expected User-Agent %q, got %q", expectedUA, ua)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, WithUserAgent(expectedUA))
	ctx := context.Background()

	err := client.Get(ctx, "/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Reuse(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"count":%d}`, requestCount)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		var result map[string]int
		err := client.Get(ctx, "/", &result)
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
		if result["count"] != i+1 {
			t.Errorf("request %d: expected count %d, got %d", i, i+1, result["count"])
		}
	}

	if requestCount != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount)
	}
}

func TestClient_CustomHTTPClient(t *testing.T) {
	customTimeout := 1 * time.Second
	customClient := &http.Client{
		Timeout: customTimeout,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, WithHTTPClient(customClient))
	if client.httpClient.Timeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, client.httpClient.Timeout)
	}

	ctx := context.Background()
	err := client.Get(ctx, "/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_NilResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	err := client.Get(ctx, "/nocontent", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
