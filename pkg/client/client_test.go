package client_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/jasonmiller-cc/parameters-core/pkg/client"
)

func TestGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/items/1" {
			t.Errorf("path = %s, want /items/1", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"widget"}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	var out struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := c.Get(context.Background(), "/items/1", &out); err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if out.ID != 1 || out.Name != "widget" {
		t.Errorf("out = %+v, want {ID:1 Name:widget}", out)
	}
}

func TestPost_SendsBodyAndAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want Bearer secret-token", got)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name":"new"}` {
			t.Errorf("body = %q, want {\"name\":\"new\"}", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, client.WithBearerToken("secret-token"))
	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.Post(context.Background(), "/items", map[string]string{"name": "new"}, &out); err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	if !out.OK {
		t.Error("expected out.OK = true")
	}
}

func TestDo_RetriesOn5xxThenSucceeds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name":"retry-me"}` {
			t.Errorf("attempt %d: body = %q, want {\"name\":\"retry-me\"}", n, body)
		}
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, client.WithRetry(3))
	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.Post(context.Background(), "/items", map[string]string{"name": "retry-me"}, &out); err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
	if !out.OK {
		t.Error("expected out.OK = true after retries succeeded")
	}
}

func TestDo_GivesUpAfterMaxRetries(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := client.New(srv.URL, client.WithRetry(2))
	err := c.Get(context.Background(), "/items", nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 3 { // initial + 2 retries
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestDo_4xxDoesNotRetry(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer srv.Close()

	c := client.New(srv.URL, client.WithRetry(2))
	err := c.Get(context.Background(), "/missing", nil)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (4xx should not retry)", got)
	}
}

func TestDelete(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL)
	if err := c.Delete(context.Background(), "/items/1"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	if !called {
		t.Error("server handler was not called")
	}
}
