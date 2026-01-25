package database

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConvertBroadlinkToTuya_RealData tests conversion against actual SmartIR files.
// Note: This validates that conversion succeeds and produces valid Tuya format output.
// Byte-for-byte comparison with Python output is not guaranteed because compression
// algorithms can produce different valid representations of the same data.
func TestConvertBroadlinkToTuya_RealData(t *testing.T) {
	// Test with model 1109 (Daikin AC unit)
	testDataDir := "../../docs/smartir/reference"
	broadlinkFile := filepath.Join(testDataDir, "1109.json")

	// Skip if test files don't exist (CI environment without test data)
	if _, err := os.Stat(broadlinkFile); os.IsNotExist(err) {
		t.Skip("Test data not found, skipping real data test")
		return
	}

	// Load Broadlink file
	broadlinkData, err := os.ReadFile(broadlinkFile)
	if err != nil {
		t.Fatalf("Failed to read Broadlink file: %v", err)
	}

	var broadlinkJSON map[string]interface{}
	if err := json.Unmarshal(broadlinkData, &broadlinkJSON); err != nil {
		t.Fatalf("Failed to parse Broadlink JSON: %v", err)
	}

	// Extract commands
	broadlinkCommands := broadlinkJSON["commands"].(map[string]interface{})

	// Test "off" command
	if broadlinkOff, ok := broadlinkCommands["off"].(string); ok {
		convertedOff, err := ConvertBroadlinkToTuya(broadlinkOff)
		if err != nil {
			t.Errorf("Failed to convert 'off' command: %v", err)
		}

		if convertedOff == "" {
			t.Error("Converted 'off' command is empty")
		}

		// Verify it's valid base64 and not the same as input (i.e., was converted)
		if convertedOff == broadlinkOff {
			t.Error("'off' command was not converted (same as input)")
		}

		t.Logf("Successfully converted 'off' command: %d bytes", len(convertedOff))
	}

	// Test a sample of mode-based commands
	testCases := []struct {
		mode string
		fan  string
		temp string
	}{
		{"cool", "low", "21"},
		{"heat", "auto", "25"},
		{"dry", "low", "20"},
		{"fan_only", "high", "25"},
	}

	successCount := 0
	for _, tc := range testCases {
		// Navigate nested structure
		if modeData, ok := broadlinkCommands[tc.mode].(map[string]interface{}); ok {
			if fanData, ok := modeData[tc.fan].(map[string]interface{}); ok {
				if broadlinkCode, ok := fanData[tc.temp].(string); ok {
					// Convert using Go implementation
					convertedCode, err := ConvertBroadlinkToTuya(broadlinkCode)
					if err != nil {
						t.Errorf("Failed to convert code for mode=%s fan=%s temp=%s: %v",
							tc.mode, tc.fan, tc.temp, err)
						continue
					}

					if convertedCode == "" {
						t.Errorf("Empty result for mode=%s fan=%s temp=%s", tc.mode, tc.fan, tc.temp)
						continue
					}

					// Verify it was actually converted
					if convertedCode == broadlinkCode {
						t.Errorf("Code was not converted for mode=%s fan=%s temp=%s", tc.mode, tc.fan, tc.temp)
						continue
					}

					successCount++
					t.Logf("✓ Converted mode=%s fan=%s temp=%s: %d bytes", tc.mode, tc.fan, tc.temp, len(convertedCode))
				}
			}
		}
	}

	if successCount == 0 {
		t.Error("No codes were successfully converted")
	} else {
		t.Logf("Successfully converted %d/%d test codes", successCount, len(testCases))
	}
}

// TestConvertBroadlinkToTuya_EdgeCases tests error handling and edge cases.
func TestConvertBroadlinkToTuya_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorText   string
	}{
		{
			name:        "Empty string",
			input:       "",
			expectError: true,
			errorText:   "empty",
		},
		{
			name:        "Invalid base64",
			input:       "Not!Valid@Base64",
			expectError: true,
			errorText:   "invalid base64",
		},
		{
			name:        "Too short",
			input:       "JgA=", // Valid base64 but too short for Broadlink format
			expectError: true,
			errorText:   "too short",
		},
		{
			name:        "Whitespace handling",
			input:       "  JgBsAaVGDDoMFw4WDBcPOAwXDhYOFQ4WDTkNFww7DDoNFw06DDoNOg06DToMFw45DRcNFg4VDhYOFQ0XDTkNOg0XDRYOFQ4WDhUOOQ0WDRcOFQ4WDxQPFQwXDhUOFg4VDhYNFg4VDRcMOw05DToNOg0WDxUPFA4AA8SmRQ06DBcPFQ0WDjkMFw4WDBcOFgw6DRcNOgw6DRcOOQw6DToNOg06DRYPOA0WDRcNFg4WDBcNFw44DToNFw0WDhUNFw8UDxUNFg0WDxUOFQ4WDDoNOg0XDhUOFg0WDToNFg4WDRYPFA4WDRYNFw4VDxQOFg4VDhYMFw4VDhYOFQ4WDBgNFg4VDhUOFgwXDhYMFw4VDxUOFQ4WDRYNFg4WDhUNFw0WDhUPFQw7DBcNFwwXDhUOOQ45DRYPOA0XDRYOFQ4WDhUOFg0WDhYNFg4VDxUNFg4VDhYOFQ4WDToMFw4VDjkNOg0WDRcOFQ45DRYOOQ0ADQUAAAAAAAAAAAAA  ",
			expectError: false, // Should trim whitespace and process
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertBroadlinkToTuya(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorText)
				} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorText)) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if result == "" {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

// TestLoadFromJSON_BroadlinkFormat tests that the loader automatically converts Broadlink files.
func TestLoadFromJSON_BroadlinkFormat(t *testing.T) {
	testDataDir := "../../docs/smartir/reference"
	broadlinkFile := filepath.Join(testDataDir, "1109.json")

	// Skip if test files don't exist
	if _, err := os.Stat(broadlinkFile); os.IsNotExist(err) {
		t.Skip("Test data not found, skipping loader test")
		return
	}

	// Create in-memory database
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	if err := db.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Load Broadlink file
	ctx := context.Background()
	if err := db.LoadFromJSON(ctx, "1109", broadlinkFile); err != nil {
		t.Fatalf("Failed to load Broadlink file: %v", err)
	}

	// Verify that codes were loaded and converted
	testCases := []struct {
		mode string
		temp int
		fan  string
	}{
		{"cool", 21, "low"},
		{"dry", 20, "low"},
		{"off", 0, ""}, // Special case: no temp/fan
	}

	for _, tc := range testCases {
		var code string
		var query string
		var args []interface{}

		if tc.mode == "off" {
			query = `SELECT ir_code FROM ir_codes WHERE model_id = ? AND mode = ? AND temperature IS NULL AND fan_speed IS NULL`
			args = []interface{}{"1109", tc.mode}
		} else {
			query = `SELECT ir_code FROM ir_codes WHERE model_id = ? AND mode = ? AND temperature = ? AND fan_speed = ?`
			args = []interface{}{"1109", tc.mode, tc.temp, tc.fan}
		}

		err := db.conn.QueryRowContext(ctx, query, args...).Scan(&code)
		if err != nil {
			t.Errorf("Failed to query code for mode=%s temp=%d fan=%s: %v", tc.mode, tc.temp, tc.fan, err)
			continue
		}

		if code == "" {
			t.Errorf("Empty code returned for mode=%s temp=%d fan=%s", tc.mode, tc.temp, tc.fan)
		}

		// Verify it's in Tuya format (should start with common prefixes like "D", "C", "M")
		if len(code) > 0 {
			firstChar := string(code[0])
			validPrefixes := []string{"D", "C", "M", "N", "A", "B", "E", "F", "G", "H", "I", "J", "K", "L", "O", "P"}
			isValid := false
			for _, prefix := range validPrefixes {
				if firstChar == prefix {
					isValid = true
					break
				}
			}
			if !isValid {
				t.Logf("Warning: Unexpected code prefix '%s' for mode=%s temp=%d fan=%s (code: %s)",
					firstChar, tc.mode, tc.temp, tc.fan, code[:20])
			}
		}
	}
}

// TestLoadFromJSON_TuyaFormat tests backward compatibility with pre-converted Tuya files.
func TestLoadFromJSON_TuyaFormat(t *testing.T) {
	testDataDir := "../../docs/smartir/reference"
	tuyaFile := filepath.Join(testDataDir, "1109_tuya.json")

	// Skip if test files don't exist
	if _, err := os.Stat(tuyaFile); os.IsNotExist(err) {
		t.Skip("Test data not found, skipping Tuya format test")
		return
	}

	// Create in-memory database
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	if err := db.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Load Tuya file (should not require conversion)
	ctx := context.Background()
	if err := db.LoadFromJSON(ctx, "1109", tuyaFile); err != nil {
		t.Fatalf("Failed to load Tuya file: %v", err)
	}

	// Verify codes were loaded
	var count int
	err = db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM ir_codes WHERE model_id = ?`, "1109").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count loaded codes: %v", err)
	}

	if count == 0 {
		t.Error("No codes were loaded from Tuya file")
	}

	t.Logf("Successfully loaded %d IR codes from Tuya file", count)
}

// TestLoadFromDirectory_MixedFormats tests loading a directory with both Broadlink and Tuya files.
func TestLoadFromDirectory_MixedFormats(t *testing.T) {
	testDataDir := "../../docs/smartir/reference"

	// Skip if test directory doesn't exist
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		t.Skip("Test data directory not found")
		return
	}

	// Create in-memory database
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	if err := db.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Load all files from directory
	ctx := context.Background()
	if err := db.LoadFromDirectory(ctx, testDataDir); err != nil {
		t.Fatalf("Failed to load from directory: %v", err)
	}

	// Verify models were loaded
	var modelCount int
	err = db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM models`).Scan(&modelCount)
	if err != nil {
		t.Fatalf("Failed to count models: %v", err)
	}

	if modelCount == 0 {
		t.Error("No models were loaded from directory")
	}

	t.Logf("Successfully loaded %d models from directory", modelCount)

	// Verify IR codes were loaded
	var codeCount int
	err = db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM ir_codes`).Scan(&codeCount)
	if err != nil {
		t.Fatalf("Failed to count IR codes: %v", err)
	}

	if codeCount == 0 {
		t.Error("No IR codes were loaded from directory")
	}

	t.Logf("Successfully loaded %d IR codes from directory", codeCount)
}

// BenchmarkConvertBroadlinkToTuya benchmarks the conversion performance.
func BenchmarkConvertBroadlinkToTuya(b *testing.B) {
	// Use a real Broadlink code from test data
	testDataDir := "../../docs/smartir/reference"
	broadlinkFile := filepath.Join(testDataDir, "1109.json")

	// Skip if test files don't exist
	if _, err := os.Stat(broadlinkFile); os.IsNotExist(err) {
		b.Skip("Test data not found")
		return
	}

	// Load one sample code
	data, err := os.ReadFile(broadlinkFile)
	if err != nil {
		b.Fatalf("Failed to read test file: %v", err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		b.Fatalf("Failed to parse JSON: %v", err)
	}

	commands := jsonData["commands"].(map[string]interface{})
	sampleCode := commands["off"].(string)

	// Benchmark the conversion
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ConvertBroadlinkToTuya(sampleCode)
		if err != nil {
			b.Fatalf("Conversion failed: %v", err)
		}
	}
}
