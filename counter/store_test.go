package main

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func testStore(t *testing.T, initialCount int64) *counterStore {
	t.Helper()
	store, err := openStore(filepath.Join(t.TempDir(), "views.sqlite3"), initialCount)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.close(); err != nil {
			t.Error(err)
		}
	})
	return store
}

func TestStorePreservesCountAcrossRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "views.sqlite3")
	store, err := openStore(path, 42)
	if err != nil {
		t.Fatal(err)
	}
	if count, err := store.read(context.Background()); err != nil || count != 42 {
		t.Fatalf("initial count = %d, error = %v", count, err)
	}
	if count, err := store.increment(context.Background()); err != nil || count != 43 {
		t.Fatalf("incremented count = %d, error = %v", count, err)
	}
	if err := store.close(); err != nil {
		t.Fatal(err)
	}
	store, err = openStore(path, 900)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := store.close(); err != nil {
			t.Error(err)
		}
	}()
	if count, err := store.read(context.Background()); err != nil || count != 43 {
		t.Fatalf("reopened count = %d, error = %v", count, err)
	}
}

func TestConcurrentIncrements(t *testing.T) {
	store := testStore(t, 3)
	var group sync.WaitGroup
	for range 80 {
		group.Go(func() {
			if _, err := store.increment(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if count, err := store.read(context.Background()); err != nil || count != 83 {
		t.Fatalf("final count = %d, error = %v", count, err)
	}
}

func TestStoreRejectsNegativeSeed(t *testing.T) {
	if store, err := openStore(filepath.Join(t.TempDir(), "views.sqlite3"), -1); err == nil {
		if err := store.close(); err != nil {
			t.Error(err)
		}
		t.Fatal("negative seed was accepted")
	}
}

func TestStoreDoesNotOverflow(t *testing.T) {
	store := testStore(t, math.MaxInt64)
	if _, err := store.increment(context.Background()); err == nil {
		t.Fatal("overflowing increment succeeded")
	}
	if count, err := store.read(context.Background()); err != nil || count != math.MaxInt64 {
		t.Fatalf("count after failed increment = %d, error = %v", count, err)
	}
}

func TestCorruptDatabaseIsNotReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "views.sqlite3")
	contents := []byte("This is not a SQLite database.")
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
	if store, err := openStore(path, 42); err == nil {
		if err := store.close(); err != nil {
			t.Error(err)
		}
		t.Fatal("corrupt database was accepted")
	}
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(current) != string(contents) {
		t.Fatal("corrupt database was overwritten")
	}
}
