package database

import (
	"context"
	"testing"
)

// TestLookupCode_FanFallback tests fan speed fallback logic
func TestLookupCode_FanFallback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert model first (required for foreign key)
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO models (model_id, manufacturer, supported_models, commands_encoding, 
			supported_controller, min_temperature, max_temperature, precision, operation_modes, fan_modes) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-model", "Test", "[]", "Raw", "MQTT", 16, 30, 1.0, "[]", "[]")
	if err != nil {
		t.Fatalf("Failed to insert model: %v", err)
	}

	// Insert test data: only "low" fan speed available
	err = db.InsertCode(ctx, &IRCode{
		ModelID:     "test-model",
		Mode:        "heat",
		Temperature: intPtr(22),
		FanSpeed:    strPtr("low"),
		IRCode:      "test-code-low",
	})
	if err != nil {
		t.Fatalf("Failed to insert test code: %v", err)
	}

	// Test: Request "auto" fan, should fallback to "low"
	code, err := db.LookupCode(ctx, "test-model", "heat", 22, "auto")
	if err != nil {
		t.Fatalf("Expected fallback to succeed, got error: %v", err)
	}
	if code != "test-code-low" {
		t.Errorf("Expected fallback to 'low', got code: %s", code)
	}
}

// TestLookupCode_TemperatureRequiredModes tests that temp is required for heat/cool/auto
func TestLookupCode_TemperatureRequiredModes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert model first
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO models (model_id, manufacturer, supported_models, commands_encoding, 
			supported_controller, min_temperature, max_temperature, precision, operation_modes, fan_modes) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-model", "Test", "[]", "Raw", "MQTT", 16, 30, 1.0, "[]", "[]")
	if err != nil {
		t.Fatalf("Failed to insert model: %v", err)
	}

	// Insert codes for heat mode with different fan speeds
	codes := []struct {
		temp int
		fan  string
		code string
	}{
		{22, "low", "heat-22-low"},
		{22, "medium", "heat-22-med"},
		{23, "low", "heat-23-low"},
	}

	for _, tc := range codes {
		err := db.InsertCode(ctx, &IRCode{
			ModelID:     "test-model",
			Mode:        "heat",
			Temperature: intPtr(tc.temp),
			FanSpeed:    strPtr(tc.fan),
			IRCode:      tc.code,
		})
		if err != nil {
			t.Fatalf("Failed to insert test code: %v", err)
		}
	}

	tests := []struct {
		name        string
		mode        string
		temp        int
		fan         string
		expectCode  string
		expectError bool
	}{
		{
			name:       "Exact match",
			mode:       "heat",
			temp:       22,
			fan:        "low",
			expectCode: "heat-22-low",
		},
		{
			name:       "Fan fallback - auto to low",
			mode:       "heat",
			temp:       22,
			fan:        "auto",
			expectCode: "heat-22-low", // Should fallback to low (first available)
		},
		{
			name:       "Fan fallback - high to low",
			mode:       "heat",
			temp:       22,
			fan:        "high",
			expectCode: "heat-22-low", // Should fallback to low
		},
		{
			name:        "Wrong temp - no fallback for temp-required modes",
			mode:        "heat",
			temp:        25,
			fan:         "low",
			expectError: true, // No temp=25 available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := db.LookupCode(ctx, "test-model", tt.mode, tt.temp, tt.fan)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got code: %s", code)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if code != tt.expectCode {
					t.Errorf("Expected code %s, got %s", tt.expectCode, code)
				}
			}
		})
	}
}

// TestLookupCode_ModeOnlyFallback tests that fan/dry modes can ignore temp
func TestLookupCode_ModeOnlyFallback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert model first
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO models (model_id, manufacturer, supported_models, commands_encoding, 
			supported_controller, min_temperature, max_temperature, precision, operation_modes, fan_modes) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-model", "Test", "[]", "Raw", "MQTT", 16, 30, 1.0, "[]", "[]")
	if err != nil {
		t.Fatalf("Failed to insert model: %v", err)
	}

	// Insert fan_only code with specific temp
	err = db.InsertCode(ctx, &IRCode{
		ModelID:     "test-model",
		Mode:        "fan_only",
		Temperature: intPtr(25),
		FanSpeed:    strPtr("high"),
		IRCode:      "fan-only-code",
	})
	if err != nil {
		t.Fatalf("Failed to insert test code: %v", err)
	}

	// Insert dry code
	err = db.InsertCode(ctx, &IRCode{
		ModelID:     "test-model",
		Mode:        "dry",
		Temperature: intPtr(24),
		FanSpeed:    strPtr("low"),
		IRCode:      "dry-code",
	})
	if err != nil {
		t.Fatalf("Failed to insert test code: %v", err)
	}

	tests := []struct {
		name       string
		mode       string
		temp       int
		fan        string
		expectCode string
	}{
		{
			name:       "fan_only with different temp - should fallback to mode-only",
			mode:       "fan_only",
			temp:       22, // Different from stored temp=25
			fan:        "medium",
			expectCode: "fan-only-code",
		},
		{
			name:       "dry with different temp - should fallback to mode-only",
			mode:       "dry",
			temp:       20, // Different from stored temp=24
			fan:        "auto",
			expectCode: "dry-code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := db.LookupCode(ctx, "test-model", tt.mode, tt.temp, tt.fan)
			if err != nil {
				t.Fatalf("Expected fallback to succeed, got error: %v", err)
			}
			if code != tt.expectCode {
				t.Errorf("Expected code %s, got %s", tt.expectCode, code)
			}
		})
	}
}

// TestLookupCode_CompleteFallbackChain tests the full fallback priority
func TestLookupCode_CompleteFallbackChain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert model first
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO models (model_id, manufacturer, supported_models, commands_encoding, 
			supported_controller, min_temperature, max_temperature, precision, operation_modes, fan_modes) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-model", "Test", "[]", "Raw", "MQTT", 16, 30, 1.0, "[]", "[]")
	if err != nil {
		t.Fatalf("Failed to insert model: %v", err)
	}

	// Only insert mode+temp, no exact match for requested fan
	err = db.InsertCode(ctx, &IRCode{
		ModelID:     "test-model",
		Mode:        "cool",
		Temperature: intPtr(21),
		FanSpeed:    strPtr("high"),
		IRCode:      "cool-21-high",
	})
	if err != nil {
		t.Fatalf("Failed to insert test code: %v", err)
	}

	// Request: cool/21/auto (auto not available)
	// Should fallback: auto → low (fail) → medium (fail) → high (success)
	code, err := db.LookupCode(ctx, "test-model", "cool", 21, "auto")
	if err != nil {
		t.Fatalf("Expected fallback to succeed, got error: %v", err)
	}
	if code != "cool-21-high" {
		t.Errorf("Expected fallback to 'high', got code: %s", code)
	}
}

// TestGetFanFallbacks tests the fan fallback order
func TestGetFanFallbacks(t *testing.T) {
	tests := []struct {
		requested string
		expected  []string
	}{
		{
			requested: "auto",
			expected:  []string{"low", "medium", "high"},
		},
		{
			requested: "low",
			expected:  []string{"medium", "high", "auto"},
		},
		{
			requested: "medium",
			expected:  []string{"low", "high", "auto"},
		},
		{
			requested: "high",
			expected:  []string{"low", "medium", "auto"},
		},
	}

	for _, tt := range tests {
		t.Run("Requested_"+tt.requested, func(t *testing.T) {
			result := getFanFallbacks(tt.requested)
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d fallbacks, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Position %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
