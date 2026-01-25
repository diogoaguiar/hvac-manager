package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

//go:embed schema.sql
var schemaSQL string

const (
	// CurrentSchemaVersion tracks the database schema version
	CurrentSchemaVersion = 1
)

// DB wraps the SQL database connection with application-specific methods
type DB struct {
	conn *sql.DB
}

// New creates a new database connection WITHOUT initializing schema
// filePath can be ":memory:" for in-memory database or a file path
// Use InitSchema() to create tables, or Migrate() for schema updates
func New(filePath string) (*DB, error) {
	conn, err := sql.Open("sqlite", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys (not enabled by default in SQLite)
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db := &DB{conn: conn}
	return db, nil
}

// InitSchema creates the database tables from scratch
// Returns error if tables already exist
// Use Migrate() for updating existing databases
func (db *DB) InitSchema(ctx context.Context) error {
	// Check if database is already initialized
	var tableName string
	err := db.conn.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='models' LIMIT 1").Scan(&tableName)
	if err == nil {
		return fmt.Errorf("database already initialized (models table exists)")
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing tables: %w", err)
	}

	// Execute schema
	_, err = db.conn.ExecContext(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Set schema version
	return db.setSchemaVersion(ctx, CurrentSchemaVersion)
}

// Migrate updates the database schema to the current version
// Safe to call on already-initialized databases
func (db *DB) Migrate(ctx context.Context) error {
	currentVersion, err := db.GetSchemaVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to check schema version: %w", err)
	}

	// Version 0 means uninitialized database
	if currentVersion == 0 {
		return db.InitSchema(ctx)
	}

	if currentVersion == CurrentSchemaVersion {
		// Already up to date
		return nil
	}

	// Future migrations would go here
	// Example:
	// if currentVersion == 1 {
	//     if err := db.migrateV1ToV2(ctx); err != nil {
	//         return err
	//     }
	//     currentVersion = 2
	// }

	return fmt.Errorf("unknown schema version %d (expected %d)", currentVersion, CurrentSchemaVersion)
}

// GetSchemaVersion retrieves the current schema version
func (db *DB) GetSchemaVersion(ctx context.Context) (int, error) {
	var version int
	err := db.conn.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get schema version: %w", err)
	}
	return version, nil
}

// setSchemaVersion sets the schema version
func (db *DB) setSchemaVersion(ctx context.Context, version int) error {
	_, err := db.conn.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version))
	if err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Model represents an AC model's metadata
type Model struct {
	ID                  int      // Auto-generated primary key
	ModelID             string   // e.g., "1109"
	Manufacturer        string   // e.g., "Daikin"
	SupportedModels     []string // e.g., ["BRC4C158"]
	CommandsEncoding    string   // e.g., "Raw"
	SupportedController string   // e.g., "MQTT"
	MinTemperature      int      // e.g., 16
	MaxTemperature      int      // e.g., 32
	Precision           float64  // e.g., 1.0
	OperationModes      []string // e.g., ["cool", "heat", "fan_only", "dry"]
	FanModes            []string // e.g., ["low", "medium", "high"]
}

// IRCode represents a single IR code entry
type IRCode struct {
	ID          int     // Auto-generated primary key
	ModelID     string  // References Model.ModelID
	Mode        string  // e.g., "cool", "heat", "off"
	Temperature *int    // Pointer to handle NULL for "off" command
	FanSpeed    *string // Pointer to handle NULL for "off" command
	IRCode      string  // Base64-encoded Tuya format code
}

// LookupCode retrieves the IR code for a specific state
// Returns sql.ErrNoRows if no matching code is found
func (db *DB) LookupCode(ctx context.Context, modelID, mode string, temperature int, fanSpeed string) (string, error) {
	var code string
	query := `
		SELECT ir_code 
		FROM ir_codes 
		WHERE model_id = ? AND mode = ? AND temperature = ? AND fan_speed = ?
	`
	err := db.conn.QueryRowContext(ctx, query, modelID, mode, temperature, fanSpeed).Scan(&code)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no IR code found for model=%s mode=%s temp=%d fan=%s",
				modelID, mode, temperature, fanSpeed)
		}
		return "", fmt.Errorf("database query failed: %w", err)
	}
	return code, nil
}

// LookupOffCode retrieves the "off" command IR code
// Returns sql.ErrNoRows if no off code is found
func (db *DB) LookupOffCode(ctx context.Context, modelID string) (string, error) {
	var code string
	query := `
		SELECT ir_code 
		FROM ir_codes 
		WHERE model_id = ? AND mode = 'off'
	`
	err := db.conn.QueryRowContext(ctx, query, modelID).Scan(&code)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no off code found for model=%s", modelID)
		}
		return "", fmt.Errorf("database query failed: %w", err)
	}
	return code, nil
}

// GetModel retrieves model metadata
func (db *DB) GetModel(ctx context.Context, modelID string) (*Model, error) {
	// Note: This is a simplified version.
	// In a real implementation, we'd need to deserialize JSON arrays for
	// SupportedModels, OperationModes, and FanModes from TEXT columns.
	// For now, returning a basic structure for the core functionality.
	var model Model
	query := `
		SELECT id, model_id, manufacturer, min_temperature, max_temperature, precision
		FROM models 
		WHERE model_id = ?
	`
	err := db.conn.QueryRowContext(ctx, query, modelID).Scan(
		&model.ID,
		&model.ModelID,
		&model.Manufacturer,
		&model.MinTemperature,
		&model.MaxTemperature,
		&model.Precision,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model %s not found", modelID)
		}
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	return &model, nil
}

// ListModels returns all available models
func (db *DB) ListModels(ctx context.Context) ([]string, error) {
	query := `SELECT model_id FROM models ORDER BY model_id`
	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query models: %w", err)
	}
	defer rows.Close()

	var models []string
	for rows.Next() {
		var modelID string
		if err := rows.Scan(&modelID); err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}
		models = append(models, modelID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating models: %w", err)
	}

	return models, nil
}

// Ping verifies the database connection is alive
func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}
