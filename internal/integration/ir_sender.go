package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/diogoaguiar/hvac-manager/internal/interfaces"
	"github.com/diogoaguiar/hvac-manager/internal/logger"
	"github.com/diogoaguiar/hvac-manager/internal/state"
)

// SendIRCode looks up the IR code for the current AC state and publishes it to Zigbee2MQTT
func SendIRCode(ctx context.Context, db interfaces.IRDatabase, mqtt interfaces.MQTTPublisher, modelID, irBlasterID string, acState *state.ACState) error {
	logger.Debug("SendIRCode called for state: %s", acState.String())

	// Check MQTT connection
	if !mqtt.IsConnected() {
		logger.Error("MQTT client not connected")
		return fmt.Errorf("MQTT client not connected")
	}
	logger.Debug("MQTT client connected")

	var code string
	var err error

	// Special case for "off" mode - use dedicated off code lookup
	if acState.Mode == "off" {
		logger.Debug("Looking up OFF code for model: %s", modelID)
		code, err = db.LookupOffCode(ctx, modelID)
		if err != nil {
			logger.Error("Failed to lookup off code for model %s: %v", modelID, err)
			return fmt.Errorf("failed to lookup off code for model %s: %w", modelID, err)
		}
		logger.Debug("Found OFF code (length: %d bytes)", len(code))
	} else {
		// Convert float temperature to int (round to nearest)
		temp := int(math.Round(acState.Temperature))

		logger.Debug("Looking up IR code: model=%s mode=%s temp=%d fan=%s",
			modelID, acState.Mode, temp, acState.FanMode)

		code, err = db.LookupCode(ctx, modelID, acState.Mode, temp, acState.FanMode)
		if err != nil {
			logger.Error("Failed to lookup IR code for %s: %v", acState.String(), err)
			return fmt.Errorf("failed to lookup IR code for %s: %w", acState.String(), err)
		}
		logger.Debug("Found IR code (length: %d bytes)", len(code))
		logger.Debug("IR code: %s", code)
	}

	// Build Zigbee2MQTT payload
	payload := map[string]string{
		"ir_code_to_send": code,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Failed to marshal IR payload: %v", err)
		return fmt.Errorf("failed to marshal IR payload: %w", err)
	}

	// Publish to Zigbee2MQTT IR blaster
	topic := fmt.Sprintf("zigbee2mqtt/%s/set", irBlasterID)
	logger.Debug("Publishing to topic: %s", topic)
	logger.Debug("Payload: %s", string(payloadJSON))

	if err := mqtt.Publish(topic, 1, false, payloadJSON); err != nil {
		logger.Error("Failed to publish IR code to %s: %v", topic, err)
		return fmt.Errorf("failed to publish IR code to %s: %w", topic, err)
	}

	logger.Info("📡 IR code sent to %s for state: %s", irBlasterID, acState.String())
	return nil
}
