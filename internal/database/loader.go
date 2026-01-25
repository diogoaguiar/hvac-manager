package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SmartIRFile represents the structure of a SmartIR JSON file
type SmartIRFile struct {
	Manufacturer        string          `json:"manufacturer"`
	SupportedModels     []string        `json:"supportedModels"`
	CommandsEncoding    string          `json:"commandsEncoding"`
	SupportedController string          `json:"supportedController"`
	MinTemperature      int             `json:"minTemperature"`
	MaxTemperature      int             `json:"maxTemperature"`
	Precision           float64         `json:"precision"`
	OperationModes      []string        `json:"operationModes"`
	FanModes            []string        `json:"fanModes"`
	Commands            SmartIRCommands `json:"commands"`
}

// SmartIRCommands represents the nested command structure
// Can contain either a direct "off" string or nested mode → fan → temp → code
type SmartIRCommands struct {
	Off   string                                  `json:"off,omitempty"`
	Modes map[string]map[string]map[string]string `json:"-"` // Populated during custom unmarshal
}

// UnmarshalJSON custom unmarshaler to handle the complex nested structure
func (c *SmartIRCommands) UnmarshalJSON(data []byte) error {
	// First, try to unmarshal into a map to inspect structure
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.Modes = make(map[string]map[string]map[string]string)

	for key, value := range raw {
		// "off" is a direct string
		if key == "off" {
			if str, ok := value.(string); ok {
				c.Off = str
			}
			continue
		}

		// Other keys are modes (fan_only, cool, heat, dry)
		mode := key
		c.Modes[mode] = make(map[string]map[string]string)

		// Value should be a nested object: fan_speed → temperature → code
		modeData, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		for fanSpeed, fanData := range modeData {
			c.Modes[mode][fanSpeed] = make(map[string]string)

			tempData, ok := fanData.(map[string]interface{})
			if !ok {
				continue
			}

			for temp, code := range tempData {
				if codeStr, ok := code.(string); ok {
					c.Modes[mode][fanSpeed][temp] = codeStr
				}
			}
		}
	}

	return nil
}

// LoadFromJSON reads a SmartIR JSON file and populates the database.
// Supports both Broadlink and Tuya formats - automatically detects and converts if needed.
// Can be called multiple times to add additional models.
// Uses UPSERT (ON CONFLICT) to update existing models if called again with same modelID.
func (db *DB) LoadFromJSON(ctx context.Context, modelID, filePath string) error {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse JSON
	var smartIR SmartIRFile
	if err := json.Unmarshal(data, &smartIR); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert Broadlink codes to Tuya if needed
	if err := db.convertCommandsIfNeeded(&smartIR); err != nil {
		return fmt.Errorf("failed to convert IR codes: %w", err)
	}

	// Start transaction for atomic insertion
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Insert model metadata
	if err := db.insertModel(ctx, tx, modelID, &smartIR); err != nil {
		return fmt.Errorf("failed to insert model: %w", err)
	}

	// Insert IR codes
	if err := db.insertIRCodes(ctx, tx, modelID, &smartIR); err != nil {
		return fmt.Errorf("failed to insert IR codes: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// insertModel inserts model metadata into the database
func (db *DB) insertModel(ctx context.Context, tx *sql.Tx, modelID string, smartIR *SmartIRFile) error {
	// Serialize array fields to JSON for storage
	supportedModelsJSON, _ := json.Marshal(smartIR.SupportedModels)
	operationModesJSON, _ := json.Marshal(smartIR.OperationModes)
	fanModesJSON, _ := json.Marshal(smartIR.FanModes)

	query := `
		INSERT INTO models (
			model_id, manufacturer, supported_models, commands_encoding, 
			supported_controller, min_temperature, max_temperature, precision, 
			operation_modes, fan_modes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(model_id) DO UPDATE SET
			manufacturer = excluded.manufacturer,
			supported_models = excluded.supported_models,
			commands_encoding = excluded.commands_encoding,
			supported_controller = excluded.supported_controller,
			min_temperature = excluded.min_temperature,
			max_temperature = excluded.max_temperature,
			precision = excluded.precision,
			operation_modes = excluded.operation_modes,
			fan_modes = excluded.fan_modes
	`

	_, err := tx.ExecContext(ctx, query,
		modelID,
		smartIR.Manufacturer,
		string(supportedModelsJSON),
		smartIR.CommandsEncoding,
		smartIR.SupportedController,
		smartIR.MinTemperature,
		smartIR.MaxTemperature,
		smartIR.Precision,
		string(operationModesJSON),
		string(fanModesJSON),
	)

	return err
}

// insertIRCodes inserts all IR codes from the SmartIR file
func (db *DB) insertIRCodes(ctx context.Context, tx *sql.Tx, modelID string, smartIR *SmartIRFile) error {
	query := `
		INSERT INTO ir_codes (model_id, mode, temperature, fan_speed, ir_code)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(model_id, mode, temperature, fan_speed) DO UPDATE SET
			ir_code = excluded.ir_code
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert "off" command (no temperature or fan speed)
	if smartIR.Commands.Off != "" {
		if _, err := stmt.ExecContext(ctx, modelID, "off", nil, nil, smartIR.Commands.Off); err != nil {
			return fmt.Errorf("failed to insert off command: %w", err)
		}
	}

	// Insert mode-based commands
	for mode, fanSpeeds := range smartIR.Commands.Modes {
		for fanSpeed, temperatures := range fanSpeeds {
			for tempStr, code := range temperatures {
				// Parse temperature string to int
				var temp int
				if _, err := fmt.Sscanf(tempStr, "%d", &temp); err != nil {
					return fmt.Errorf("invalid temperature %s: %w", tempStr, err)
				}

				if _, err := stmt.ExecContext(ctx, modelID, mode, temp, fanSpeed, code); err != nil {
					return fmt.Errorf("failed to insert code for mode=%s temp=%d fan=%s: %w",
						mode, temp, fanSpeed, err)
				}
			}
		}
	}

	return nil
}

// convertCommandsIfNeeded detects the format and converts Broadlink codes to Tuya if necessary.
// Detection is based on the commandsEncoding field:
// - "Base64" = Broadlink format (needs conversion)
// - "Raw" = Tuya format (already converted)
//
// After conversion, updates the metadata fields to reflect Tuya format.
func (db *DB) convertCommandsIfNeeded(smartIR *SmartIRFile) error {
	// Check if conversion is needed
	if smartIR.CommandsEncoding == "Raw" && smartIR.SupportedController == "MQTT" {
		// Already in Tuya format, no conversion needed
		return nil
	}

	if smartIR.CommandsEncoding != "Base64" {
		return fmt.Errorf("unsupported commandsEncoding: %s (expected 'Base64' or 'Raw')",
			smartIR.CommandsEncoding)
	}

	// Convert "off" command if present
	if smartIR.Commands.Off != "" {
		converted, err := ConvertBroadlinkToTuya(smartIR.Commands.Off)
		if err != nil {
			return fmt.Errorf("failed to convert 'off' command: %w", err)
		}
		smartIR.Commands.Off = converted
	}

	// Convert all mode-based commands
	for mode, fanSpeeds := range smartIR.Commands.Modes {
		for fanSpeed, temperatures := range fanSpeeds {
			for tempStr, code := range temperatures {
				converted, err := ConvertBroadlinkToTuya(code)
				if err != nil {
					return fmt.Errorf("failed to convert code for mode=%s fan=%s temp=%s: %w",
						mode, fanSpeed, tempStr, err)
				}
				smartIR.Commands.Modes[mode][fanSpeed][tempStr] = converted
			}
		}
	}

	// Update metadata to reflect Tuya format
	smartIR.CommandsEncoding = "Raw"
	smartIR.SupportedController = "MQTT"

	return nil
}

// LoadFromDirectory loads all SmartIR JSON files from a directory.
// Supports both naming conventions:
// - Modern: "1109.json" (Broadlink format, will be auto-converted)
// - Legacy: "1109_tuya.json" (pre-converted Tuya format)
//
// The model ID is extracted from the filename (part before .json or _tuya.json).
func (db *DB) LoadFromDirectory(ctx context.Context, dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process JSON files
		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}

		// Extract model ID based on naming convention
		var modelID string
		if len(name) > 10 && name[len(name)-10:] == "_tuya.json" {
			// Legacy format: "1109_tuya.json" → "1109"
			modelID = name[:len(name)-10]
		} else {
			// Modern format: "1109.json" → "1109"
			modelID = name[:len(name)-5]
		}

		filePath := filepath.Join(dirPath, name)
		if err := db.LoadFromJSON(ctx, modelID, filePath); err != nil {
			return fmt.Errorf("failed to load %s: %w", name, err)
		}
	}

	return nil
}
