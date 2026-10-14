package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type counterStore struct {
	db *sql.DB
}

func openStore(path string, initialCount int64) (_ *counterStore, err error) {
	if initialCount < 0 {
		return nil, errors.New("initial count must not be negative")
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	query := url.Values{}
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(FULL)")
	dsn := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: query.Encode()}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer func() {
		if err != nil {
			err = errors.Join(err, db.Close())
		}
	}()
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS profile_views (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		count INTEGER NOT NULL CHECK (typeof(count) = 'integer' AND count >= 0)
	)`); err != nil {
		return nil, fmt.Errorf("create counter table: %w", err)
	}
	if _, err = db.Exec("INSERT INTO profile_views (id, count) VALUES (1, ?) ON CONFLICT (id) DO NOTHING", initialCount); err != nil {
		return nil, fmt.Errorf("initialize counter: %w", err)
	}
	return &counterStore{db: db}, nil
}

func (s *counterStore) read(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT count FROM profile_views WHERE id = 1").Scan(&count)
	return count, err
}

func (s *counterStore) increment(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "UPDATE profile_views SET count = count + 1 WHERE id = 1 RETURNING count").Scan(&count)
	return count, err
}

func (s *counterStore) backup(ctx context.Context, destination string) error {
	path, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve backup path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	_, err = s.db.ExecContext(ctx, "VACUUM INTO ?", path)
	return err
}

func (s *counterStore) close() error {
	return s.db.Close()
}
