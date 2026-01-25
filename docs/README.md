# Documentation Index

This directory contains comprehensive documentation for the HVAC Manager project.

## Documentation Structure

### Core Documentation

- **[architecture.md](architecture.md)** - System architecture, component design, and data flow diagrams
- **[protocols.md](protocols.md)** - Technical specifications for Daikin IR protocol and Tuya encoding
- **[api.md](api.md)** - MQTT topics, message formats, and Home Assistant integration
- **[development.md](development.md)** - Development setup, testing, and contribution guidelines
- **[ir-code-prep.md](ir-code-prep.md)** - IR code conversion workflow (SmartIR to Tuya format)

### Quick Navigation

#### For New Developers
1. Start with [../README.md](../README.md) for project overview
2. Read [development.md](development.md) for setup instructions
3. Review [architecture.md](architecture.md) to understand system design
4. Check [protocols.md](protocols.md) for technical details

#### For AI Assistants
- Read [../AGENTS.md](../AGENTS.md) for AI-optimized context
- Reference [architecture.md](architecture.md) for component structure
- Use [protocols.md](protocols.md) for implementation details

#### For Integration Work
- [api.md](api.md) - MQTT message formats
- [architecture.md](architecture.md) - Integration points
- [ir-code-prep.md](ir-code-prep.md) - Working with IR codes

#### For Protocol Implementation
- [protocols.md](protocols.md) - Complete specifications
- [architecture.md](architecture.md) - Component responsibilities
- [development.md](development.md) - Testing strategies

## Reference Materials

### SmartIR Reference Files

Located in `smartir/reference/`:
- `1109.json` / `1109_tuya.json` - Daikin model 1109 IR codes (Broadlink & Tuya formats)
- `1116.json` / `1116_tuya.json` - Daikin model 1116 IR codes (Broadlink & Tuya formats)
- `broadlink_to_tuya.py` - Python conversion script

These files serve as:
- Reference implementations for IR code conversion
- Test data for encoder validation
- Examples of Tuya format structure

## Documentation Maintenance

### Keeping Docs Updated

**Critical:** Documentation must evolve with the codebase!

Update documentation when:
- [ ] Adding new features or components
- [ ] Changing MQTT topics or message formats
- [ ] Modifying protocol implementation
- [ ] Updating dependencies or build process
- [ ] Making architectural decisions
- [ ] Fixing bugs that affect behavior

### Documentation Review Checklist

Before merging code changes:
1. ✅ Code changes documented in relevant files
2. ✅ [../README.md](../README.md) updated if user-facing changes
3. ✅ [../AGENTS.md](../AGENTS.md) updated if structure/architecture changes
4. ✅ Examples updated to reflect new APIs
5. ✅ External references checked and updated
6. ✅ Phase status updated in roadmap sections

### File Organization Principles

- **README.md** (root) - Entry point for humans, project overview
- **AGENTS.md** (root) - Entry point for AI, structured context
- **docs/** - Detailed technical documentation
- **docs/smartir/** - Reference materials and conversion tools

## Documentation Standards

### Writing Style

- **Clear and Concise** - Get to the point quickly
- **Examples-Driven** - Show, don't just tell
- **Assume Context** - Link to related docs, don't repeat everything
- **Keep Current** - Remove outdated information promptly

### Code Examples

- Use real, runnable code whenever possible
- Include expected output
- Show error cases, not just happy paths
- Comment non-obvious parts

### Diagrams

- **Use Mermaid for all diagrams** - Better rendering, versioning, and maintainability
- Use flowcharts for architecture and data flow
- Use sequence diagrams for code-level interactions
- Always include text description as alternative

### Links

- Use relative paths for internal docs
- Include external links with full URLs
- Check links periodically for rot

## External Resources

### Official Documentation
- [Go Documentation](https://go.dev/doc/)
- [Eclipse Paho Go](https://pkg.go.dev/github.com/eclipse/paho.mqtt.golang)
- [MQTT Protocol Spec](https://mqtt.org/mqtt-specification/)
- [Home Assistant MQTT](https://www.home-assistant.io/integrations/mqtt/)
- [Zigbee2MQTT Docs](https://www.zigbee2mqtt.io/)

### Technical References
- [Daikin Protocol Analysis](https://github.com/Arduino-IRremote/Arduino-IRremote)
- [Tuya IR Codec](https://gist.github.com/mildsunrise/1d576669b63a260d2cff35fda63ec0b5)
- [SmartIR Project](https://github.com/smartHomeHub/SmartIR)

### Hardware
- [ZS06 IR Blaster Reference](https://www.aliexpress.com/item/1005003878194474.html)

## Contributing to Documentation

### Adding New Documents

1. Create file in appropriate location
2. Add entry to this index
3. Update navigation sections
4. Link from related documents
5. Update [../AGENTS.md](../AGENTS.md) if relevant

### Improving Existing Documents

1. Make changes inline
2. Update "Last Updated" dates if present
3. Check for broken links
4. Ensure examples still work
5. Test commands and code snippets

### Documentation Pull Requests

Documentation-only PRs are welcome! Focus areas:
- Fixing typos and clarity issues
- Adding missing examples
- Improving diagrams
- Expanding troubleshooting sections
- Updating outdated information

---

**Last Updated:** 2026-01-24  
**Documentation Version:** 1.0.0 (Initial Structure)
