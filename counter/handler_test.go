package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testHandler(store viewsStore) *badgeHandler {
	return newBadgeHandler(store, slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func TestOnlyGitHubImageFetchesIncrement(t *testing.T) {
	store := testStore(t, 10)
	handler := testHandler(store)
	cases := []struct {
		method    string
		userAgent string
		count     int64
	}{
		{http.MethodGet, "Mozilla/5.0", 10},
		{http.MethodGet, "", 10},
		{http.MethodHead, "github-camo", 10},
		{http.MethodGet, "github-camo test", 11},
	}
	for _, test := range cases {
		request := httptest.NewRequest(test.method, "/badge.svg", nil)
		request.Header.Set("User-Agent", test.userAgent)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d", response.Code)
		}
		if count, err := store.read(context.Background()); err != nil || count != test.count {
			t.Fatalf("count = %d, error = %v, want %d", count, err, test.count)
		}
		if test.method == http.MethodHead && response.Body.Len() != 0 {
			t.Fatal("HEAD returned a body")
		}
	}
}

func TestInvalidRequestsDoNotIncrement(t *testing.T) {
	store := testStore(t, 10)
	handler := testHandler(store)
	cases := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodGet, "/", "", http.StatusNotFound},
		{http.MethodGet, "/health", "", http.StatusNotFound},
		{http.MethodGet, "/docs", "", http.StatusNotFound},
		{http.MethodGet, "/openapi.json", "", http.StatusNotFound},
		{http.MethodPost, "/reset", "", http.StatusNotFound},
		{http.MethodPost, "/badge.svg", "", http.StatusMethodNotAllowed},
		{http.MethodGet, "/badge.svg?count=999", "", http.StatusBadRequest},
		{http.MethodGet, "/badge.svg", "payload", http.StatusBadRequest},
	}
	for _, test := range cases {
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		request.Header.Set("User-Agent", "github-camo test")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, response.Code, test.status)
		}
	}
	if count, err := store.read(context.Background()); err != nil || count != 10 {
		t.Fatalf("count = %d, error = %v", count, err)
	}
}

func TestBadgeHeaders(t *testing.T) {
	handler := testHandler(testStore(t, 42))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/badge.svg", nil))
	for name, expected := range map[string]string{
		"Content-Type":            "image/svg+xml; charset=utf-8",
		"Cache-Control":           "no-cache",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'none'; sandbox",
	} {
		if actual := response.Header().Get(name); actual != expected {
			t.Errorf("%s = %q, want %q", name, actual, expected)
		}
	}
	if !strings.Contains(response.Body.String(), "Profile views: 42") {
		t.Fatal("badge does not contain the count")
	}
}

func TestRequestsAreRateLimited(t *testing.T) {
	store := testStore(t, 10)
	handler := testHandler(store)
	now := time.Now()
	handler.limiter = newLimiter(func() time.Time { return now })
	for range 20 {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/badge.svg", nil)
		request.Header.Set("User-Agent", "github-camo test")
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("allowed request returned %d", response.Code)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/badge.svg", nil))
	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") == "" {
		t.Fatalf("rate limited response = %d, headers = %v", response.Code, response.Header())
	}
	if count, err := store.read(context.Background()); err != nil || count != 30 {
		t.Fatalf("count = %d, error = %v", count, err)
	}
	now = now.Add(time.Second)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/badge.svg", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("refilled request returned %d", response.Code)
	}
}

type failingStore struct{}

func (failingStore) read(context.Context) (int64, error) {
	return 0, errors.New("private database failure details")
}

func (failingStore) increment(context.Context) (int64, error) {
	return 0, errors.New("private database failure details")
}

func TestStorageErrorsStayOutOfResponses(t *testing.T) {
	for _, userAgent := range []string{"", "github-camo test"} {
		handler := testHandler(failingStore{})
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/badge.svg", nil)
		request.Header.Set("User-Agent", userAgent)
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d", response.Code)
		}
		if strings.Contains(response.Body.String(), "database") {
			t.Fatal("response exposed database details")
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("failure response permits caching")
		}
	}
}
