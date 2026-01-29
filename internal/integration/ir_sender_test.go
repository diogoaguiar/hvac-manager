package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/diogoaguiar/hvac-manager/internal/mocks"
	"github.com/diogoaguiar/hvac-manager/internal/state"
)

func TestSendIRCode_Success(t *testing.T) {
	// Setup
	mockDB := &mocks.MockDatabase{
		Codes: map[string]string{
			"1109:cool:21:low": "C/MgAQUBFAUUBRQFFAUUBRQFFAU...", // Fake Tuya code
		},
	}
	mockMQTT := &mocks.MockMQTT{Connected: true}

	acState := state.NewACState()
	acState.SetMode("cool")
	acState.SetTemperature(21.0)
	acState.SetFanMode("low")

	// Execute
	err := SendIRCode(context.Background(), mockDB, mockMQTT, "1109", "ir-blaster", acState)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify database was called with correct parameters
	if len(mockDB.Calls) != 1 {
		t.Fatalf("Expected 1 DB call, got %d", len(mockDB.Calls))
	}
	expectedCall := "1109:cool:21:low"
	if mockDB.Calls[0] != expectedCall {
		t.Errorf("DB call = %q, want %q", mockDB.Calls[0], expectedCall)
	}

	// Verify MQTT publish was called
	if len(mockMQTT.Published) != 1 {
		t.Fatalf("Expected 1 MQTT publish, got %d", len(mockMQTT.Published))
	}

	pub := mockMQTT.Published[0]

	// Check topic
	expectedTopic := "zigbee2mqtt/ir-blaster/set"
	if pub.Topic != expectedTopic {
		t.Errorf("Topic = %q, want %q", pub.Topic, expectedTopic)
	}

	// Check QoS
	if pub.QoS != 1 {
		t.Errorf("QoS = %d, want 1", pub.QoS)
	}

	// Check retained flag
	if pub.Retained {
		t.Error("Expected retained=false")
	}

	// Check payload structure
	var payload map[string]string
	if err := json.Unmarshal(pub.Payload.([]byte), &payload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}
	if code, ok := payload["ir_code_to_send"]; !ok {
		t.Error("Payload missing 'ir_code_to_send' field")
	} else if code != mockDB.Codes["1109:cool:21:low"] {
		t.Errorf("IR code = %q, want %q", code, mockDB.Codes["1109:cool:21:low"])
	}
}

func TestSendIRCode_OffMode(t *testing.T) {
	// Setup
	mockDB := &mocks.MockDatabase{
		OffCodes: map[string]string{
			"1109": "OFF_CODE_1109",
		},
	}
	mockMQTT := &mocks.MockMQTT{Connected: true}

	acState := state.NewACState()
	acState.SetMode("off")

	// Execute
	err := SendIRCode(context.Background(), mockDB, mockMQTT, "1109", "ir-blaster", acState)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify LookupOffCode was called (not LookupCode)
	if len(mockDB.Calls) != 1 {
		t.Fatalf("Expected 1 DB call, got %d", len(mockDB.Calls))
	}
	if mockDB.Calls[0] != "1109:off" {
		t.Errorf("Expected off code lookup, got %q", mockDB.Calls[0])
	}

	// Verify correct code was sent
	if len(mockMQTT.Published) != 1 {
		t.Fatalf("Expected 1 MQTT publish, got %d", len(mockMQTT.Published))
	}

	var payload map[string]string
	json.Unmarshal(mockMQTT.Published[0].Payload.([]byte), &payload)
	if payload["ir_code_to_send"] != "OFF_CODE_1109" {
		t.Errorf("Wrong off code sent: %q", payload["ir_code_to_send"])
	}
}

func TestSendIRCode_TemperatureRounding(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		expectedKey string
	}{
		{"Round down 21.4", 21.4, "1109:cool:21:auto"},
		{"Round up 21.5", 21.5, "1109:cool:22:auto"},
		{"Round up 21.6", 21.6, "1109:cool:22:auto"},
		{"Exact 22.0", 22.0, "1109:cool:22:auto"},
		{"Round down 16.2", 16.2, "1109:cool:16:auto"},
		{"Round up 29.9", 29.9, "1109:cool:30:auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mocks.MockDatabase{
				Codes: map[string]string{
					"1109:cool:21:auto": "CODE_21",
					"1109:cool:22:auto": "CODE_22",
					"1109:cool:16:auto": "CODE_16",
					"1109:cool:30:auto": "CODE_30",
				},
			}
			mockMQTT := &mocks.MockMQTT{Connected: true}

			acState := state.NewACState()
			acState.SetMode("cool")
			acState.SetTemperature(tt.temperature)

			err := SendIRCode(context.Background(), mockDB, mockMQTT, "1109", "ir-blaster", acState)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(mockDB.Calls) != 1 {
				t.Fatalf("Expected 1 DB call, got %d", len(mockDB.Calls))
			}

			if mockDB.Calls[0] != tt.expectedKey {
				t.Errorf("Temperature %.1f: DB call = %q, want %q",
					tt.temperature, mockDB.Calls[0], tt.expectedKey)
			}
		})
	}
}

func TestSendIRCode_DatabaseError(t *testing.T) {
	mockDB := &mocks.MockDatabase{
		Err: errors.New("database connection lost"),
	}
	mockMQTT := &mocks.MockMQTT{Connected: true}

	acState := state.NewACState()
	acState.SetMode("cool")

	err := SendIRCode(context.Background(), mockDB, mockMQTT, "1109", "ir-blaster", acState)

	// Should return error
	if err == nil {
		t.Fatal("Expected error when database fails, got nil")
	}

	// Should not publish to MQTT when DB lookup fails
	if len(mockMQTT.Published) != 0 {
		t.Errorf("Expected 0 MQTT publishes when DB fails, got %d", len(mockMQTT.Published))
	}
}

func TestSendIRCode_CodeNotFound(t *testing.T) {
	mockDB := &mocks.MockDatabase{
		Codes: map[string]string{
			// No code for cool:21:low
		},
	}
	mockMQTT := &mocks.MockMQTT{Connected: true}

	acState := state.NewACState()
	acState.SetMode("cool")
	acState.SetTemperature(21.0)
	acState.SetFanMode("low")

	err := SendIRCode(context.Background(), mockDB, mockMQTT, "1109", "ir-blaster", acState)

	// Should return error when code not found
	if err == nil {
		t.Fatal("Expected error when code not found, got nil")
	}

	// Should not publish to MQTT
	if len(mockMQTT.Published) != 0 {
		t.Errorf("Expected 0 MQTT publishes when code not found, got %d", len(mockMQTT.Published))
	}
}

func TestSendIRCode_MQTTDisconnected(t *testing.T) {
	mockDB := &mocks.MockDatabase{
		Codes: map[string]string{
			"1109:cool:21:low": "FAKE_CODE",
		},
	}
	mockMQTT := &mocks.MockMQTT{Connected: false}

	acState := state.NewACState()
	acState.SetMode("cool")
	acState.SetTemperature(21.0)
	acState.SetFanMode("low")

	err := SendIRCode(context.Background(), mockDB, mockMQTT, "1109", "ir-blaster", acState)

	// Should return error when MQTT disconnected
	if err == nil {
		t.Fatal("Expected error when MQTT disconnected, got nil")
	}

	// Should not even try database lookup
	if len(mockDB.Calls) != 0 {
		t.Errorf("Expected 0 DB calls when MQTT disconnected, got %d", len(mockDB.Calls))
	}
}

func TestSendIRCode_MQTTPublishError(t *testing.T) {
	mockDB := &mocks.MockDatabase{
		Codes: map[string]string{
			"1109:cool:21:low": "FAKE_CODE",
		},
	}
	mockMQTT := &mocks.MockMQTT{
		Connected: true,
		Err:       errors.New("MQTT publish timeout"),
	}

	acState := state.NewACState()
	acState.SetMode("cool")
	acState.SetTemperature(21.0)
	acState.SetFanMode("low")

	err := SendIRCode(context.Background(), mockDB, mockMQTT, "1109", "ir-blaster", acState)

	// Should return error when publish fails
	if err == nil {
		t.Fatal("Expected error when MQTT publish fails, got nil")
	}

	// Should have attempted the publish
	if len(mockMQTT.Published) != 1 {
		t.Errorf("Expected 1 publish attempt, got %d", len(mockMQTT.Published))
	}
}

func TestSendIRCode_AllModes(t *testing.T) {
	modes := []string{"cool", "heat", "dry", "fan_only", "auto"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			key := "1109:" + mode + ":22:auto"
			mockDB := &mocks.MockDatabase{
				Codes: map[string]string{
					key: "CODE_" + mode,
				},
			}
			mockMQTT := &mocks.MockMQTT{Connected: true}

			acState := state.NewACState()
			acState.SetMode(mode)

			err := SendIRCode(context.Background(), mockDB, mockMQTT, "1109", "ir-blaster", acState)

			if err != nil {
				t.Fatalf("Mode %s failed: %v", mode, err)
			}

			if len(mockDB.Calls) != 1 {
				t.Fatalf("Expected 1 DB call for mode %s, got %d", mode, len(mockDB.Calls))
			}

			if mockDB.Calls[0] != key {
				t.Errorf("Mode %s: expected call %q, got %q", mode, key, mockDB.Calls[0])
			}
		})
	}
}

func TestSendIRCode_AllFanModes(t *testing.T) {
	fanModes := []string{"auto", "low", "medium", "high"}

	for _, fan := range fanModes {
		t.Run(fan, func(t *testing.T) {
			key := "1109:cool:22:" + fan
			mockDB := &mocks.MockDatabase{
				Codes: map[string]string{
					key: "CODE_" + fan,
				},
			}
			mockMQTT := &mocks.MockMQTT{Connected: true}

			acState := state.NewACState()
			acState.SetMode("cool")
			acState.SetFanMode(fan)

			err := SendIRCode(context.Background(), mockDB, mockMQTT, "1109", "ir-blaster", acState)

			if err != nil {
				t.Fatalf("Fan mode %s failed: %v", fan, err)
			}

			if len(mockDB.Calls) != 1 {
				t.Fatalf("Expected 1 DB call for fan %s, got %d", fan, len(mockDB.Calls))
			}

			if mockDB.Calls[0] != key {
				t.Errorf("Fan %s: expected call %q, got %q", fan, key, mockDB.Calls[0])
			}
		})
	}
}
