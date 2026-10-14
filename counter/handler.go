package main

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type viewsStore interface {
	read(context.Context) (int64, error)
	increment(context.Context) (int64, error)
}

type requestLimiter struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
	clock  func() time.Time
}

func newLimiter(clock func() time.Time) *requestLimiter {
	return &requestLimiter{tokens: 20, last: clock(), clock: clock}
}

func (l *requestLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock()
	l.tokens = min(20, l.tokens+max(0, now.Sub(l.last).Seconds())*2)
	l.last = now
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

type badgeHandler struct {
	store   viewsStore
	logger  *slog.Logger
	limiter *requestLimiter
}

func newBadgeHandler(store viewsStore, logger *slog.Logger) *badgeHandler {
	return &badgeHandler{store: store, logger: logger, limiter: newLimiter(time.Now)}
}

func (h *badgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Path != "/badge.svg" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.RawQuery != "" || r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if !h.limiter.allow() {
		w.Header().Set("Retry-After", "1")
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var count int64
	var err error
	if r.Method == http.MethodGet && strings.HasPrefix(r.UserAgent(), "github-camo") {
		count, err = h.store.increment(ctx)
	} else {
		count, err = h.store.read(ctx)
	}
	if err != nil {
		h.logger.Error("counter request failed", "error", err)
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "Request failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if _, err := w.Write([]byte(renderBadge(count))); err != nil {
		h.logger.Error("badge response failed", "error", err)
	}
}
