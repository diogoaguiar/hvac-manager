package state

import (
	"testing"
)

func TestNewACState(t *testing.T) {
	s := NewACState()

	if s.Temperature != 22.0 {
		t.Errorf("Expected default temperature 22.0, got %.1f", s.Temperature)
	}
	if s.Mode != "off" {
		t.Errorf("Expected default mode 'off', got '%s'", s.Mode)
	}
	if s.FanMode != "auto" {
		t.Errorf("Expected default fan mode 'auto', got '%s'", s.FanMode)
	}
	if s.Power != false {
		t.Errorf("Expected default power false, got %v", s.Power)
	}
	if s.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set")
	}
}

func TestSetTemperature(t *testing.T) {
	tests := []struct {
		name    string
		temp    float64
		wantErr bool
	}{
		{"Min boundary", 16.0, false},
		{"Max boundary", 30.0, false},
		{"Middle value", 22.5, false},
		{"Common value", 21.0, false},
		{"Below min", 15.9, true},
		{"Above max", 30.1, true},
		{"Way too low", 0.0, true},
		{"Way too high", 50.0, true},
		{"Negative", -5.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewACState()
			oldTime := s.LastUpdated

			err := s.SetTemperature(tt.temp)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error for temperature %.1f, got nil", tt.temp)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for temperature %.1f: %v", tt.temp, err)
			}
			if !tt.wantErr {
				if s.Temperature != tt.temp {
					t.Errorf("Temperature not set: expected %.1f, got %.1f", tt.temp, s.Temperature)
				}
				if !s.LastUpdated.After(oldTime) {
					t.Error("LastUpdated should be updated")
				}
			}
		})
	}
}

func TestSetMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		wantPower bool
		wantErr   bool
	}{
		{"Cool mode", "cool", true, false},
		{"Heat mode", "heat", true, false},
		{"Dry mode", "dry", true, false},
		{"Fan only mode", "fan_only", true, false},
		{"Auto mode", "auto", true, false},
		{"Off mode", "off", false, false},
		{"Invalid mode", "turbo", false, true},
		{"Empty mode", "", false, true},
		{"Random string", "invalid", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewACState()
			oldTime := s.LastUpdated

			err := s.SetMode(tt.mode)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error for mode '%s', got nil", tt.mode)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for mode '%s': %v", tt.mode, err)
			}
			if !tt.wantErr {
				if s.Mode != tt.mode {
					t.Errorf("Mode not set: expected '%s', got '%s'", tt.mode, s.Mode)
				}
				if s.Power != tt.wantPower {
					t.Errorf("Power state mismatch: expected %v, got %v", tt.wantPower, s.Power)
				}
				if !s.LastUpdated.After(oldTime) {
					t.Error("LastUpdated should be updated")
				}
			}
		})
	}
}

func TestSetFanMode(t *testing.T) {
	tests := []struct {
		name    string
		fanMode string
		wantErr bool
	}{
		{"Auto fan", "auto", false},
		{"Low fan", "low", false},
		{"Medium fan", "medium", false},
		{"High fan", "high", false},
		{"Invalid fan mode", "turbo", true},
		{"Empty fan mode", "", true},
		{"Random string", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewACState()
			oldTime := s.LastUpdated

			err := s.SetFanMode(tt.fanMode)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error for fan mode '%s', got nil", tt.fanMode)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for fan mode '%s': %v", tt.fanMode, err)
			}
			if !tt.wantErr {
				if s.FanMode != tt.fanMode {
					t.Errorf("Fan mode not set: expected '%s', got '%s'", tt.fanMode, s.FanMode)
				}
				if !s.LastUpdated.After(oldTime) {
					t.Error("LastUpdated should be updated")
				}
			}
		})
	}
}

func TestACState_String(t *testing.T) {
	tests := []struct {
		name     string
		state    *ACState
		expected string
	}{
		{
			name: "Default state",
			state: &ACState{
				Mode:        "off",
				Temperature: 22.0,
				FanMode:     "auto",
				Power:       false,
			},
			expected: "Mode: off, Temp: 22.0°C, Fan: auto, Power: false",
		},
		{
			name: "Cool mode running",
			state: &ACState{
				Mode:        "cool",
				Temperature: 21.5,
				FanMode:     "high",
				Power:       true,
			},
			expected: "Mode: cool, Temp: 21.5°C, Fan: high, Power: true",
		},
		{
			name: "Heat mode low fan",
			state: &ACState{
				Mode:        "heat",
				Temperature: 25.0,
				FanMode:     "low",
				Power:       true,
			},
			expected: "Mode: heat, Temp: 25.0°C, Fan: low, Power: true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPowerStateCorrelation(t *testing.T) {
	s := NewACState()

	// Initially off
	if s.Power != false {
		t.Error("New state should have Power=false")
	}

	// Turn on by setting mode to cool
	if err := s.SetMode("cool"); err != nil {
		t.Fatalf("SetMode failed: %v", err)
	}
	if !s.Power {
		t.Error("Power should be true when mode is 'cool'")
	}

	// Turn off
	if err := s.SetMode("off"); err != nil {
		t.Fatalf("SetMode failed: %v", err)
	}
	if s.Power {
		t.Error("Power should be false when mode is 'off'")
	}

	// Turn on again with different mode
	if err := s.SetMode("heat"); err != nil {
		t.Fatalf("SetMode failed: %v", err)
	}
	if !s.Power {
		t.Error("Power should be true when mode is 'heat'")
	}
}
