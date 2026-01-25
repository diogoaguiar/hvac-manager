# Internal Database Package

SQLite-based IR code database for HVAC Manager with automatic Broadlink-to-Tuya conversion.

## Overview

This package provides a comprehensive solution for managing SmartIR IR codes:
- **Auto-conversion**: Automatically converts Broadlink format to Tuya during import
- **Dual format support**: Handles both original SmartIR files and pre-converted Tuya files
- **SQLite storage**: Fast, reliable IR code storage and retrieval
- **State-based lookup**: Query codes by AC state (mode, temperature, fan speed)

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

// Load IR codes from JSON files (auto-detects format and converts if needed)
err = db.LoadFromDirectory(ctx, "docs/smartir/reference")
err = db.LoadFromJSON(ctx, "1109", "path/to/1109.json") // Broadlink or Tuya

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

## Broadlink to Tuya Conversion

The package includes a complete implementation of the Broadlink-to-Tuya IR code conversion pipeline.

### Why Conversion is Needed

SmartIR stores IR codes in Broadlink's proprietary format, but our ZS06 IR blaster requires Tuya format. The loader automatically handles this conversion during import.

### Conversion Pipeline

The conversion follows these steps:

1. **Decode Broadlink base64** → hex string
2. **Parse durations** from Broadlink format:
   - 8-bit values are standard durations
   - `0x00` prefix indicates 16-bit big-endian extended duration
3. **Convert to microseconds**: `ceil(duration / 32.84)`
4. **Pack as raw bytes**: Little-endian 16-bit unsigned integers
5. **Compress** using Tuya LZ-style algorithm (sliding window, literal/distance blocks)
6. **Encode** result as base64

### Implementation Files

- **[tuya_codec.go](tuya_codec.go)** - Core compression algorithm
  - `parseBroadlinkDurations()` - Extract durations from hex payload
  - `convertToMicroseconds()` - Convert Broadlink ticks to µs
  - `packRawBytes()` - Pack as little-endian uint16 stream
  - `compressTuya()` - LZ-style compression (level 2)
  - `encodeTuyaBase64()` - Final base64 encoding

- **[converter.go](converter.go)** - High-level conversion API
  - `ConvertBroadlinkToTuya()` - Main conversion function
  - `convertSmartIRCommands()` - Recursive command tree conversion

- **[loader.go](loader.go)** - Auto-detecting loader
  - `convertCommandsIfNeeded()` - Format detection and conversion
  - `LoadFromJSON()` - Load with auto-conversion
  - `LoadFromDirectory()` - Batch import with mixed formats

### Algorithm Details

**Tuya LZ Compression (Level 2):**
- **Window size:** 8KB (2^13 bytes)
- **Max match:** 265 bytes
- **Strategy:** Eagerly use best match found (linear search)
- **Tokens:**
  - Literal blocks: 1-32 bytes of raw data
  - Distance blocks: (length, distance) pairs for repeated sequences

**Performance:**
- Typical IR code: ~300-400 bytes compressed
- Conversion time: <1ms per code
- Compression ratio: varies based on signal repetition

### Format Detection

The loader inspects the `commandsEncoding` field:
- `"Base64"` → Broadlink format (converts to Tuya)
- `"Raw"` → Tuya format (no conversion)

After conversion, metadata is updated:
- `commandsEncoding` → `"Raw"`
- `supportedController` → `"MQTT"`

### Validation

The converter includes comprehensive tests:
```bash
# Run conversion tests
go test ./internal/database -run TestConvert -v

# Run all tests including loader integration
make db-test-conversion
```

Tests validate against real SmartIR files to ensure correct conversion.

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

# Load all SmartIR files from docs/smartir/reference/ (auto-converts Broadlink → Tuya)
make db-load

# Import from custom directory
make db-import DIR=/path/to/smartir/files

# Import single model file
make db-import-model FILE=/path/to/model.json

# Test conversion implementation
make db-test-conversion

# Check database status
make db-status

# Reset database (delete and reinitialize)
make db-reset
```

Or use the db tool directly:

```bash
go run -tags dbtools ./tools/db init hvac.db
go run -tags dbtools ./tools/db load hvac.db docs/smartir/reference
go run -tags dbtools ./tools/db load-single hvac.db 1109 /path/to/1109.json
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

