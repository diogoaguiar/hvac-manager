package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/diogoaguiar/hvac-manager/internal/homeassistant"
	"github.com/diogoaguiar/hvac-manager/internal/mqtt"
	"github.com/diogoaguiar/hvac-manager/internal/state"
)

const (
	defaultBroker   = "tcp://localhost:1883"
	defaultDeviceID = "living_room"
)

// loadEnv loads environment variables from .env file if it exists
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		// .env file is optional, so don't error if it doesn't exist
		return
	}
	defer file.Close()

	log.Println("📄 Loading .env file...")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first = sign
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := parts[1] // Don't trim the value yet

		// Only remove surrounding quotes if they match
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		} else {
			// If no quotes, trim whitespace
			value = strings.TrimSpace(value)
		}

		// Set environment variable (overwrite if from .env)
		os.Setenv(key, value)
		if key == "MQTT_PASSWORD" {
			log.Printf("   ✓ %s=***", key)
		} else {
			log.Printf("   ✓ %s=%s", key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Warning: Error reading .env file: %v", err)
	}
}

func main() {
	// Load .env file if it exists
	loadEnv()
	fmt.Println("🌡️  HVAC Manager - E2E POC")
	fmt.Println("=" + string(make([]byte, 50)) + "=")

	// Configuration from environment or defaults
	broker := getEnv("MQTT_BROKER", defaultBroker)
	deviceID := getEnv("DEVICE_ID", defaultDeviceID)
	username := getEnv("MQTT_USERNAME", "")
	password := getEnv("MQTT_PASSWORD", "")

	log.Printf("Config: Broker=%s, Device=%s", broker, deviceID)

	// Create MQTT client
	mqttConfig := mqtt.Config{
		Broker:   broker,
		ClientID: fmt.Sprintf("hvac-manager-%s", deviceID),
		Username: username,
		Password: password,
	}

	client, err := mqtt.NewClient(mqttConfig)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}

	// Connect to MQTT broker
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer client.Disconnect()

	// Initialize AC state
	acState := state.NewACState()
	log.Printf("Initial state: %s", acState.String())

	// Publish Home Assistant MQTT Discovery
	if err := publishDiscovery(client, deviceID); err != nil {
		log.Fatalf("Failed to publish discovery: %v", err)
	}

	// Publish availability (online)
	availTopic := fmt.Sprintf("homeassistant/climate/%s/availability", deviceID)
	if err := client.Publish(availTopic, 1, true, "online"); err != nil {
		log.Printf("Warning: Failed to publish availability: %v", err)
	}

	// Publish initial state
	if err := publishState(client, deviceID, acState); err != nil {
		log.Printf("Warning: Failed to publish initial state: %v", err)
	}

	// Subscribe to command topic
	cmdTopic := fmt.Sprintf("homeassistant/climate/%s/set", deviceID)
	if err := client.Subscribe(cmdTopic, 1, func(topic string, payload []byte) {
		handleCommand(client, deviceID, acState, payload)
	}); err != nil {
		log.Fatalf("Failed to subscribe to command topic: %v", err)
	}

	fmt.Println("\n✅ POC is running! Integration points:")
	fmt.Printf("   📡 MQTT Broker: %s\n", broker)
	fmt.Printf("   🏠 HA Device ID: %s\n", deviceID)
	fmt.Printf("   📥 Listening on: %s\n", cmdTopic)
	fmt.Printf("   📤 State topic: homeassistant/climate/%s/state\n", deviceID)
	fmt.Println("\nℹ️  This POC does NOT send IR signals - commands are logged only.")
	fmt.Println("   Press Ctrl+C to stop\n")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("\n🛑 Shutting down...")
	// Publish offline status
	if err := client.Publish(availTopic, 1, true, "offline"); err != nil {
		log.Printf("Warning: Failed to publish offline status: %v", err)
	}
}

// publishDiscovery publishes the Home Assistant MQTT Discovery payload
func publishDiscovery(client *mqtt.Client, deviceID string) error {
	discovery := homeassistant.NewClimateDiscovery(deviceID, "Living Room AC")
	payload, err := discovery.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal discovery: %w", err)
	}

	topic := discovery.ConfigTopic(deviceID)
	if err := client.Publish(topic, 2, true, payload); err != nil {
		return fmt.Errorf("failed to publish discovery: %w", err)
	}

	log.Printf("✅ Published discovery to: %s", topic)
	return nil
}

// publishState publishes the current AC state to Home Assistant
func publishState(client *mqtt.Client, deviceID string, acState *state.ACState) error {
	haState := &homeassistant.ClimateState{
		Temperature: acState.Temperature,
		Mode:        acState.Mode,
		FanMode:     acState.FanMode,
	}

	payload, err := homeassistant.StateToJSON(haState)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	topic := fmt.Sprintf("homeassistant/climate/%s/state", deviceID)
	if err := client.Publish(topic, 0, true, payload); err != nil {
		return fmt.Errorf("failed to publish state: %w", err)
	}

	log.Printf("📤 Published state: %s", acState.String())
	return nil
}

// handleCommand processes commands received from Home Assistant
func handleCommand(client *mqtt.Client, deviceID string, acState *state.ACState, payload []byte) {
	fmt.Println("\n" + strings.Repeat("─", 60))
	log.Printf("📥 Received command: %s", string(payload))

	// Try to parse as JSON first
	cmd, err := homeassistant.ParseCommand(payload)
	if err != nil {
		// If JSON parsing fails, treat as plain text (temperature or mode value)
		payloadStr := string(payload)
		log.Printf("📋 Plain text command: %s", payloadStr)

		// Try to parse as temperature (numeric)
		if temp, err := strconv.ParseFloat(payloadStr, 64); err == nil {
			if err := acState.SetTemperature(temp); err != nil {
				log.Printf("❌ Invalid temperature: %v", err)
				return
			}
			log.Printf("🌡️  Temperature set to: %.1f°C", temp)
			if err := publishState(client, deviceID, acState); err != nil {
				log.Printf("❌ Failed to publish state: %v", err)
			}
			fmt.Println(strings.Repeat("─", 60))
			return
		}

		// Otherwise treat as mode or fan mode
		if err := acState.SetMode(payloadStr); err == nil {
			log.Printf("🔄 Mode set to: %s", payloadStr)
			log.Printf("💡 [POC] Would look up IR code for: %s", acState.String())
			log.Printf("💡 [POC] Would publish to: zigbee2mqtt/ir-blaster/set")
			if err := publishState(client, deviceID, acState); err != nil {
				log.Printf("❌ Failed to publish state: %v", err)
			}
			fmt.Println(strings.Repeat("─", 60))
			return
		}

		if err := acState.SetFanMode(payloadStr); err == nil {
			log.Printf("💨 Fan mode set to: %s", payloadStr)
			log.Printf("💡 [POC] Would look up IR code for: %s", acState.String())
			log.Printf("💡 [POC] Would publish to: zigbee2mqtt/ir-blaster/set")
			if err := publishState(client, deviceID, acState); err != nil {
				log.Printf("❌ Failed to publish state: %v", err)
			}
			fmt.Println(strings.Repeat("─", 60))
			return
		}

		log.Printf("❌ Could not parse command as JSON or plain text: %s", payloadStr)
		return
	}

	// Pretty print the command for visibility
	cmdJSON, _ := json.MarshalIndent(cmd, "", "  ")
	log.Printf("📋 Parsed command:\n%s", string(cmdJSON))

	// Apply changes to state
	stateChanged := false

	if cmd.Temperature != nil {
		if err := acState.SetTemperature(*cmd.Temperature); err != nil {
			log.Printf("❌ Invalid temperature: %v", err)
			return
		}
		stateChanged = true
		log.Printf("🌡️  Temperature set to: %.1f°C", *cmd.Temperature)
	}

	if cmd.Mode != nil {
		if err := acState.SetMode(*cmd.Mode); err != nil {
			log.Printf("❌ Invalid mode: %v", err)
			return
		}
		stateChanged = true
		log.Printf("🔄 Mode set to: %s", *cmd.Mode)
	}

	if cmd.FanMode != nil {
		if err := acState.SetFanMode(*cmd.FanMode); err != nil {
			log.Printf("❌ Invalid fan mode: %v", err)
			return
		}
		stateChanged = true
		log.Printf("💨 Fan mode set to: %s", *cmd.FanMode)
	}

	if !stateChanged {
		log.Println("⚠️  No valid state changes in command")
		return
	}

	// Log what would be sent to IR blaster (POC - no actual IR sending)
	log.Printf("💡 [POC] Would look up IR code for: %s", acState.String())
	log.Printf("💡 [POC] Would publish to: zigbee2mqtt/ir-blaster/set")

	// Publish updated state back to Home Assistant
	if err := publishState(client, deviceID, acState); err != nil {
		log.Printf("❌ Failed to publish state: %v", err)
	}

	fmt.Println(strings.Repeat("─", 60))
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
