# HVAC Manager

> **A Go Climate Sidecar for Home Assistant, through MQTT and Zigbee**

> **Status:** 🚧 Work In Progress - Phase 1 (Connectivity) Complete

A standalone Go microservice for intelligent AC control via Zigbee2MQTT. This service acts as a bridge between Home Assistant and Zigbee IR blasters, managing AC state and dispatching pre-translated IR codes from the SmartIR database.

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

**🚧 Phase 2: IR Code Database (In Progress)**
- [ ] Load SmartIR JSON files (Tuya format)
- [ ] Implement state-to-code lookup function
- [ ] Handle missing codes gracefully

**📋 Phase 3: State Management (Planned)**
- [ ] Implement `ACState` struct (Temp, Mode, Fan, Swing)
- [ ] State validation and transition logic
- [ ] Rate limiting and error handling

**📋 Phase 4: Home Assistant Integration (Planned)**
- [ ] MQTT Auto-Discovery implementation
- [ ] Climate entity command parsing
- [ ] State synchronization with HA

**📋 Phase 4: HA Integration (Planned)**
- [ ] Implement MQTT Auto-Discovery payload
- [ ] Link HA commands to Go logic

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

```bash
# Prerequisites: Go 1.25+, MQTT broker, Zigbee2MQTT with IR blaster

# Clone and setup
git clone https://github.com/diogoaguiar/hvac-manager.git
cd hvac-manager
go mod download

# Configure (create config file - TODO)
# Edit configuration for your MQTT broker and devices

# Build and run (using Make)
make build
./bin/hvac-manager

# Or run directly
make run

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
