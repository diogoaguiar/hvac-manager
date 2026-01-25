# Internal Database Package

SQLite-based IR code database for HVAC Manager.

## Overview

This package provides a simple SQLite abstraction for storing and querying IR codes from the SmartIR database. It handles:
- Loading SmartIR JSON files (Tuya format)
- Storing model metadata and IR codes
- Fast lookups by AC state (mode, temperature, fan speed)

## Usage

```go
import "github.com/diogoaguiar/hvac-manager/internal/database"

// Create database connection (doesn't initialize schema)
db, err := database.New("hvac.db")  // or ":memory:" for testing
defer db.Close()

// Option 1: Use Migrate() - recommended for production
// Handles both initial setup and future schema updates
err = db.Migrate(ctx)

// Option 2: Use InitSchema() - explicit initialization
// Returns error if database is already initialized
err = db.InitSchema(ctx)

// Load IR codes from JSON files
err = db.LoadFromDirectory(ctx, "docs/smartir/reference")

// Query IR codes
code, err := db.LookupCode(ctx, "1109", "cool", 21, "low")
offCode, err := db.LookupOffCode(ctx, "1109")

// Get model information
model, err := db.GetModel(ctx, "1109")
models, err := db.ListModels(ctx)
```

## Schema Management

The database separates connection creation from schema initialization:

### Database Connection
`New(filePath)` - Creates connection only, no schema changes

### Schema Initialization
`InitSchema(ctx)` - Creates tables from scratch (fails if already exists)

### Migrations
`Migrate(ctx)` - Smart migration that:
- Initializes schema if database is empty (version 0)
- No-op if schema is current version
- Runs migration steps for older versions (future)

### Version Tracking
Schema version is stored using SQLite's `PRAGMA user_version`:
- Version 0 = uninitialized database
- Version 1 = current schema (Phase 2)

## Schema

### `models` table
Stores AC model metadata:
- `model_id`: e.g., "1109"
- `manufacturer`: e.g., "Daikin"
- `min_temperature`, `max_temperature`: Temperature range
- `operation_modes`: JSON array of supported modes
- `fan_modes`: JSON array of supported fan speeds

### `ir_codes` table
Stores IR codes for each state:
- `model_id`: References models table
- `mode`: "cool", "heat", "fan_only", "dry", "off"
- `temperature`: Integer (NULL for "off")
- `fan_speed`: "low", "medium", "high" (NULL for "off")
- `ir_code`: Base64-encoded Tuya format code

## Testing

```bash
go test ./internal/database -v
```

Tests use in-memory database and real SmartIR JSON files from `docs/smartir/reference/`.

## Demo

See [cmd/demo/main.go](../../cmd/demo/main.go) for a complete working example.

```bash
# Run in-memory demo
make demo

# Or run directly
go run cmd/demo/main.go

# Query specific code
go run cmd/demo/main.go 1109 cool 21 low
```

## Database Management CLI

Use the provided Makefile commands for database management:

```bash
# Initialize database
make db-init

# Load IR codes from SmartIR files
make db-load

# Check database status
make db-status

# Reset database (delete and reinitialize)
make db-reset

# Clean all build artifacts including database
make clean
```

Or use the db tool directly:

```bash
go run -tags dbtools ./tools/db init hvac.db
go run -tags dbtools ./tools/db load hvac.db docs/smartir/reference
go run -tags dbtools ./tools/db status hvac.db
```

## Design Decisions

- **Separated connection from migration**: `New()` only creates connection; `Migrate()` or `InitSchema()` handle schema
- **Version tracking**: Uses SQLite's `PRAGMA user_version` for schema versioning
- **UPSERT support**: `LoadFromJSON()` can be called multiple times to update existing models
- **SQLite over in-memory maps**: Provides query flexibility, proper data types, and familiar SQL interface
- **Pure Go driver** (`modernc.org/sqlite`): No CGO dependencies, easier cross-compilation
- **Embedded schema** (`go:embed`): Schema SQL embedded in binary
- **Context-aware queries**: All database operations accept `context.Context` for cancellation/timeout support
- **Transaction-based loading**: JSON imports are atomic (all-or-nothing)

## Future Enhancements

- [ ] Add indexes for faster range queries
- [ ] Implement caching layer for frequent lookups
- [ ] Support for fuzzy matching (e.g., find nearest temperature)
- [ ] Database migrations framework
- [ ] Export/import database to file

