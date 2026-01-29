# AGENTS.md - AI Assistant Context

> **Concise context for AI assistants.** Links to detailed docs where needed. Avoid duplicating information from other files.

## Project Identity

**HVAC Manager** - Go microservice bridging Home Assistant ↔ Zigbee2MQTT IR blaster for AC control  
**Status:** Phase 4 Complete (Full IR Integration) | Hardware Testing Ready  
**Current:** Single AC unit/blaster per instance | **Next:** Production deployment (Phase 5)  
**Last Updated:** 2026-01-29

## Quick Context

**Purpose:** Learn Go by building a Home Assistant climate integration that:
- Receives HA commands via MQTT → Looks up IR codes (SQLite) → Publishes to Zigbee2MQTT → IR blaster transmits

**Tech Stack:** Go 1.25.5, Eclipse Paho MQTT, modernc.org/sqlite (pure Go), SmartIR database (Tuya format)

**Architecture & Flow:** See [README.md](README.md#architecture) and [docs/architecture.md](docs/architecture.md)

**Current Limitation:** One AC unit per instance. Multi-device support planned for Phase 6 (see [README.md](README.md#current-limitations--future-work))

## Project Structure

```
hvac-manager/
├── cmd/main.go              # Phase 4 entry point - full integration
├── internal/
│   ├── integration/         # SendIRCode - core IR transmission logic
│   ├── database/            # SQLite IR code lookup
│   ├── state/               # ACState struct & validation
│   ├── mqtt/                # MQTT client wrapper
│   ├── homeassistant/       # HA Discovery payloads
│   ├── interfaces/          # Testable interfaces (IRDatabase, MQTTPublisher)
│   └── mocks/               # Test mocks
├── tools/
│   ├── discover/main.go     # Zigbee2MQTT device discovery
│   └── db/main.go           # Database CLI
├── docs/                    # See "Essential Reading" below
└── testdata/ir_codes/       # Test fixtures
```

**Coverage:** 37.9% overall | 90%+ business logic (state: 100%, integration: 90%, homeassistant: 88.9%)

## Key Dependencies

- **Eclipse Paho MQTT (Go):** v1.5.1 - MQTT client library
- **modernc.org/sqlite:** v1.44.3 - Pure Go SQLite driver (no CGO)
- **SmartIR database:** Pre-translated IR codes in Tuya format (JSON files)

## Technical Details

### IR Transmission Flow
```
HA Command (MQTT) → Parse JSON → Validate State → 
integration.SendIRCode() → Database Lookup → 
MQTT Publish (zigbee2mqtt/[blaster]/set) → IR Blaster → AC Unit
```

**Key Functions:**
- `integration.SendIRCode()` - Core IR transmission ([ir_sender.go](internal/integration/ir_sender.go))
- `database.LookupCode()` - Query IR codes by state ([database.go](internal/database/database.go))
- `state.Validate()` - Temperature/mode validation ([state.go](internal/state/state.go))

### MQTT Topics
- **Subscribe:** `homeassistant/climate/+/set` (HA commands)
- **Publish:** `zigbee2mqtt/[device]/set` (IR codes), `homeassistant/climate/+/state` (state updates)

**Full API:** See [docs/api.md](docs/api.md)  
**Protocol Details:** See [docs/protocols.md](docs/protocols.md)  
**Code Lookup Logic:** See [docs/architecture.md](docs/architecture.md#ir-code-lookup) for fallback strategy and validation rules

## Development Phases

**Phase Status:** See [README.md](README.md#project-status) for full phase details

- ✅ Phase 1-3: Connectivity, Database, E2E POC
- ✅ **Phase 4: Full Integration** (Current - Complete, hardware testing pending)
- 📋 Phase 5: Production Ready (containerization, logging, CI/CD)
- 📋 Phase 6: Multi-Device Support ([architecture plan](README.md#current-limitations--future-work))

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
**Current Coverage:** 37.9% overall | 90%+ business logic  
**Details:** See [docs/development.md](docs/development.md) for testing guidelines

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
- [Eclipse Paho MQTT](https://github.com/eclipse/paho.mqtt.golang) - Go MQTT client
- [Home Assistant MQTT](https://www.home-assistant.io/integrations/climate.mqtt/) - HA integration docs

## AI Assistant Guidelines

### When Asked About This Project
1. **Check documentation first:** README.md, then architecture.md, then specific docs
2. **Validate against code:** Documentation describes intent, code is current reality
3. **Reference files correctly:** Use workspace-relative paths with line numbers

### When Suggesting Code Changes
1. **Check go.mod version:** Currently using Go 1.25.5
2. **Maintain consistency:** Follow existing patterns in codebase
3. **Update docs:** If code changes architecture/API, flag documentation updates needed
4. **Test implications:** Suggest test updates alongside code changes

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
