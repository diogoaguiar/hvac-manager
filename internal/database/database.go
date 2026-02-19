package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/diogoaguiar/hvac-manager/internal/logger"
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

// LookupCode retrieves the IR code for a specific state with intelligent fallback
// Priority order:
// 1. Exact match (mode + temp + fan)
// 2. Mode + temp (ignore fan) - for heat/cool/auto modes
// 3. Fan fallback: auto → low → medium → high
// 4. Mode only (ignore temp + fan) - for fan_only/dry modes
func (db *DB) LookupCode(ctx context.Context, modelID, mode string, temperature int, fanSpeed string) (string, error) {
	logger.Debug("DB LookupCode: model=%s mode=%s temp=%d fan=%s", modelID, mode, temperature, fanSpeed)

	// Try exact match first
	code, err := db.lookupExact(ctx, modelID, mode, temperature, fanSpeed)
	if err == nil {
		logger.Info("✓ Exact match: mode=%s temp=%d fan=%s", mode, temperature, fanSpeed)
		return code, nil
	}
	if err != sql.ErrNoRows {
		return "", err // Database error, not just missing data
	}

	logger.Debug("Exact match failed, trying fallback strategies...")

	// Determine if temperature is required for this mode
	tempRequired := mode == "heat" || mode == "cool" || mode == "auto"

	// Strategy 1: Try different fan speeds (if fan was specified)
	if fanSpeed != "" {
		fanFallbacks := getFanFallbacks(fanSpeed)
		for _, fallbackFan := range fanFallbacks {
			code, err := db.lookupExact(ctx, modelID, mode, temperature, fallbackFan)
			if err == nil {
				logger.Info("✓ Fan fallback: mode=%s temp=%d fan=%s (requested: %s)",
					mode, temperature, fallbackFan, fanSpeed)
				return code, nil
			}
		}
	}

	// Strategy 2: Try ignoring fan speed (mode + temp only)
	if tempRequired {
		code, err := db.lookupModeTemp(ctx, modelID, mode, temperature)
		if err == nil {
			logger.Info("✓ Mode+temp match: mode=%s temp=%d (ignoring fan)", mode, temperature)
			return code, nil
		}
	}

	// Strategy 3: Try mode only (ignore temp + fan) for modes that don't require temp
	if !tempRequired {
		code, err := db.lookupModeOnly(ctx, modelID, mode)
		if err == nil {
			logger.Info("✓ Mode-only match: mode=%s (ignoring temp/fan)", mode)
			return code, nil
		}
	}

	// All strategies failed
	logger.Warn("⚠️  No IR code found for model=%s mode=%s temp=%d fan=%s (tried all fallbacks)",
		modelID, mode, temperature, fanSpeed)

	// Debug info: show what's available
	var count int
	checkQuery := `SELECT COUNT(*) FROM ir_codes WHERE model_id = ? AND mode = ?`
	db.conn.QueryRowContext(ctx, checkQuery, modelID, mode).Scan(&count)
	logger.Debug("Found %d codes for model=%s mode=%s (any temp/fan)", count, modelID, mode)

	return "", fmt.Errorf("no IR code found for model=%s mode=%s temp=%d fan=%s",
		modelID, mode, temperature, fanSpeed)
}

// lookupExact performs exact match query
func (db *DB) lookupExact(ctx context.Context, modelID, mode string, temperature int, fanSpeed string) (string, error) {
	var code string
	query := `
		SELECT ir_code 
		FROM ir_codes 
		WHERE model_id = ? AND mode = ? AND temperature = ? AND fan_speed = ?
	`
	err := db.conn.QueryRowContext(ctx, query, modelID, mode, temperature, fanSpeed).Scan(&code)
	if err == nil {
		logger.Debug("Found IR code in DB (length: %d bytes)", len(code))
	}
	return code, err
}

// lookupModeTemp tries to find code matching mode + temperature (any fan speed)
func (db *DB) lookupModeTemp(ctx context.Context, modelID, mode string, temperature int) (string, error) {
	var code string
	query := `
		SELECT ir_code 
		FROM ir_codes 
		WHERE model_id = ? AND mode = ? AND temperature = ?
		LIMIT 1
	`
	err := db.conn.QueryRowContext(ctx, query, modelID, mode, temperature).Scan(&code)
	return code, err
}

// lookupModeOnly tries to find code matching mode only (any temp/fan)
func (db *DB) lookupModeOnly(ctx context.Context, modelID, mode string) (string, error) {
	var code string
	query := `
		SELECT ir_code 
		FROM ir_codes 
		WHERE model_id = ? AND mode = ?
		LIMIT 1
	`
	err := db.conn.QueryRowContext(ctx, query, modelID, mode).Scan(&code)
	return code, err
}

// getFanFallbacks returns alternative fan speeds to try
// Order: current speed is removed, then try: low → medium → high → auto
func getFanFallbacks(requestedFan string) []string {
	allFans := []string{"low", "medium", "high", "auto"}
	fallbacks := []string{}

	// Add all fan speeds except the one already tried
	for _, fan := range allFans {
		if fan != requestedFan {
			fallbacks = append(fallbacks, fan)
		}
	}

	return fallbacks
}

// LookupOffCode retrieves the "off" command IR code
// Returns sql.ErrNoRows if no off code is found
func (db *DB) LookupOffCode(ctx context.Context, modelID string) (string, error) {
	logger.Debug("DB LookupOffCode: model=%s", modelID)

	var code string
	query := `
		SELECT ir_code 
		FROM ir_codes 
		WHERE model_id = ? AND mode = 'off'
	`
	err := db.conn.QueryRowContext(ctx, query, modelID).Scan(&code)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("No off code found in DB for model=%s", modelID)
			return "", fmt.Errorf("no off code found for model=%s", modelID)
		}
		logger.Error("Database query failed: %v", err)
		return "", fmt.Errorf("database query failed: %w", err)
	}

	logger.Debug("Found off code in DB (length: %d bytes)", len(code))
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

// InsertCode inserts a single IR code into the database (for testing)
func (db *DB) InsertCode(ctx context.Context, code *IRCode) error {
	query := `
		INSERT INTO ir_codes (model_id, mode, temperature, fan_speed, ir_code)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := db.conn.ExecContext(ctx, query,
		code.ModelID,
		code.Mode,
		code.Temperature,
		code.FanSpeed,
		code.IRCode,
	)
	return err
}
