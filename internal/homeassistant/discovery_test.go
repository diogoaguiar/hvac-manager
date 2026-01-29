package homeassistant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewClimateDiscovery(t *testing.T) {
	deviceID := "living_room"
	deviceName := "Living Room AC"

	discovery := NewClimateDiscovery(deviceID, deviceName)

	// Test basic fields
	if discovery.Name != deviceName {
		t.Errorf("Expected name %q, got %q", deviceName, discovery.Name)
	}

	expectedUniqueID := "hvac_manager_living_room"
	if discovery.UniqueID != expectedUniqueID {
		t.Errorf("Expected unique_id %q, got %q", expectedUniqueID, discovery.UniqueID)
	}

	// Test topics
	expectedStateTopic := "homeassistant/climate/living_room/state"
	if discovery.StateTopic != expectedStateTopic {
		t.Errorf("Expected state topic %q, got %q", expectedStateTopic, discovery.StateTopic)
	}

	expectedCmdTopic := "homeassistant/climate/living_room/set"
	if discovery.TemperatureCommandTopic != expectedCmdTopic {
		t.Errorf("Expected temperature command topic %q, got %q", expectedCmdTopic, discovery.TemperatureCommandTopic)
	}
	if discovery.ModeCommandTopic != expectedCmdTopic {
		t.Errorf("Expected mode command topic %q, got %q", expectedCmdTopic, discovery.ModeCommandTopic)
	}
	if discovery.FanModeCommandTopic != expectedCmdTopic {
		t.Errorf("Expected fan mode command topic %q, got %q", expectedCmdTopic, discovery.FanModeCommandTopic)
	}

	expectedAvailTopic := "homeassistant/climate/living_room/availability"
	if discovery.AvailabilityTopic != expectedAvailTopic {
		t.Errorf("Expected availability topic %q, got %q", expectedAvailTopic, discovery.AvailabilityTopic)
	}

	// Test temperature configuration
	if discovery.MinTemp != 16.0 {
		t.Errorf("Expected min temp 16.0, got %.1f", discovery.MinTemp)
	}
	if discovery.MaxTemp != 30.0 {
		t.Errorf("Expected max temp 30.0, got %.1f", discovery.MaxTemp)
	}
	if discovery.TempStep != 1.0 {
		t.Errorf("Expected temp step 1.0, got %.1f", discovery.TempStep)
	}
	if discovery.Precision != 0.1 {
		t.Errorf("Expected precision 0.1, got %.1f", discovery.Precision)
	}
	if discovery.TemperatureUnit != "C" {
		t.Errorf("Expected temperature unit 'C', got %q", discovery.TemperatureUnit)
	}

	// Test modes
	expectedModes := []string{"off", "cool", "heat", "dry", "fan_only", "auto"}
	if len(discovery.Modes) != len(expectedModes) {
		t.Errorf("Expected %d modes, got %d", len(expectedModes), len(discovery.Modes))
	}

	expectedFanModes := []string{"auto", "low", "medium", "high"}
	if len(discovery.FanModes) != len(expectedFanModes) {
		t.Errorf("Expected %d fan modes, got %d", len(expectedFanModes), len(discovery.FanModes))
	}

	// Test device info
	if discovery.Device.Name != deviceName {
		t.Errorf("Expected device name %q, got %q", deviceName, discovery.Device.Name)
	}
	if discovery.Device.Model != "HVAC Manager POC" {
		t.Errorf("Expected device model 'HVAC Manager POC', got %q", discovery.Device.Model)
	}
	if discovery.Device.Manufacturer != "HVAC Manager" {
		t.Errorf("Expected manufacturer 'HVAC Manager', got %q", discovery.Device.Manufacturer)
	}
}

func TestClimateDiscovery_ToJSON(t *testing.T) {
	discovery := NewClimateDiscovery("test_room", "Test AC")

	jsonData, err := discovery.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	// Verify it's valid JSON by unmarshaling it back
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Check that key fields are present
	requiredFields := []string{
		"name", "unique_id", "state_topic", "temperature_command_topic",
		"mode_command_topic", "fan_mode_command_topic", "availability_topic",
		"modes", "fan_modes", "min_temp", "max_temp", "device",
	}

	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Required field %q missing from JSON output", field)
		}
	}

	// Verify JSON is indented (pretty-printed)
	if !strings.Contains(string(jsonData), "\n") {
		t.Error("JSON should be indented with newlines")
	}
}

func TestClimateDiscovery_ConfigTopic(t *testing.T) {
	discovery := NewClimateDiscovery("test_room", "Test AC")

	topic := discovery.ConfigTopic("test_room")
	expected := "homeassistant/climate/test_room/config"

	if topic != expected {
		t.Errorf("ConfigTopic() = %q, want %q", topic, expected)
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		wantTemp    *float64
		wantMode    *string
		wantFanMode *string
		wantErr     bool
	}{
		{
			name:     "Temperature only",
			payload:  `{"temperature": 22.5}`,
			wantTemp: floatPtr(22.5),
		},
		{
			name:     "Mode only",
			payload:  `{"mode": "cool"}`,
			wantMode: strPtr("cool"),
		},
		{
			name:        "Fan mode only",
			payload:     `{"fan_mode": "high"}`,
			wantFanMode: strPtr("high"),
		},
		{
			name:        "All fields",
			payload:     `{"temperature": 21.0, "mode": "heat", "fan_mode": "low"}`,
			wantTemp:    floatPtr(21.0),
			wantMode:    strPtr("heat"),
			wantFanMode: strPtr("low"),
		},
		{
			name:    "Empty JSON",
			payload: `{}`,
		},
		{
			name:    "Invalid JSON",
			payload: `{invalid}`,
			wantErr: true,
		},
		{
			name:    "Not JSON",
			payload: `not json at all`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseCommand([]byte(tt.payload))

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check temperature
			if tt.wantTemp != nil {
				if cmd.Temperature == nil {
					t.Error("Expected temperature to be set, got nil")
				} else if *cmd.Temperature != *tt.wantTemp {
					t.Errorf("Temperature = %.1f, want %.1f", *cmd.Temperature, *tt.wantTemp)
				}
			} else if cmd.Temperature != nil {
				t.Errorf("Expected temperature to be nil, got %.1f", *cmd.Temperature)
			}

			// Check mode
			if tt.wantMode != nil {
				if cmd.Mode == nil {
					t.Error("Expected mode to be set, got nil")
				} else if *cmd.Mode != *tt.wantMode {
					t.Errorf("Mode = %q, want %q", *cmd.Mode, *tt.wantMode)
				}
			} else if cmd.Mode != nil {
				t.Errorf("Expected mode to be nil, got %q", *cmd.Mode)
			}

			// Check fan mode
			if tt.wantFanMode != nil {
				if cmd.FanMode == nil {
					t.Error("Expected fan_mode to be set, got nil")
				} else if *cmd.FanMode != *tt.wantFanMode {
					t.Errorf("FanMode = %q, want %q", *cmd.FanMode, *tt.wantFanMode)
				}
			} else if cmd.FanMode != nil {
				t.Errorf("Expected fan_mode to be nil, got %q", *cmd.FanMode)
			}
		})
	}
}

func TestClimateState_JSON(t *testing.T) {
	state := ClimateState{
		Temperature: 22.5,
		Mode:        "cool",
		FanMode:     "high",
	}

	jsonData, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	// Parse it back
	var parsed ClimateState
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal state: %v", err)
	}

	if parsed.Temperature != state.Temperature {
		t.Errorf("Temperature = %.1f, want %.1f", parsed.Temperature, state.Temperature)
	}
	if parsed.Mode != state.Mode {
		t.Errorf("Mode = %q, want %q", parsed.Mode, state.Mode)
	}
	if parsed.FanMode != state.FanMode {
		t.Errorf("FanMode = %q, want %q", parsed.FanMode, state.FanMode)
	}
}

// Helper functions for creating pointers
func floatPtr(f float64) *float64 {
	return &f
}

func strPtr(s string) *string {
	return &s
}
