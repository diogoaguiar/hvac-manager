# AGENTS.md - AI Assistant Context

> **Purpose:** This file provides structured context for AI coding assistants working on this project. It follows best practices for AI-readable documentation to enable effective collaboration.

## Project Identity

**Name:** HVAC Manager (repo: hvac-manager)  
**Subtitle:** A Go Climate Sidecar for Home Assistant, through MQTT and Zigbee  
**Type:** Standalone Go microservice  
**Purpose:** Intelligent AC control via MQTT and Zigbee2MQTT IR blaster  
**Status:** Phase 4 Complete (Full Integration), Hardware Testing Ready  
**Last Updated:** 2026-01-29

## Critical Context

### What This Project Does
This is a **production-ready** Home Assistant integration with full IR transmission:
1. Connects to MQTT broker
2. Publishes HA MQTT Discovery (climate entity appears automatically)
3. Maintains internal AC state (temperature, mode, fan speed)
4. Receives commands from Home Assistant via MQTT
5. Validates and updates state
6. **Looks up IR codes from SQLite database**
7. **Publishes IR codes to Zigbee2MQTT IR blaster**
8. Publishes state back to HA
9. IR blaster transmits to AC unit

**Current:** Single AC unit per instance, ready for hardware testing  
**Next:** Multi-device support (Phase 6)

### Why It Exists
- **Goal:** Learn Go by building a practical home automation service
- **Current Focus:** Validate HA integration with working E2E POC
- **Problem:** SmartIR provides IR codes, but we need a bridge to HA + Zigbee2MQTT
- **Problem:** SmartIR codes converted to Tuya format for ZS06 IR Blaster
- **Solution:** Go microservice as container sidecar that manages state and IR code dispatch

## Project Structure

```
hvac-manager/
├── cmd/
│   └── main.go              # Application entry point (currently minimal)
├── docs/
│   ├── smartir/
│   │   └── reference/       # SmartIR code samples and conversion scripts
│   │       ├── *.json       # Daikin model IR codes (Broadlink & Tuya)
│   │       └── broadlink_to_tuya.py  # Python converter script
│   ├── architecture.md      # System design and data flow
│   ├── protocols.md         # Daikin & Tuya technical details
│   ├── development.md       # Developer setup and guidelines
│   ├── api.md               # MQTT topics and message formats
│   ├── ir-code-prep.md      # IR code preparation workflow
│   └── README.md            # Documentation index
├── go.mod                   # Go module definition (v1.25.5)
├── go.sum                   # Go dependency checksums
├── README.md                # Main project documentation (human-focused)
└── AGENTS.md                # This file (AI-focused)
```

### Code Organization (Phase 4 Complete)
```
cmd/
  ├── main.go                # ✅ Phase 4 - Full integration entry point
  └── demo/main.go           # ✅ Database demo
internal/
  ├── database/              # ✅ SQLite IR code database
  │   ├── database.go        # Core DB operations, queries
  │   ├── loader.go          # Load SmartIR JSON files (auto-converts formats)
  │   ├── converter.go       # Broadlink-to-Tuya conversion API
  │   ├── tuya_codec.go      # Tuya LZ compression algorithm
  │   ├── schema.sql         # Database schema (embedded)
  │   ├── database_test.go   # Database unit tests (73.2% coverage)
  │   ├── converter_test.go  # Conversion validation tests
  │   └── README.md          # Package documentation
  ├── interfaces/            # ✅ Testable interfaces
  │   └── interfaces.go      # IRDatabase, MQTTPublisher
  ├── mocks/                 # ✅ Mock implementations for testing
  │   └── mocks.go           # MockDatabase, MockMQTT
  ├── integration/           # ✅ Phase 4 - IR transmission
  │   ├── ir_sender.go       # SendIRCode function
  │   └── ir_sender_test.go  # 90% coverage, 50+ test scenarios
  ├── mqtt/                  # ✅ MQTT client wrapper
  │   ├── client.go          # Connection, publish, subscribe
  │   └── client_integration_test.go  # Integration tests
  ├── state/                 # ✅ AC state management
  │   ├── state.go           # AC state struct and validation
  │   └── state_test.go      # 100% coverage
  └── homeassistant/         # ✅ HA MQTT Discovery
      ├── discovery.go       # Auto-discovery payload generation
      └── discovery_test.go  # 88.9% coverage
tools/
  ├── db/main.go             # ✅ Database CLI tool
  └── discover/main.go       # ✅ Zigbee2MQTT device discovery
testdata/
  └── ir_codes/              # ✅ Test fixtures
      └── 1109_tuya.json     # Sample SmartIR data
```

## Key Dependencies

- **Eclipse Paho MQTT (Go):** v1.5.1 - MQTT client library
- **modernc.org/sqlite:** v1.44.3 - Pure Go SQLite driver (no CGO)
- **SmartIR database:** Pre-translated IR codes in Tuya format (JSON files)

## Technical Deep Dive
### Phase 4 - Full Integration Flow
```
1. User Action in HA
   ↓ MQTT: homeassistant/climate/living_room/set
2. Go Service receives JSON {"temperature": 21, "mode": "cool"}
   ↓ Parse and validate command
3. State Update
   ↓ Update ACState struct
   ↓ Validate temperature range (16-30°C)
   ↓ Validate mode (off, cool, heat, etc.)
4. IR Code Lookup (integration.SendIRCode)
   ↓ Query SQLite: LookupCode(modelID="1109", mode="cool", temp=21, fan="auto")
   ↓ Returns: "C/MgAQUBFAUUBRQFFAUUBRQFFAUUBRQFFAU..." (Tuya format)
5. Publish IR Code to Zigbee2MQTT
   ↓ MQTT QoS 1: zigbee2mqtt/ir-blaster/set
   ↓ Payload: {"ir_code_to_send": "C/MgAQUBFAU..."}
6. IR Blaster Transmits
   ↓ ZS06 converts Tuya code to IR pulses
   ↓ IR LED transmits to AC unit
7. Publish state back to homeassistant/climate/living_room/state
   ↓ MQTT: {"temperature": 21, "mode": "cool", "fan_mode": "auto"}
8. HA UI updates
   ↓ User sees confirmation
   ↓ AC unit responds to IR command
```

### Critical Technical Details

#### SmartIR Code Database
- **Source:** Pre-translated IR codes from SmartIR project
- **Format:** JSON files in `docs/smartir/reference/` directory
- **Structure:** Maps AC states (temp, mode, fan) to Tuya-format IR codes
- **Tuya Format:** Base64-encoded compressed pulse timings, prefixed with `C/` or `M/`
- **Example Entry:** `{"mode": "cool", "temp": 21, "fan": "auto"}` → `"C/MgAQUBFAU..."`

#### Code Lookup Logic
- **State Matching:** Find exact match for (temperature, mode, fan_speed)
- **Fallbacks:** If exact match not found, use closest temperature or default fan speed
- **Validation:** Ensure requested state exists in database before sending

#### MQTT Topics
- **Subscribe:**
  - `homeassistant/climate/+/set` - HA command input
- **Publish:**
  - `zigbee2mqtt/[device]/set` - IR blaster control
  - `homeassistant/climate/+/state` - State updates to HA
  - `homeassistant/climate/+/config` - Auto-discovery payload (once on startup)

## Development Phases

### Phase 1: Connectivity ✅
- [x] Go environment setup
- [x] MQTT client connection
- [x] Basic publish/subscribe test with captured IR code

### Phase 2: IR Code Database ✅
- [x] SQLite database with schema versioning and migrations
- [x] Load SmartIR JSON files (Tuya format)
- [x] Implement lookup function (state → IR code)
- [x] Handle missing codes gracefully
- [x] Unit tests for lookup logic with real data
- [x] Database CLI tool for management
- [x] WorkingE2E POC ✅
- [x] Basic `ACState` struct with validation
- [x] MQTT client wrapper (connect, publish, subscribe)
- [x] Home Assistant MQTT Discovery payload
- [x] Command parsing and handling
- [x] State synchronization with HA
- [x] Docker Compose setup for MQTT broker
- [x] POC documentation and setup guide
- [x] Full integration demonstration (no IR yet)

### Phase 4: Full Integration ✅
- [x] Connect state changes to IR database lookup
- [x] Implement IR code retrieval on state update (integration.SendIRCode)
- [x] Publish IR codes to Zigbee2MQTT
- [x] Handle IR transmission errors (MQTT disconnection, DB lookup failures)
- [x] Comprehensive unit tests (90%+ coverage for business logic)
- [x] Integration test infrastructure (docker-compose.test.yml)
- [x] State synchronization
- [x] Error handling and recovery
- [x] Device discovery tool (tools/discover)
- [x] Configuration via environment variables
- [ ] End-to-end testing with real hardware (ready, needs physical setup)

### Phase 5: Production Ready 📋
- [ ] Container image (Docker)
- [ ] Docker Compose production deployment
- [ ] Structured logging (JSON format)
- [ ] Health check endpoints
- [ ] Graceful shutdown handling
- [ ] Metrics/monitoring (Prometheus)
- [ ] Documentation for deployment
- [ ] CI/CD pipeline

### Phase 6: Multi-Device Support 📋 (Future)
**Goal:** Support multiple AC units with multiple IR blasters in a single instance

**Architecture changes:**
- Device registry mapping: `device_id → (ir_blaster_id, ac_model_id, location)`
- Configuration format:
  ```env
  # Multi-device configuration
  DEVICES='[
    {"id":"living_room","blaster":"Living Room IR","model":"1109"},
    {"id":"bedroom","blaster":"Bedroom IR","model":"1116"}
  ]'
  ```
- Router logic: `handleCommand()` → determine device → lookup correct blaster
- Multiple climate entities in HA (one per AC unit)
- Separate database per AC model (or model_id column in queries)

**Implementation tasks:**
- [ ] Device registry package
- [ ] JSON/YAML device configuration
- [ ] Command router (device_id → blaster mapping)
- [ ] Discovery tool: detect and configure multiple devices
- [ ] Per-device state tracking
- [ ] Multiple MQTT Discovery payloads
- [ ] Update integration tests for multi-device scenarios

## Common Tasks

### When Adding New Features
1. Update relevant docs in `docs/`
2. Add tests if applicable
3. **Run `make fmt` to format code**
4. Update AGENTS.md and README.md if architecture changes
5. Commit docs and code together

### When Debugging MQTT
- Check topic subscriptions match expected format
- Verify JSON payload structure
- Use MQTT explorer tool to inspect messages
- Check zigbee2mqtt logs for IR blaster errors

### When Modifying Protocol Logic
- Reference [docs/protocols.md](docs/protocols.md) for specification
- Validate checksums manually
- Test with physical AC unit (if available) or captured codes
- Update protocol documentation if behavior changes

## Code Patterns & Conventions

### Go as Learning Tool
**Important:** This project serves as a Go learning exercise for experienced developers.

**Code Quality Guidelines:**
- Write clear, idiomatic Go code that demonstrates best practices
- Add comments explaining Go-specific patterns (e.g., error handling, interfaces, goroutines)
- Balance education with readability - don't over-comment, but explain non-obvious choices
- Use real-world patterns that would appear in production Go services
- Avoid clever tricks; prefer straightforward, understandable implementations
- **Write testable code:** Design functions to be easily unit tested (pure functions, dependency injection)
- **Include tests:** Every feature implementation must include adequate test coverage

**Comment Examples:**
```go
// Good: Explains Go-specific pattern
// Use defer to ensure cleanup even if function panics
defer conn.Close()

// Good: Clarifies design decision
// Accept interface, return struct (Go best practice)
func NewClient(broker MQTTBroker) *Client { ... }

// Avoid: States the obvious
// Create a new variable
var state ACState
```

### Go Style
- Follow standard Go conventions (gofmt, golint)
- Use structured logging (consider adding logging library)
- Error handling: explicit returns, wrap errors with context
- Avoid global state; pass dependencies explicitly

### MQTT Message Handling
- Always validate incoming JSON before parsing
- Use timeouts for synchronous operations
- Publish state updates atomically
- Log all MQTT errors with context

### Testing Strategy
**Critical:** All code must be testable and tested!

- **Unit tests:** Required for all pure functions (code lookup, state validation, JSON parsing)
  - Target: >80% code coverage for business logic ✅ **ACHIEVED**
  - Use table-driven tests for multiple scenarios
  - Test edge cases and error conditions
- **Integration tests:** For MQTT flows (use test broker)
  - Test full command processing pipeline ✅ **IMPLEMENTED**
  - Verify state synchronization
  - Run with: `make test-integration`
- **Test data:** Keep fixtures in `testdata/` directories
  - Store sample SmartIR JSON files ✅ **testdata/ir_codes/1109_tuya.json**
  - Keep example IR codes for validation
- **Mocking:** Use interfaces for external dependencies (MQTT client, file system) ✅ **internal/interfaces + internal/mocks**
- **CI/CD:** All tests must pass before merging
- **Current coverage:** 37.9% overall, 90%+ business logic (state: 100%, integration: 90%, homeassistant: 88.9%)

## Essential Reading Order

1. [README.md](README.md) - Quick overview and Phase 4 setup
2. [docs/poc-setup.md](docs/poc-setup.md) - Complete setup guide
3. [docs/architecture.md](docs/architecture.md) - System design
4. [docs/protocols.md](docs/protocols.md) - Technical protocol specs
5. [docs/api.md](docs/api.md) - MQTT topics and message formats
6. [docs/development.md](docs/development.md) - Development workflows
7. [docs/ir-code-prep.md](docs/ir-code-prep.md) - IR code conversion workflow
8. [cmd/main.go](cmd/main.go) - Phase 4 implementation entry point
9. [internal/integration/ir_sender.go](internal/integration/ir_sender.go) - Core IR transmission logic

## Diagrams and Visualizations

**Note:** This project uses **Mermaid** format for diagrams. When creating or editing diagrams:
- Use Mermaid syntax for architecture and flow diagrams
- Use sequence diagrams for code-level interactions
- Avoid ASCII art diagrams in favor of Mermaid

## External References

### Specifications & Research
- [Daikin Protocol Analysis](https://github.com/danny-source/Arduino-IRremote/blob/master/ir_Daikin.cpp) - Reference implementation
- [Tuya IR Codec Spec](https://gist.github.com/mildsunrise/1d576669b63a260d2cff35fda63ec0b5) - Compression format
- [Broadlink→Tuya Converter](https://gist.github.com/svyatogor/7839d00303998a9fa37eb079328e4ddaf9) - Python reference
- [SmartIR Project](https://github.com/smartHomeHub/SmartIR) - IR code database source

### Hardware
- [ZS06 IR Blaster](https://www.aliexpress.com/item/1005003878194474.html) - Hardware device
- [Zigbee2MQTT Docs](https://www.zigbee2mqtt.io/) - Z2M integration guide

### Libraries & Tools (includes POC info), then docs/poc-setup.md, then architecture.md
2. **Validate against code:** Documentation describes intent, code is current reality
3. **POC Status:** Phase 3 (E2E POC) is complete - working HA integration without IRtegrations/climate.mqtt/) - HA integration docs

## AI Assistant Guidelines

### When Asked About This Project
1. **Check documentation first:** README.md, then architecture.md, then specific docs
2. **Validate against code:** Documentation describes intent, code is current reality
3. **Consider phase:** Don't assume Phase 3/4 features exist yet
4. **Reference files correctly:** Use workspace-relative paths with line numbers

### When Suggesting Code Changes
1. **Check go.mod version:** Currently using Go 1.25.5
2. **Maintain consistency:** Follow existing patterns in codebase
3. **Update docs:** If code changes architecture/API, flag documentation updates needed
4. **Test implications:** Suggest test updates alongside code changes

### When Documentation is Outdated
1. **Flag it explicitly:** Tell user "Documentation may be outdated, checking code..."
2. **Verify current state:** Read actual implementation
3. **Suggest updates:** Offer to update documentation to match reality

## Maintenance Reminders

⚠️ **CRITICAL:** Keep this file updated as the project evolves!

Update this file when:
- [ ] Project structure changes (new packages, moved files)
- [ ] Phase transitions (Phase 4 completion marked ✅)
- [ ] Architecture decisions change (MQTT topics, data flow)
- [ ] New dependencies added
- [ ] Major features implemented
- [ ] External references change or become outdated

**Last Major Update:** Phase 4 completion - Full IR Integration (2026-01-29)
