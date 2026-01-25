# AGENTS.md - AI Assistant Context

> **Purpose:** This file provides structured context for AI coding assistants working on this project. It follows best practices for AI-readable documentation to enable effective collaboration.

## Project Identity

**Name:** HVAC Manager (repo: hvac-manager)  
**Subtitle:** A Go Climate Sidecar for Home Assistant, through MQTT and Zigbee  
**Type:** Standalone Go microservice  
**Purpose:** Intelligent AC control via MQTT and Zigbee2MQTT IR blaster  
**Status:** Phase 1 Complete (Connectivity), Phase 2 In Progress (Encoder)  
**Last Updated:** 2026-01-24

## Critical Context

### What This Project Does
This is a **stateful IR code lookup service** that:
1. Maintains internal AC state (temperature, mode, fan speed)
2. Looks up appropriate IR codes from SmartIR database (Tuya format)
3. Sends IR commands via MQTT to Zigbee2MQTT IR blaster
4. Communicates with Home Assistant via MQTT
5. Appears as a native Climate entity in Home Assistant via MQTT Discovery

### Why It Exists
- **Goal:** Learn Go by building a practical home automation service
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

### Code Organization (Planned)
```
cmd/main.go                  # Entry point, MQTT setup, main loop
internal/
  ├── mqtt/                  # MQTT client wrapper, message handlers
  ├── state/                 # AC state management
  │   └── state.go          # AC state struct and transitions
  ├── ircodes/               # IR code database and lookup
  │   ├── loader.go         # Load SmartIR JSON files
  │   └── lookup.go         # Find code for given state
  └── homeassistant/         # HA MQTT Discovery integration
      └── discovery.go       # Auto-discovery payload generation
```

## Key Dependencies

- **Eclipse Paho MQTT (Go):** v1.5.1 - MQTT client library
- **No external IR libraries:** We implement protocol generation from scratch

## Technical Deep Dive

### Data Flow (Complete Pipeline)
```
1. User Action in HA
   ↓ MQTT: homeassistant/climate/ac/set
2. Go Service receives JSON {"temperature": 21, "mode": "cool"}
   ↓ Update internal state
3. IR Code Lookup
   ↓ Query SmartIR database: {temp: 21, mode: "cool", fan: "auto"}
   ↓ Retrieve Tuya-format IR code (e.g., "C/MgAQUBFAU...")
4. Publish to zigbee2mqtt/ir-blaster/set
   ↓ MQTT: {"ir_code_to_send": "C/MgAQUBFAU..."}
5. IR Blaster transmits signal
   ↓ Zigbee2MQTT forwards to ZS06
   ↓ ZS06 transmits IR to AC unit
6. Publish state back to homeassistant/climate/ac/state
   ↓ MQTT: {"temperature": 21, "mode": "cool", "fan_mode": "auto"}
7. HA UI updates
   ↓ User sees confirmation
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

### Phase 2: IR Code Database 🚧
- [ ] Load SmartIR JSON files (Tuya format)
- [ ] Implement lookup function (state → IR code)
- [ ] Handle missing codes gracefully
- [ ] Unit tests for lookup logic

### Phase 3: State Management 📋
- [ ] Define `ACState` struct
- [ ] Implement state transitions
- [ ] State validation (temp ranges, valid modes)
- [ ] Track last command timestamp

### Phase 4: HA Integration 📋
- [ ] MQTT Auto-Discovery payload
- [ ] Command parsing from HA
- [ ] State synchronization
- [ ] Error handling and recovery

### Phase 5: Production Ready 📋
- [ ] Container image (Docker)
- [ ] Configuration via environment variables
- [ ] Logging and monitoring
- [ ] Documentation for deployment

## Common Tasks

### When Adding New Features
1. Update relevant docs in `docs/`
2. Add tests if applicable
3. **Run `go fmt ./...` to format code**
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
  - Target: >80% code coverage for business logic
  - Use table-driven tests for multiple scenarios
  - Test edge cases and error conditions
- **Integration tests:** For MQTT flows (use test broker)
  - Test full command processing pipeline
  - Verify state synchronization
- **Test data:** Keep fixtures in `testdata/` directories
  - Store sample SmartIR JSON files
  - Keep example IR codes for validation
- **Mocking:** Use interfaces for external dependencies (MQTT client, file system)
- **CI/CD:** All tests must pass before merging

## Important Files for AI Context

When working on this project, these files are essential reading:

1. [README.md](README.md) - Project overview and quick start
2. [docs/architecture.md](docs/architecture.md) - System design
3. [docs/protocols.md](docs/protocols.md) - Technical protocol specs
4. [docs/api.md](docs/api.md) - MQTT topics and message formats
5. [docs/development.md](docs/development.md) - Development workflows
6. [docs/ir-code-prep.md](docs/ir-code-prep.md) - IR code conversion workflow
7. [cmd/main.go](cmd/main.go) - Current implementation entry point

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

### Libraries & Tools
- [Eclipse Paho Go](https://github.com/eclipse/paho.mqtt.golang) - MQTT client
- [Home Assistant MQTT Climate](https://www.home-assistant.io/integrations/climate.mqtt/) - HA integration docs

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
- [ ] Phase transitions (mark phases complete, update current focus)
- [ ] Architecture decisions change (MQTT topics, data flow)
- [ ] New dependencies added
- [ ] Major features implemented
- [ ] External references change or become outdated

**Last Major Update:** Initial structure creation (2026-01-24)
