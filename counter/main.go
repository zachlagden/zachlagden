package main

import (
	"context"
	"errors"
	"fmt"
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
	if err := run(os.Args[1:], logger); err != nil {
		logger.Error("counter stopped", "error", err)
		os.Exit(1)
	}
}

func run(args []string, logger *slog.Logger) error {
	if len(args) == 1 && args[0] == "healthcheck" {
		return healthcheck()
	}
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
	if len(args) == 2 && args[0] == "backup" {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("find existing database: %w", err)
		}
		return backup(path, args[1], logger)
	}
	if len(args) != 0 {
		return errors.New("usage: profile-views [healthcheck | backup destination]")
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

func backup(path string, destination string, logger *slog.Logger) (err error) {
	store, err := openStore(path, 0)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, store.close()) }()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := store.backup(ctx, destination); err != nil {
		return err
	}
	logger.Info("backup created", "path", destination)
	return nil
}

func healthcheck() error {
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest(http.MethodHead, "http://127.0.0.1:8080/badge.svg", nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if err := response.Body.Close(); err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck returned status %d", response.StatusCode)
	}
	return nil
}
