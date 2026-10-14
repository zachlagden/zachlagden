package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("counter stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "/data/views.sqlite3"
	}
	initialCount := int64(0)
	if value := os.Getenv("INITIAL_COUNT"); value != "" {
		var err error
		initialCount, err = strconv.ParseInt(value, 10, 64)
		if err != nil || initialCount < 0 {
			return errors.New("INITIAL_COUNT must be a nonnegative integer")
		}
	}
	return serve(path, initialCount, logger)
}

func serve(path string, initialCount int64, logger *slog.Logger) (err error) {
	store, err := openStore(path, initialCount)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, store.close()) }()
	server := &http.Server{
		Addr:              ":8080",
		Handler:           newBadgeHandler(store, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8192,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	logger.Info("counter started", "address", server.Addr)
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
