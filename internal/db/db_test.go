package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPackageCompiles(t *testing.T) {
	t.Log("db package compiles")
}

func TestUniquePortfolioName(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		_ = database.Close()
	}()

	// Insert first portfolio
	_, err = database.ExecContext(ctx, "INSERT INTO portfolios (name, created_at) VALUES (?, ?)", "Test Portfolio", 1234567890)
	if err != nil {
		t.Fatalf("failed to insert first portfolio: %v", err)
	}

	// Try to insert second portfolio with same name - should fail due to UNIQUE constraint
	_, err = database.ExecContext(ctx, "INSERT INTO portfolios (name, created_at) VALUES (?, ?)", "Test Portfolio", 1234567891)
	if err == nil {
		t.Error("expected error when inserting portfolio with duplicate name, got nil")
	}
}
