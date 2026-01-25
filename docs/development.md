# Development Guide

> **Learning Go:** This project serves as a practical learning exercise for experienced developers new to Go. Code follows Go best practices with educational comments explaining Go-specific patterns. See [Code Style & Conventions](#code-style--conventions) section for details.

This guide covers environment setup, development workflow, testing, and contribution guidelines for HVAC Manager.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Development Setup](#development-setup)
3. [Building the Project](#building-the-project)
4. [Configuration](#configuration)
5. [Testing](#testing)
6. [Debugging](#debugging)
7. [Code Style & Conventions](#code-style--conventions)
8. [Contributing](#contributing)
9. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

- **Go:** Version 1.25.5 or higher
  - Download: https://go.dev/dl/
  - Verify: `go version`

- **Git:** For version control
  - Verify: `git --version`

- **MQTT Broker:** Mosquitto or Home Assistant's embedded broker
  - Installation: `sudo apt install mosquitto mosquitto-clients` (Ubuntu/Debian)
  - Verify: `mosquitto -h`

### Optional Tools

- **MQTT Explorer:** GUI for debugging MQTT messages
  - Download: http://mqtt-explorer.com/
  
- **Docker:** For containerized deployment
  - Installation: https://docs.docker.com/get-docker/

- **golangci-lint:** For code quality checks
  - Installation: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

### Hardware Requirements

For full testing, you'll need:
- Zigbee coordinator (for Zigbee2MQTT)
- Tuya-compatible IR blaster (ZS06 or equivalent)
- Daikin AC unit (for hardware-in-the-loop testing)

Development and unit testing can be done without hardware.

---

## Development Setup

### 1. Clone Repository

```bash
git clone https://github.com/diogoaguiar/hvac-manager.git
cd hvac-manager
```

### 2. Install Dependencies

```bash
go mod download
```

This will download all dependencies listed in [go.mod](../go.mod):
- `github.com/eclipse/paho.mqtt.golang` - MQTT client
- Supporting libraries

### 3. Verify Installation

```bash
go build ./cmd
```

If successful, this creates a `hvac-manager` executable (or `hvac-manager.exe` on Windows).

### 4. Setup MQTT Broker (Local Development)

#### Option A: Mosquitto (Standalone)

```bash
# Install
sudo apt install mosquitto mosquitto-clients

# Start service
sudo systemctl start mosquitto
sudo systemctl enable mosquitto

# Test
mosquitto_sub -h localhost -t test &
mosquitto_pub -h localhost -t test -m "hello"
```

#### Option B: Home Assistant Mosquitto Add-on

If you're running Home Assistant OS:
1. Navigate to **Settings → Add-ons → Add-on Store**
2. Install **Mosquitto broker**
3. Start the add-on
4. Note the broker address (usually `homeassistant.local:1883`)

### 5. Setup Zigbee2MQTT (Optional for Hardware Testing)

Follow the official guide: https://www.zigbee2mqtt.io/guide/installation/

Connect your IR blaster (ZS06) and note its device ID in Zigbee2MQTT.

---

## Building the Project

### Using Make (Recommended)

The project includes a Makefile for common tasks:

```bash
make build    # Build binary to bin/hvac-manager
make test     # Run all tests
make run      # Run directly without building
make fmt      # Format code
make vet      # Run static analysis
make check    # Run fmt, vet, and test
make coverage # Generate HTML coverage report
make clean    # Remove build artifacts
make help     # Show all available commands
```

### Development Build

```bash
go build -o go-climate-sidecar ./cmd
```

### Production Build (Optimized)

```bash
go build -ldflags="-s -w" -o go-climate-sidecar ./cmd
```

Flags:
- `-s`: Strip symbol table
- `-w`: Strip debug info
- Result: Smaller binary size

### Cross-Compilation

Build for different platforms:

```bash
# Linux ARM (Raspberry Pi)
GOOS=linux GOARCH=arm GOARM=7 go build -o go-climate-sidecar-arm ./cmd

# Linux x86-64
GOOS=linux GOARCH=amd64 go build -o go-climate-sidecar-amd64 ./cmd

# macOS
GOOS=darwin GOARCH=amd64 go build -o go-climate-sidecar-macos ./cmd
```

### Docker Build

```bash
docker build -t go-climate-sidecar:latest .
```

---

## Configuration

### Environment Variables

Create a `.env` file in the project root:

```bash
# MQTT Configuration
MQTT_BROKER=tcp://homeassistant.local:1883
MQTT_CLIENT_ID=go-climate-sidecar
MQTT_USERNAME=your-username
MQTT_PASSWORD=your-password

# Device Configuration
DEVICE_ID=living_room_ac
ZIGBEE_DEVICE_ID=ir_blaster_01

# Logging
LOG_LEVEL=debug  # debug, info, warn, error
LOG_FORMAT=json  # json, text

# Development
DEV_MODE=true
MOCK_HARDWARE=false
```

### Configuration File (Future)

A YAML/JSON configuration file is planned for Phase 4:

```yaml
mqtt:
  broker: tcp://homeassistant.local:1883
  client_id: go-climate-sidecar
  auth:
    username: user
    password: pass

devices:
  - id: living_room_ac
    name: "Living Room AC"
    zigbee_device: ir_blaster_01
    model: daikin_ftxm35
    features:
      min_temp: 16
      max_temp: 30
      modes: [cool, heat, dry, fan, auto]
      fan_speeds: [auto, quiet, 1, 2, 3, 4, 5]
```

---

## Testing

> **Critical:** All code must be testable and tested! Write tests as you implement features, not as an afterthought.

### Testing Philosophy

- **Test-Driven Development (TDD):** Write tests first when practical
- **Coverage Goals:** Aim for >80% coverage on business logic
- **Test Quality:** Tests should be readable, maintainable, and fast
- **Fail Fast:** Tests should catch bugs before they reach production

### Unit Tests

Run all unit tests:

```bash
go test ./...
```

Run with coverage:

```bash
go test -cover ./...
```

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

**Coverage targets:**
- Core logic (state, lookup): >90%
- MQTT handlers: >70%
- Overall project: >80%

### Test Structure

```
internal/
  protocol/
    daikin_test.go      # Protocol generation tests
    state_test.go       # State management tests
  encoder/
    tuya_test.go        # Compression tests
    pulses_test.go      # Timing conversion tests
  mqtt/
    handler_test.go     # MQTT message tests
```

### Writing Tests

#### Table-Driven Tests (Go Best Practice)

```go
package ircodes

import "testing"

func TestLookupCode(t *testing.T) {
    // Table-driven test: efficient way to test multiple scenarios
    tests := []struct {
        name    string
        state   ACState
        want    string
        wantErr bool
    }{
        {
            name: "cool mode 21C auto fan",
            state: ACState{Mode: "cool", Temp: 21, Fan: "auto"},
            want: "C/MgAQUBFAU...",
            wantErr: false,
        },
        {
            name: "invalid mode",
            state: ACState{Mode: "invalid", Temp: 21, Fan: "auto"},
            want: "",
            wantErr: true,
        },
        {
            name: "temp out of range",
            state: ACState{Mode: "cool", Temp: 50, Fan: "auto"},
            want: "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        // Run each test case as a subtest
        t.Run(tt.name, func(t *testing.T) {
            got, err := LookupCode(tt.state)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("LookupCode() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if got != tt.want {
                t.Errorf("LookupCode() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

#### Testing with Mocks

```go
package mqtt

import "testing"

// Mock MQTT client for testing
type MockClient struct {
    publishedMessages []Message
}

func (m *MockClient) Publish(topic string, payload []byte) error {
    m.publishedMessages = append(m.publishedMessages, Message{topic, payload})
    return nil
}

func TestHandleCommand(t *testing.T) {
    mockClient := &MockClient{}
    handler := NewHandler(mockClient)
    
    cmd := `{"temperature": 21, "mode": "cool"}`
    err := handler.HandleCommand("test/set", []byte(cmd))
    
    if err != nil {
        t.Fatalf("HandleCommand() error = %v", err)
    }
    
    // Verify IR code was published
    if len(mockClient.publishedMessages) != 1 {
        t.Errorf("Expected 1 published message, got %d", len(mockClient.publishedMessages))
    }
}
```

### Integration Tests

Test MQTT communication with a test broker:

```bash
# Terminal 1: Start test broker
mosquitto -v -p 1884

# Terminal 2: Run integration tests
MQTT_BROKER=tcp://localhost:1884 go test -tags=integration ./...
```

### Hardware-in-the-Loop Tests

Test with real hardware (when available):

```bash
# Set hardware flag
MOCK_HARDWARE=false go run ./cmd
```

Monitor in another terminal:

```bash
mosquitto_sub -h localhost -t 'zigbee2mqtt/#' -v
```

### Test Data

Store test fixtures in `testdata/` directories:

```
internal/encoder/testdata/
  daikin_cool_21c.bin       # Raw timing data
  daikin_cool_21c_tuya.txt  # Expected Tuya output
```

---

## Debugging

### Logging

Add debug logging to your code:

```go
import "log"

log.Printf("DEBUG: Processing command: %+v", command)
log.Printf("INFO: Generated IR code: %s", irCode)
log.Printf("ERROR: Failed to connect: %v", err)
```

For structured logging (recommended for Phase 4):

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("Processing command",
    zap.String("device", deviceID),
    zap.Float32("temperature", temp),
    zap.String("mode", mode),
)
```

### MQTT Debugging

#### Monitor All Topics

```bash
mosquitto_sub -h localhost -t '#' -v
```

#### Monitor Specific Topics

```bash
# Watch Home Assistant commands
mosquitto_sub -h localhost -t 'homeassistant/climate/+/set' -v

# Watch Zigbee2MQTT
mosquitto_sub -h localhost -t 'zigbee2mqtt/#' -v
```

#### Manual Testing

Simulate Home Assistant commands:

```bash
mosquitto_pub -h localhost \
  -t 'homeassistant/climate/living_room/set' \
  -m '{"temperature": 21, "mode": "cool"}'
```

### Go Debugger (Delve)

Install Delve:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Debug the application:

```bash
dlv debug ./cmd
```

Set breakpoints:

```
(dlv) break main.main
(dlv) break internal/protocol.GenerateFrame2
(dlv) continue
```

### VS Code Debugging

Create `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd",
            "env": {
                "MQTT_BROKER": "tcp://localhost:1883",
                "LOG_LEVEL": "debug"
            }
        }
    ]
}
```

### Common Debug Scenarios

#### MQTT Connection Issues

```go
opts := mqtt.NewClientOptions()
opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
    log.Printf("Connection lost: %v", err)
})
opts.SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
    log.Println("Reconnecting...")
})
```

#### Protocol Generation Issues

```go
// Add hex dump
import "encoding/hex"
log.Printf("Generated frame: %s", hex.EncodeToString(frame))
```

#### Compression Issues

```go
// Test round-trip
original := []byte{...}
compressed := compressTuya(original)
decompressed := decompressTuya(compressed)

if !bytes.Equal(original, decompressed) {
    log.Fatal("Compression round-trip failed!")
}
```

---

## Code Style & Conventions

### Go Standards

Follow standard Go conventions:

```bash
# Format code (ALWAYS run after editing Go files)
go fmt ./...

# Check for common issues
go vet ./...

# Run linter (if installed)
golangci-lint run
```

**Important:** Always run `go fmt ./...` after creating or editing Go files. This ensures consistent formatting according to Go standards and prevents formatting-related diffs in version control.

### Naming Conventions

- **Packages:** Short, lowercase, no underscores (`protocol`, `mqtt`, `encoder`)
- **Files:** Lowercase with underscores (`daikin_generator.go`, `tuya_encoder.go`)
- **Types:** PascalCase (`ACState`, `DaikinFrame`)
- **Functions/Methods:** PascalCase for exported, camelCase for private
- **Constants:** PascalCase or UPPER_SNAKE_CASE for global constants

### Code Organization

```go
// 1. Package declaration
package protocol

// 2. Imports (grouped: std, external, internal)
import (
    "fmt"
    "time"
    
    "github.com/eclipse/paho.mqtt.golang"
    
    "github.com/diogoaguiar/hvac-manager/internal/encoder"
)

// 3. Constants
const (
    DaikinHeader = 0x11
    MaxTemperature = 32
)

// 4. Types
type ACState struct {
    Power bool
    Mode  Mode
}

// 5. Functions
func GenerateFrame(state ACState) []byte {
    // ...
}
```

### Error Handling

Always check and handle errors explicitly:

```go
// Good
data, err := readFile("config.json")
if err != nil {
    return fmt.Errorf("failed to read config: %w", err)
}

// Bad
data, _ := readFile("config.json")
```

### Comments

**Learning-Focused Comments:** Since this project serves as a Go learning tool, add educational comments for Go-specific patterns:

```go
// Good: Explains Go-specific pattern for experienced devs
// Defer ensures MQTT client disconnects even if function panics.
// Deferred calls execute in LIFO order.
defer client.Disconnect(250)

// Good: Clarifies Go idiom
// Accept interfaces, return concrete types (Go best practice).
// This allows callers to pass any MQTT client implementation.
func NewManager(client mqtt.Client) *StateManager {
    return &StateManager{client: client}
}

// Avoid: States the obvious
// Set power to true
state.Power = true

// Avoid: Over-explaining basic operations
// Loop through the array of modes
for _, mode := range modes {
    // Process each mode
    processMode(mode)
}
```

**General Guidelines:**
- Document all exported functions, types, and constants (required by golint)
- Explain *why* not *what* for complex logic
- Add context for non-obvious Go patterns (channels, goroutines, error wrapping)
- Keep educational comments concise - target experienced developers learning Go
// Returns a 19-byte slice with checksum in the last position.
func GenerateFrame2(state ACState) []byte {
    // Implementation...
}
```

---

## Contributing

### Workflow

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/my-feature`
3. **Make** your changes
4. **Test** thoroughly: `go test ./...`
5. **Format** code: `go fmt ./...`
6. **Commit** with clear messages: `git commit -m "Add Daikin frame generation"`
7. **Push** to your fork: `git push origin feature/my-feature`
8. **Open** a Pull Request

### Commit Messages

Follow conventional commits:

```
feat: add Tuya compression algorithm
fix: correct checksum calculation for frame 2
docs: update protocol documentation
test: add unit tests for state transitions
refactor: extract pulse timing to separate package
```

### Pull Request Checklist

- [ ] Code builds without errors
- [ ] **Code formatted** (`go fmt ./...` - REQUIRED before commit)
- [ ] **All tests pass** (`go test ./...`)
- [ ] **New tests added** for new functionality (required!)
- [ ] **Coverage maintained/improved** (`go test -cover ./...`)
- [ ] No linter warnings (`go vet ./...`)
- [ ] Documentation updated (README, docs/)
- [ ] AGENTS.md updated if architecture changed
- [ ] Commit messages are clear and descriptive
- [ ] PR description explains changes and motivation

### Documentation Updates

**Critical:** Always update documentation alongside code changes.

- Code changes → Update relevant docs in `docs/`
- New features → Update [README.md](../README.md) and [AGENTS.md](../AGENTS.md)
- Architecture changes → Update [architecture.md](architecture.md)
- Protocol changes → Update [protocols.md](protocols.md)
- New dependencies → Update [go.mod](../go.mod) comments

---

## Troubleshooting

### "Cannot connect to MQTT broker"

**Check broker is running:**
```bash
mosquitto -v
```

**Check firewall:**
```bash
sudo ufw allow 1883/tcp
```

**Test connectivity:**
```bash
telnet localhost 1883
```

### "Module not found"

**Sync dependencies:**
```bash
go mod tidy
go mod download
```

### "Import cycle not allowed"

Reorganize imports to break circular dependencies. Generally:
- `protocol` should not import `mqtt`
- `encoder` should not import `protocol`
- Use interfaces to decouple components

### "Checksum mismatch"

**Verify byte order:**
```go
// Debug output
for i, b := range frame {
    fmt.Printf("Byte %d: 0x%02X\n", i, b)
}
```

**Check calculation:**
```go
sum := 0
for i := 0; i < len(frame)-1; i++ {
    sum += int(frame[i])
    fmt.Printf("After byte %d: sum = %d\n", i, sum)
}
fmt.Printf("Final checksum: 0x%02X\n", byte(sum&0xFF))
```

### "IR blaster not responding"

**Check Zigbee2MQTT logs:**
```bash
docker logs zigbee2mqtt
```

**Verify device topic:**
```bash
mosquitto_sub -t 'zigbee2mqtt/bridge/devices' -C 1
```

**Test with known-good code:**
```bash
mosquitto_pub -t 'zigbee2mqtt/ir_blaster/set' \
  -m '{"ir_code_to_send": "C/..."}'  # Use captured code
```

### Performance Issues

**Profile CPU usage:**
```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```

**Profile memory:**
```bash
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

**Check allocations in hot paths:**
```bash
go test -benchmem -bench=GenerateFrame
```

---

## Development Roadmap

### Current Phase: Phase 2 (Encoder)

Focus areas:
- Implement Tuya compression algorithm
- Unit tests for compression/decompression
- Verify ZS06 accepts generated codes

### Next Steps: Phase 3 (Generator)

- Implement `DaikinState` struct
- Write frame generation functions
- Connect encoder to generator

### Future: Phase 4 (HA Integration)

- MQTT Discovery implementation
- Command parsing from Home Assistant
- State synchronization

---

## Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [Eclipse Paho Go Client](https://github.com/eclipse/paho.mqtt.golang)
- [MQTT Protocol](https://mqtt.org/)
- [Home Assistant MQTT](https://www.home-assistant.io/integrations/mqtt/)
- [Zigbee2MQTT](https://www.zigbee2mqtt.io/)
- [Project Architecture](architecture.md)
- [Protocol Specifications](protocols.md)
- [API Documentation](API.md)

---

## Getting Help

- **Documentation:** Check [README.md](../README.md) and docs in `docs/`
- **Issues:** Open an issue on GitHub with:
  - Go version (`go version`)
  - OS and architecture
  - Error messages and logs
  - Steps to reproduce
  - Expected vs actual behavior

- **Discussions:** Use GitHub Discussions for questions and ideas

---

**Remember:** Keep documentation updated as the project evolves! 📚
