package state

import (
	"fmt"
	"time"
)

// ACState represents the current state of the air conditioner
type ACState struct {
	Temperature float64   `json:"temperature"`  // Temperature in Celsius
	Mode        string    `json:"mode"`         // off, cool, heat, dry, fan_only, auto
	FanMode     string    `json:"fan_mode"`     // auto, low, medium, high
	Power       bool      `json:"power"`        // true = on, false = off
	LastUpdated time.Time `json:"last_updated"` // Timestamp of last state change
}

// Valid modes for the AC
var ValidModes = []string{"off", "cool", "heat", "dry", "fan_only", "auto"}

// Valid fan modes
var ValidFanModes = []string{"auto", "low", "medium", "high"}

// NewACState creates a new AC state with default values
func NewACState() *ACState {
	return &ACState{
		Temperature: 22.0,
		Mode:        "off",
		FanMode:     "auto",
		Power:       false,
		LastUpdated: time.Now(),
	}
}

// SetTemperature updates the temperature and validates the range
func (s *ACState) SetTemperature(temp float64) error {
	// Typical AC range: 16-30°C
	if temp < 16.0 || temp > 30.0 {
		return fmt.Errorf("temperature %.1f out of range (16-30°C)", temp)
	}
	s.Temperature = temp
	s.LastUpdated = time.Now()
	return nil
}

// SetMode updates the mode after validation
func (s *ACState) SetMode(mode string) error {
	if !isValidMode(mode) {
		return fmt.Errorf("invalid mode: %s (valid: %v)", mode, ValidModes)
	}
	s.Mode = mode
	s.Power = mode != "off"
	s.LastUpdated = time.Now()
	return nil
}

// SetFanMode updates the fan mode after validation
func (s *ACState) SetFanMode(fanMode string) error {
	if !isValidFanMode(fanMode) {
		return fmt.Errorf("invalid fan mode: %s (valid: %v)", fanMode, ValidFanModes)
	}
	s.FanMode = fanMode
	s.LastUpdated = time.Now()
	return nil
}

// isValidMode checks if the mode is in the valid list
func isValidMode(mode string) bool {
	for _, valid := range ValidModes {
		if mode == valid {
			return true
		}
	}
	return false
}

// isValidFanMode checks if the fan mode is in the valid list
func isValidFanMode(fanMode string) bool {
	for _, valid := range ValidFanModes {
		if fanMode == valid {
			return true
		}
	}
	return false
}

// String returns a human-readable representation of the state
func (s *ACState) String() string {
	return fmt.Sprintf("Mode: %s, Temp: %.1f°C, Fan: %s, Power: %v",
		s.Mode, s.Temperature, s.FanMode, s.Power)
}
