package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/diogoaguiar/hvac-manager/internal/mqtt"
)

// Z2MDevice represents a Zigbee2MQTT device
type Z2MDevice struct {
	IEEEAddress  string `json:"ieee_address"`
	FriendlyName string `json:"friendly_name"`
	ModelID      string `json:"model_id"`
	Manufacturer string `json:"manufacturer"`
	Definition   struct {
		Model       string `json:"model"`
		Vendor      string `json:"vendor"`
		Description string `json:"description"`
		Exposes     []struct {
			Type     string                   `json:"type"`
			Features []map[string]interface{} `json:"features,omitempty"`
			Name     string                   `json:"name,omitempty"`
		} `json:"exposes,omitempty"`
	} `json:"definition"`
}

// Z2MBridgeDevices represents the device list from bridge
type Z2MBridgeDevices []Z2MDevice

func main() {
	// Parse command-line flags
	autoUpdate := flag.Bool("y", false, "Automatically update .env file without prompting")
	flag.Parse()

	fmt.Println("🔍 HVAC Manager - Zigbee2MQTT Device Discovery")
	fmt.Println(strings.Repeat("=", 60))

	// Load environment variables
	loadEnv()

	broker := getEnv("MQTT_BROKER", "tcp://localhost:1883")
	username := getEnv("MQTT_USERNAME", "")
	password := getEnv("MQTT_PASSWORD", "")

	fmt.Printf("📡 Connecting to MQTT broker: %s\n", broker)

	// Create MQTT client
	mqttConfig := mqtt.Config{
		Broker:   broker,
		ClientID: "hvac-discovery-tool",
		Username: username,
		Password: password,
	}

	client, err := mqtt.NewClient(mqttConfig)
	if err != nil {
		log.Fatalf("❌ Failed to create MQTT client: %v", err)
	}

	if err := client.Connect(); err != nil {
		log.Fatalf("❌ Failed to connect to broker: %v", err)
	}
	defer client.Disconnect()

	fmt.Println("✅ Connected to broker")
	fmt.Println("\n🔎 Scanning for Zigbee2MQTT devices...")
	fmt.Println("   Listening on topics:")
	fmt.Println("   - zigbee2mqtt/bridge/devices")
	fmt.Println("   - zigbee2mqtt/+") // All device topics

	devices := make(map[string]*Z2MDevice)
	deviceChan := make(chan bool, 1)

	// Subscribe to bridge devices topic
	err = client.Subscribe("zigbee2mqtt/bridge/devices", 0, func(topic string, payload []byte) {
		var deviceList Z2MBridgeDevices
		if err := json.Unmarshal(payload, &deviceList); err != nil {
			log.Printf("⚠️  Failed to parse bridge devices: %v", err)
			return
		}

		for _, device := range deviceList {
			devices[device.FriendlyName] = &device
		}

		deviceChan <- true
	})

	if err != nil {
		log.Fatalf("❌ Failed to subscribe to bridge: %v", err)
	}

	// Also listen to individual device topics to catch any active devices
	err = client.Subscribe("zigbee2mqtt/+", 0, func(topic string, payload []byte) {
		// Extract device name from topic
		parts := strings.Split(topic, "/")
		if len(parts) < 2 {
			return
		}
		deviceName := parts[1]

		// Skip bridge topics
		if strings.HasPrefix(deviceName, "bridge") {
			return
		}

		// Try to parse as device message
		var msg map[string]interface{}
		if err := json.Unmarshal(payload, &msg); err != nil {
			return
		}

		// If we see a device we don't know about, add it
		if _, exists := devices[deviceName]; !exists {
			devices[deviceName] = &Z2MDevice{
				FriendlyName: deviceName,
			}
		}
	})

	if err != nil {
		log.Fatalf("❌ Failed to subscribe to devices: %v", err)
	}

	// Request bridge info
	fmt.Println("\n📤 Requesting device list from Zigbee2MQTT bridge...")
	if err := client.Publish("zigbee2mqtt/bridge/request/devices", 0, false, ""); err != nil {
		log.Printf("⚠️  Failed to request devices: %v", err)
	}

	// Wait for responses with timeout
	fmt.Println("⏳ Waiting for responses (5 seconds)...")
	timeout := time.After(5 * time.Second)
	receivedBridge := false

	select {
	case <-deviceChan:
		receivedBridge = true
	case <-timeout:
		// Continue anyway
	}

	// Give extra time for individual device messages
	time.Sleep(2 * time.Second)

	// Display results
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("📊 Found %d Zigbee2MQTT devices\n", len(devices))
	fmt.Println(strings.Repeat("=", 60))

	if len(devices) == 0 {
		fmt.Println("\n❌ No devices found!")
		fmt.Println("\nTroubleshooting:")
		fmt.Println("  1. Check Zigbee2MQTT is running")
		fmt.Println("  2. Verify MQTT broker address in .env")
		fmt.Println("  3. Check MQTT credentials if authentication required")
		fmt.Println("  4. Ensure devices are paired with Zigbee2MQTT")
		fmt.Println("\nYou can check Zigbee2MQTT web UI or logs for paired devices.")
		return
	}

	// Categorize devices
	var irBlasters []*Z2MDevice
	var otherDevices []*Z2MDevice

	for _, device := range devices {
		if isIRBlaster(device) {
			irBlasters = append(irBlasters, device)
		} else {
			otherDevices = append(otherDevices, device)
		}
	}

	// Display IR blasters
	if len(irBlasters) > 0 {
		fmt.Println("\n📡 IR Blasters Found:")
		fmt.Println(strings.Repeat("-", 60))
		for i, device := range irBlasters {
			fmt.Printf("\n%d. Device: %s\n", i+1, device.FriendlyName)
			if device.Definition.Model != "" {
				fmt.Printf("   Model: %s (%s)\n", device.Definition.Model, device.Definition.Vendor)
			}
			if device.Definition.Description != "" {
				fmt.Printf("   Description: %s\n", device.Definition.Description)
			}
			if device.IEEEAddress != "" {
				fmt.Printf("   IEEE Address: %s\n", device.IEEEAddress)
			}
			fmt.Printf("   ✅ IR transmission capable\n")
		}

		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("💡 Configuration:")
		fmt.Println(strings.Repeat("=", 60))

		selectedDevice := irBlasters[0].FriendlyName

		// Show formatting rules
		fmt.Println("\n📝 .env File Format:")
		fmt.Println("   • Use device name exactly as shown (case-sensitive)")
		fmt.Println("   • No quotes needed")
		fmt.Println("   • Spaces and special characters are allowed")
		fmt.Println("\n✅ Recommended entry:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("IR_BLASTER_ID=%s\n", selectedDevice)
		fmt.Println(strings.Repeat("-", 60))

		if len(irBlasters) > 1 {
			fmt.Println("\n🔄 Alternative IR blasters found:")
			for i := 1; i < len(irBlasters); i++ {
				fmt.Printf("   %d. %s\n", i+1, irBlasters[i].FriendlyName)
			}
			fmt.Println("   (Comment out the line above and use these if needed)")
		}

		// Prompt to update .env file
		if *autoUpdate {
			fmt.Println("\n⚡ Auto-update enabled (-y flag)")
			updateEnvFile(selectedDevice)
		} else {
			fmt.Println("\n❓ Update .env file automatically?")
			fmt.Print("   [y/N]: ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "y" || response == "yes" {
				updateEnvFile(selectedDevice)
			} else {
				fmt.Println("\n⏭️  Skipped. Manually add the line above to your .env file.")
			}
		}
	} else {
		fmt.Println("\n⚠️  No IR blasters detected!")
		fmt.Println("\nCommon IR blaster models:")
		fmt.Println("  - Tuya TS1201 (ZS06)")
		fmt.Println("  - Moes UFO-R11")
		fmt.Println("  - Xiaomi IR Remote")
		fmt.Println("\nMake sure your IR blaster is:")
		fmt.Println("  1. Paired with Zigbee2MQTT")
		fmt.Println("  2. Visible in Zigbee2MQTT web UI")
		fmt.Println("  3. Supported by Zigbee2MQTT")
	}

	// Display other devices (for context)
	if len(otherDevices) > 0 && receivedBridge {
		fmt.Println("\n" + strings.Repeat("-", 60))
		fmt.Printf("📱 Other Zigbee Devices (%d):\n", len(otherDevices))
		for _, device := range otherDevices {
			fmt.Printf("   - %s", device.FriendlyName)
			if device.Definition.Model != "" {
				fmt.Printf(" (%s)", device.Definition.Model)
			}
			fmt.Println()
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("✅ Discovery complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Update .env with IR_BLASTER_ID")
	fmt.Println("  2. Run: make run")
	fmt.Println("  3. Test IR transmission from Home Assistant")
}

// isIRBlaster checks if a device is an IR blaster
func isIRBlaster(device *Z2MDevice) bool {
	if device == nil {
		return false
	}

	// Check model patterns (common IR blasters)
	model := strings.ToLower(device.Definition.Model)
	if strings.Contains(model, "ts1201") || // Tuya ZS06
		strings.Contains(model, "ufo-r11") || // Moes
		strings.Contains(model, "ir remote") { // Xiaomi
		return true
	}

	// Check description for IR keywords
	desc := strings.ToLower(device.Definition.Description)
	if strings.Contains(desc, "ir blaster") ||
		strings.Contains(desc, "ir remote") ||
		strings.Contains(desc, "infrared") {
		return true
	}

	// Check exposes for IR send feature
	for _, expose := range device.Definition.Exposes {
		if expose.Type == "composite" || expose.Type == "specific" {
			if expose.Name == "ir_code_to_send" ||
				strings.Contains(strings.ToLower(expose.Name), "ir") {
				return true
			}
		}
	}

	return false
}

// updateEnvFile updates the .env file with the IR_BLASTER_ID
func updateEnvFile(deviceID string) {
	envPath := ".env"

	// Read existing .env file
	content, err := os.ReadFile(envPath)
	if err != nil {
		fmt.Printf("\n❌ Failed to read .env file: %v\n", err)
		fmt.Println("   Please manually add the configuration.")
		return
	}

	lines := strings.Split(string(content), "\n")
	updated := false
	var newLines []string

	// Look for existing IR_BLASTER_ID line
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is the IR_BLASTER_ID line (active or commented)
		if strings.HasPrefix(trimmed, "IR_BLASTER_ID=") ||
			strings.HasPrefix(trimmed, "#IR_BLASTER_ID=") {
			// Replace with new value
			newLines = append(newLines, fmt.Sprintf("IR_BLASTER_ID=%s", deviceID))
			updated = true
		} else {
			newLines = append(newLines, line)
		}
	}

	// If no existing line found, add it after DATABASE_PATH or at end
	if !updated {
		inserted := false
		for i, line := range newLines {
			if strings.HasPrefix(strings.TrimSpace(line), "DATABASE_PATH=") ||
				strings.HasPrefix(strings.TrimSpace(line), "AC_MODEL_ID=") {
				// Insert after this line
				newLines = append(newLines[:i+1], append([]string{fmt.Sprintf("IR_BLASTER_ID=%s", deviceID)}, newLines[i+1:]...)...)
				inserted = true
				break
			}
		}

		if !inserted {
			// Add at the end
			newLines = append(newLines, fmt.Sprintf("IR_BLASTER_ID=%s", deviceID))
		}
	}

	// Write back to file
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(envPath, []byte(newContent), 0644); err != nil {
		fmt.Printf("\n❌ Failed to write .env file: %v\n", err)
		return
	}

	fmt.Println("\n✅ Successfully updated .env file!")
	fmt.Printf("   Added/updated: IR_BLASTER_ID=%s\n", deviceID)
}

// loadEnv loads environment variables from .env file if it exists
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return // .env is optional
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		os.Setenv(key, value)
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
