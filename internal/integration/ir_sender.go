package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/diogoaguiar/hvac-manager/internal/interfaces"
	"github.com/diogoaguiar/hvac-manager/internal/state"
)

// SendIRCode looks up the IR code for the current AC state and publishes it to Zigbee2MQTT
func SendIRCode(ctx context.Context, db interfaces.IRDatabase, mqtt interfaces.MQTTPublisher, modelID, irBlasterID string, acState *state.ACState) error {
	// Check MQTT connection
	if !mqtt.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	var code string
	var err error

	// Special case for "off" mode - use dedicated off code lookup
	if acState.Mode == "off" {
		code, err = db.LookupOffCode(ctx, modelID)
		if err != nil {
			return fmt.Errorf("failed to lookup off code for model %s: %w", modelID, err)
		}
	} else {
		// Convert float temperature to int (round to nearest)
		temp := int(math.Round(acState.Temperature))

		code, err = db.LookupCode(ctx, modelID, acState.Mode, temp, acState.FanMode)
		if err != nil {
			return fmt.Errorf("failed to lookup IR code for %s: %w", acState.String(), err)
		}
	}

	// Build Zigbee2MQTT payload
	payload := map[string]string{
		"ir_code_to_send": code,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal IR payload: %w", err)
	}

	// Publish to Zigbee2MQTT IR blaster
	topic := fmt.Sprintf("zigbee2mqtt/%s/set", irBlasterID)
	if err := mqtt.Publish(topic, 1, false, payloadJSON); err != nil {
		return fmt.Errorf("failed to publish IR code to %s: %w", topic, err)
	}

	return nil
}
