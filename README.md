# HVAC Manager

> **A Go Climate Sidecar for Home Assistant, through MQTT and Zigbee**

> **Status:** 🎉 Phase 4 Complete - Full IR Transmission Ready!

A standalone Go microservice for intelligent AC control via Zigbee2MQTT. This service acts as a bridge between Home Assistant and Zigbee IR blasters, managing AC state and dispatching pre-translated IR codes from the SmartIR database.

## 🚀 Quick Start

**NEW:** Full integration with IR transmission to real AC units!

```bash
# 1. Find your IR blaster device
go run tools/discover/main.go

# 2. Configure environment (use discovery tool output)
export MQTT_BROKER="tcp://YOUR_HA_IP:1883"
export MQTT_USERNAME="your_mqtt_user"  # if needed
export MQTT_PASSWORD="your_password"    # if needed
export DATABASE_PATH="ir_codes.db"     # SQLite database for IR codes
export AC_MODEL_ID="1109"               # SmartIR model (e.g., Daikin ARC433A1)
export IR_BLASTER_ID="friendly_name"    # From discovery tool

# 3. Run the service
go run cmd/main.go

# 4. Climate entity appears automatically in Home Assistant! 🎉
#    Check: Settings → Devices & Services → MQTT → "Living Room AC"
#    Control AC: It will send actual IR signals!
```

The service demonstrates MQTT Discovery, state management, IR code lookup from database, and actual IR transmission!

**[🚀 Quick Start Guide →](GETTING-STARTED.md)** | **[📖 Full Setup Guide →](docs/poc-setup.md)**

## Quick Overview

Instead of using static Home Assistant integrations or manual IR code recording, this service:

- **Runs Independently:** Manages AC state and logic in a standalone Go container (perfect for learning Go!)
- **Integrates Seamlessly:** Exposes controls to Home Assistant using MQTT Auto-Discovery (appears as a native Climate entity)
- **Simple & Reliable:** Looks up pre-translated IR codes from SmartIR database based on desired AC state

## Architecture

```mermaid
graph LR
    HA[Home Assistant<br/>UI/Control] <-->|MQTT JSON| GCS[HVAC Manager<br/>State + Lookup]
    GCS <-->|MQTT JSON| Z2M[Zigbee2MQTT<br/>IR Blaster]
    GCS -.->|Queries| DB[(SmartIR<br/>IR Codes<br/>Tuya Format)]
    Z2M -->|IR Signal<br/>via ZS06| AC[Daikin AC Unit]
```

**Tech Stack:**
- **Language:** Go 1.25.5
- **Communication:** MQTT (Eclipse Paho)
- **Hardware:** Zigbee IR Blaster (Model ZS06) via Zigbee2MQTT
- **UI:** Home Assistant (Web/App)

## Project Status

**✅ Phase 1: Connectivity (Complete)**
- [x] Setup Go environment with MQTT client
- [x] Connect to MQTT Broker
- [x] Verify control by re-sending captured IR commands

**✅ Phase 2: IR Code Database (Complete)**
- [x] SQLite database with schema versioning
- [x] Load SmartIR JSON files (Tuya format)
- [x] Implement state-to-code lookup function
- [x] Handle missing codes gracefully
- [x] Unit tests with real SmartIR data
- [x] CLI tool for database management
- [x] Working demo application

**🎉 Phase 3: E2E POC (Complete)**
- [x] Basic `ACState` struct with validation
- [x] MQTT client wrapper
- [x] Home Assistant MQTT Discovery
- [x] Command handling and state synchronization
- [x] Docker Compose setup for testing
- [x] Full integration without IR signals

**✅ Phase 4: Full Integration (Complete)**
- [x] Connect state management to IR database
- [x] Implement IR code lookup on state change (integration.SendIRCode)
- [x] Publish to Zigbee2MQTT for actual IR transmission
- [x] Advanced state validation and error recovery
- [x] Comprehensive test suite (90%+ business logic coverage)
- [x] Device discovery tool
- [ ] Hardware validation (ready for testing)

**📋 Phase 5: Production Ready (Next)**
- [ ] Container image and Docker Compose deployment
- [ ] Structured logging and health checks
- [ ] Metrics and monitoring
- [ ] CI/CD pipeline

**🔮 Phase 6: Multi-Device Support (Future)**
- [ ] Support multiple AC units per instance
- [ ] Device-to-IR-blaster mapping configuration
- [ ] Zone/room-based routing logic
- [ ] Multiple climate entities in Home Assistant

## Current Limitations & Future Work

### Single Device Per Instance (Current)
The current implementation supports **one AC unit controlled by one IR blaster** per service instance. Configuration uses single environment variables:

```env
AC_MODEL_ID="1109"               # One AC model
IR_BLASTER_ID="ir-blaster-01"    # One IR blaster device
```

### Multi-Device Architecture (Phase 6 - Future)
Future versions will support multiple AC units with multiple IR blasters in a single instance:

**Planned Configuration Format:**
```env
DEVICES='[
  {"id":"living_room","blaster":"Living Room IR","model":"1109","location":"Living Room"},
  {"id":"bedroom","blaster":"Bedroom IR","model":"1116","location":"Bedroom"},
  {"id":"office","blaster":"Living Room IR","model":"1109","location":"Office"}
]'
```

**Architecture Components:**
- **Device Registry:** Maps device_id → (ir_blaster_id, ac_model_id, friendly_name)
- **Command Router:** Routes HA commands to correct IR blaster based on device_id
- **State Manager:** Tracks state separately for each AC unit
- **Discovery:** Auto-generates multiple climate entities (one per AC)
- **Database:** Supports multiple AC models simultaneously

**Use Cases:**
- Control 3 AC units with 2 IR blasters (zones with shared blasters)
- Mix different AC models (Daikin, Mitsubishi, LG) in one home
- Room-based climate control with centralized management

## Key Technical Challenges

### 1. Dynamic Protocol Generation
Daikin ACs use a complex, multi-frame protocol with checksums and time-based rolling codes. We cannot simply record static "On"/"Off" commands. Solution: Implement a protocol generator that constructs commands dynamically.

### 2. Encoding Format Conversion
SmartIR databases use **Broadlink format** (`JgB...`), but our ZS06 hardware requires **Tuya Compressed format** (`C/M...`). Solution: Build a custom encoder that outputs Tuya-compatible strings directly.

## Documentation

### For Developers
- [📐 Architecture](docs/architecture.md) - System design, data flow, and component interactions
- [🔧 Development Guide](docs/development.md) - Setup, building, testing, and contributing
- [📡 API & MQTT](docs/api.md) - MQTT topics, message formats, and HA integration
- [🔢 Protocols](docs/protocols.md) - Daikin protocol and Tuya encoding details
- [📝 IR Code Preparation](docs/ir-code-prep.md) - Converting SmartIR codes to Tuya format

### For AI Agents
- [🤖 AGENTS.md](AGENTS.md) - Project structure, patterns, and context for AI assistants

## Quick Start

### Full Integration Setup (Phase 4)

```bash
# Prerequisites: Go 1.25+, Home Assistant with MQTT, Zigbee IR Blaster

# Clone and setup
git clone https://github.com/diogoaguiar/hvac-manager.git
cd hvac-manager
go mod download

# 1. Discover your IR blaster (finds Zigbee2MQTT devices)
go run tools/discover/main.go
# Interactive prompt will help configure .env file

# 2. Configure environment (or use .env file)
export MQTT_BROKER="tcp://YOUR_HA_BROKER_IP:1883"
export MQTT_USERNAME="mqtt_user"         # optional
export MQTT_PASSWORD="password"          # optional
export DATABASE_PATH="ir_codes.db"       # SQLite database
export AC_MODEL_ID="1109"                # SmartIR model (e.g., Daikin ARC433A1)
export IR_BLASTER_ID="your-ir-blaster"   # From discovery tool

# 3. Run the service
go run cmd/main.go

# 4. Control your AC from Home Assistant!
#    Climate entity auto-discovers: Settings → Devices & Services → MQTT
```

**📖 [Complete Setup Guide](docs/poc-setup.md)** - Full instructions with troubleshooting

### Testing Without Hardware

```bash
# Run with test MQTT broker (no IR blaster needed)
make test-integration

# Or manually
docker-compose -f docker-compose.test.yml up -d
export MQTT_BROKER="tcp://localhost:1884"
go run cmd/main.go
```

### Production Build (Future)

```bash
# Build and run (using Make)
make build
./bin/hvac-manager

# See all available commands
make help
```

## Hardware Requirements

- **IR Blaster:** Tuya-compatible Zigbee IR Blaster (ZS06 or equivalent)
  - [Reference listing](https://www.aliexpress.com/item/1005003878194474.html)
- **Zigbee Coordinator:** Any Zigbee2MQTT-compatible coordinator
- **AC Unit:** Daikin AC (initially supporting specific models, expandable)

## Key Resources

- **MQTT Library:** [Eclipse Paho Go Client](https://github.com/eclipse/paho.mqtt.golang)
- **SmartIR Project:** [Home Assistant IR Codes](https://github.com/smartHomeHub/SmartIR)
- **Tuya IR Codec:** [Compression format reference](https://gist.github.com/mildsunrise/1d576669b63a260d2cff35fda63ec0b5)
- **Broadlink→Tuya Converter:** [Community implementations](https://gist.github.com/svyatogor/7839d00303998a9fa37eb079328e4ddaf9)

## Contributing

This project is in active development. Documentation and code are evolving rapidly. Please:
- Check [docs/development.md](docs/development.md) for contribution guidelines
- Ensure changes update relevant documentation
- Test MQTT integration before submitting PRs

## License

[To be determined]

## Project Name

**HVAC Manager** - A Go Climate Sidecar for Home Assistant. The repository is named `hvac-manager` for brevity.
