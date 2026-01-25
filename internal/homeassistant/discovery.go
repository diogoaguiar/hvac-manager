package homeassistant

import (
	"encoding/json"
	"fmt"
)

// ClimateDiscovery represents the MQTT Discovery payload for a Climate entity
type ClimateDiscovery struct {
	Name                     string   `json:"name"`
	UniqueID                 string   `json:"unique_id"`
	DeviceClass              string   `json:"device_class,omitempty"`
	StateTopic               string   `json:"state_topic"`
	TemperatureCommandTopic  string   `json:"temperature_command_topic"`
	ModeCommandTopic         string   `json:"mode_command_topic"`
	FanModeCommandTopic      string   `json:"fan_mode_command_topic"`
	TemperatureStateTemplate string   `json:"temperature_state_template"`
	ModeStateTemplate        string   `json:"mode_state_template"`
	FanModeStateTemplate     string   `json:"fan_mode_state_template"`
	AvailabilityTopic        string   `json:"availability_topic"`
	Modes                    []string `json:"modes"`
	FanModes                 []string `json:"fan_modes"`
	MinTemp                  float64  `json:"min_temp"`
	MaxTemp                  float64  `json:"max_temp"`
	TempStep                 float64  `json:"temp_step"`
	TemperatureUnit          string   `json:"temperature_unit"`
	Precision                float64  `json:"precision"`
	Device                   Device   `json:"device"`
}

// Device represents the device information in the discovery payload
type Device struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Model        string   `json:"model"`
	Manufacturer string   `json:"manufacturer"`
	SWVersion    string   `json:"sw_version,omitempty"`
}

// NewClimateDiscovery creates a new MQTT Discovery payload for a climate entity
func NewClimateDiscovery(deviceID string, deviceName string) *ClimateDiscovery {
	cmdTopic := fmt.Sprintf("homeassistant/climate/%s/set", deviceID)
	return &ClimateDiscovery{
		Name:                     deviceName,
		UniqueID:                 fmt.Sprintf("hvac_manager_%s", deviceID),
		StateTopic:               fmt.Sprintf("homeassistant/climate/%s/state", deviceID),
		TemperatureCommandTopic:  cmdTopic,
		ModeCommandTopic:         cmdTopic,
		FanModeCommandTopic:      cmdTopic,
		TemperatureStateTemplate: "{{ value_json.temperature }}",
		ModeStateTemplate:        "{{ value_json.mode }}",
		FanModeStateTemplate:     "{{ value_json.fan_mode }}",
		AvailabilityTopic:        fmt.Sprintf("homeassistant/climate/%s/availability", deviceID),
		Modes:                    []string{"off", "cool", "heat", "dry", "fan_only", "auto"},
		FanModes:                 []string{"auto", "low", "medium", "high"},
		MinTemp:                  16.0,
		MaxTemp:                  30.0,
		TempStep:                 1.0,
		TemperatureUnit:          "C",
		Precision:                0.1,
		Device: Device{
			Identifiers:  []string{fmt.Sprintf("hvac_manager_%s", deviceID)},
			Name:         deviceName,
			Model:        "HVAC Manager POC",
			Manufacturer: "HVAC Manager",
			SWVersion:    "0.1.0-poc",
		},
	}
}

// ToJSON converts the discovery payload to JSON
func (d *ClimateDiscovery) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// ConfigTopic returns the MQTT topic for publishing this discovery payload
func (d *ClimateDiscovery) ConfigTopic(deviceID string) string {
	return fmt.Sprintf("homeassistant/climate/%s/config", deviceID)
}

// ClimateState represents the current state published to Home Assistant
type ClimateState struct {
	Temperature float64 `json:"temperature"`
	Mode        string  `json:"mode"`
	FanMode     string  `json:"fan_mode"`
}

// ClimateCommand represents a command received from Home Assistant
type ClimateCommand struct {
	Temperature *float64 `json:"temperature,omitempty"`
	Mode        *string  `json:"mode,omitempty"`
	FanMode     *string  `json:"fan_mode,omitempty"`
}

// ParseCommand parses a JSON command from Home Assistant
func ParseCommand(payload []byte) (*ClimateCommand, error) {
	var cmd ClimateCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}
	return &cmd, nil
}

// StateToJSON converts a state struct to JSON
func StateToJSON(state *ClimateState) ([]byte, error) {
	return json.Marshal(state)
}
