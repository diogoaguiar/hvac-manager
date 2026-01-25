package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}
	// Initialize schema for tests
	ctx := context.Background()
	if err := db.InitSchema(ctx); err != nil {
		t.Fatalf("failed to initialize schema: %v", err)
	}
	return db
}

func TestNew(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	if err := db.Ping(context.Background()); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestMigrate(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// First call should initialize schema
	err = db.Migrate(ctx)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Verify version
	version, err := db.GetSchemaVersion(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if version != CurrentSchemaVersion {
		t.Errorf("expected version %d, got %d", CurrentSchemaVersion, version)
	}

	// Second call should be idempotent
	err = db.Migrate(ctx)
	if err != nil {
		t.Errorf("Migrate not idempotent: %v", err)
	}
}

func TestLoadAndQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	testFile := filepath.Join("..", "..", "docs", "smartir", "reference", "1109_tuya.json")

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	// Load data
	if err := db.LoadFromJSON(ctx, "1109", testFile); err != nil {
		t.Fatalf("LoadFromJSON failed: %v", err)
	}

	// Query code
	code, err := db.LookupCode(ctx, "1109", "cool", 21, "low")
	if err != nil {
		t.Fatalf("LookupCode failed: %v", err)
	}
	if code == "" {
		t.Error("expected non-empty IR code")
	}

	// Query off code
	offCode, err := db.LookupOffCode(ctx, "1109")
	if err != nil {
		t.Fatalf("LookupOffCode failed: %v", err)
	}
	if offCode == "" {
		t.Error("expected non-empty off code")
	}
}
