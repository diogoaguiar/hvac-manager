# Debug Session Notes - 2026-01-29

## Database Available Options

**Modes:** cool, dry, fan_only, heat, off  
**Fan speeds (heat mode):** high, low, medium  
**NO "auto" fan speed in database**  
**NO "auto" mode in database**

---

## TEST 1: Turn AC ON with Heat Mode (22°C, Auto fan)

**Actions taken in HA:**
1. Set mode to "heat" 
2. Set mode to "auto"

**Result:** ❌ FAILED - No AC response (no beep on either command)

**Logs Analysis:**

### Command 1: Heat mode
```
mode=heat temp=22 fan=auto
⚠️  No IR code found for model=1109 mode=heat temp=22 fan=auto
🔍 Found 48 codes for model=1109 mode=heat (any temp/fan)
```
- Database HAS heat codes
- Database has NO fan=auto codes
- Available fans: low, medium, high

### Command 2: Auto mode
```
mode=auto temp=22 fan=auto
⚠️  No IR code found for model=1109 mode=auto temp=22 fan=auto
🔍 Found 0 codes for model=1109 mode=auto (any temp/fan)
```
- Database has NO auto mode at all

**Root Causes:**
1. HA defaults fan to "auto" but DB only has low/medium/high
2. HA offers "auto" mode but DB doesn't have it

**Needed Fixes:**
- [ ] Implement fallback: fan=auto → try fan=low (or medium/high)
- [ ] Remove "auto" from supported modes in HA discovery
- [ ] OR map auto mode to another mode (heat/cool based on temp?)

---

## TEST 2: Heat with explicit fan speed (LOW)

**Actions taken in HA:**
- Changed fan to LOW (while still in OFF mode)

**Result:** ⚠️  UNEXPECTED - Sent OFF code, not HEAT code

**Logs Analysis:**
```
📥 Received command: low
💨 Fan mode set to: low
🔍 SendIRCode called for state: Mode: off, Temp: 22.0°C, Fan: low, Power: false
🔍 Looking up OFF code for model: 1109
✅ Found off code in DB (length: 356 bytes)
📡 IR code sent to Diogo's Office: IR Blaster
```

**What happened:**
- Only changed fan to LOW
- Mode was still OFF from previous state
- System correctly sent OFF code (mode=off takes precedence)
- MQTT delivery successful

**Question for user:** Did AC respond? Did it turn off or do nothing?

**Answer:** No response from AC. OFF codes don't seem to be working.

**Note:** Need to turn AC ON first (set mode to heat), THEN test fan changes

---

## TEST 2b: Heat mode with LOW fan (complete)

**Approach:** Change temp 22→23→22 to trigger commands while on heat/low

**Actions taken:**
- Changed temp to 23°C
- Changed temp back to 22°C

**Result:** ❌ FAILED - No AC response on either command

**Logs Analysis:**
```
Command 1: Temp 23
📥 Received command: 23.0
🌡️  Temperature set to: 23.0°C
🔍 SendIRCode: Mode: off, Temp: 23.0°C, Fan: low
🔍 Looking up OFF code
✅ Sent OFF code

Command 2: Temp 22
📥 Received command: 22.0
🌡️  Temperature set to: 22.0°C
🔍 SendIRCode: Mode: off, Temp: 22.0°C, Fan: low
🔍 Looking up OFF code
✅ Sent OFF code (same code as before)
```

**CRITICAL ISSUE:** Mode is stuck at OFF!
- HA interface shows "heat" but internal state is "off"
- Every command sends OFF code instead of heat code
- Temp changes don't change mode

**Root cause:** Mode was never successfully changed from OFF to HEAT
- Earlier "heat" command failed (no fan=auto in DB)
- System kept mode=off
- HA UI is out of sync with actual state

**User feedback:** Suspects fan=low codes don't work. Has seen medium work.

---

## TEST 3: Try MEDIUM fan to verify it works

**Instruction:** Change fan to MEDIUM (should trigger heat code with medium fan)

**Action taken:** Changed fan to medium

**Result:** ❌ FAILED - Still sending OFF code

**Logs:**
```
📥 Received command: medium
💨 Fan mode set to: medium
🔍 SendIRCode: Mode: off, Temp: 22.0°C, Fan: medium
🔍 Looking up OFF code
✅ Sent OFF code (same OFF code every time)
```

**Confirmation:** Mode is definitely stuck at OFF. Every command sends the same OFF code.

---

## CRITICAL BUG IDENTIFIED: State Synchronization Issue

**Problem:** When IR code lookup/send fails, internal state updates but IR transmission doesn't happen. Then HA UI and service state are out of sync.

**What happened in TEST 1:**
1. HA sent: mode=heat, fan=auto
2. Service tried to set mode to "heat" ✅
3. Service tried to send IR code for heat/22/auto ❌ (no code in DB)
4. IR send failed, BUT mode was already set to "heat" in memory
5. Service published state back to HA as "heat" 
6. BUT service never actually sent a HEAT IR code to AC!

Wait, looking at logs again... Let me check TEST 1 logs more carefully.

Actually, in TEST 1 the state DIDN'T update to heat. Let me trace through the code logic...

**Actual sequence from logs:**
1. Initial state: mode=off, fan=auto
2. Command "heat" received → set mode to heat → try SendIRCode → FAILED → but state shows heat in publish
3. Command "auto" received → set mode to auto → try SendIRCode → FAILED → but state shows auto
4. Then all subsequent commands show mode=off

**Root cause options:**
A) State partially updates on failure then reverts?
B) Plain text mode command sets mode, tries to send, fails, but published state before checking success?
C) Later commands are resetting mode to off somehow?

Looking at the code flow... when SendIRCode fails, we still call publishState. So HA thinks the mode changed, but internally it might have reverted.

**Solution needed:**
1. DON'T update internal state until IR code successfully sent
2. OR: If IR send fails, revert state and publish OLD state back to HA
3. OR: Only publish state after successful IR transmission

**User's suggestions:**
- Inform HA of actual state when command fails
- Track HA requested state vs actual state

**Best approach:** Transactional state updates - only commit state change if IR send succeeds

---

## Key Findings Summary

### 1. State Synchronization Bug (CRITICAL)
**Current behavior:** State updates before verifying IR code can be sent
**Impact:** HA UI shows state that was never actually sent to AC
**Fix needed:** Don't update state until IR code successfully transmitted

### 2. Missing Fan Mode Fallback
**Current:** Exact fan mode match required (auto, low, medium, high)
**Problem:** DB has no "auto" fan codes
**Fix needed:** fan=auto → fallback to low/medium

### 3. Mode "auto" Not in Database
**Current:** HA offers "auto" mode, DB doesn't have it
**Fix needed:** Remove "auto" from HA discovery OR map it

### 4. OFF Codes Don't Work
**Current:** OFF code sends but AC doesn't respond
**Possible causes:** 
- Wrong IR code for this specific AC model
- IR blaster issue
- Code format issue

### 5. Fan "low" Codes Suspected Not Working
**Current:** User reports medium works, but low doesn't
**Needs testing:** Once state sync is fixed, test each fan mode

---

## Recommended Fixes Priority

1. **FIX STATE SYNC** (blocks everything else)
2. Add fan=auto fallback
3. Remove/map mode=auto  
4. Test actual IR codes with physical AC
5. Build feedback/override system

---

## Next Steps

Want to stop here and implement the state sync fix? Or continue testing to gather more data first?

---

## TEST PHASE 2: After Forcing State Reset (00:04:43 - 00:04:55)

### Test 4a: OFF then ON to Force State Transition

**Time:** 00:04:43 → 00:04:46

**Actions:**
1. Turned OFF via HA
2. Turned ON (heat) via HA

**Logs:**
```
00:04:43: Command "off" → mode=off, sent OFF code (356 bytes)
00:04:46: Command "heat" → mode=heat, sent HEAT code (304 bytes) ✅
```

**Result:** ✅ State successfully transitioned from OFF to HEAT internally

### Test 4b: Fan Speed HIGH (While in HEAT Mode)

**Time:** 00:04:55

**Command:** Changed fan from medium to high via HA

**Logs:**
```
00:04:55: Command "high" → mode=heat, temp=22, fan=high
00:04:55: Looking up IR code: model=1109 mode=heat temp=22 fan=high
00:04:55: Found IR code in DB (length: 312 bytes)
00:04:55: Published to zigbee2mqtt
00:04:55: MQTT publish successful
00:04:55: Published state: Mode: heat, Temp: 22.0°C, Fan: high, Power: true
```

**Physical AC Response:** ✅ **AC RESPONDED CORRECTLY!** Switched to heat/22/high

**IR Code:** Different from OFF code (312 bytes vs 356 bytes) - correct HEAT/HIGH code was sent

**Analysis:** Once internal state was properly set to HEAT mode, fan changes work correctly and AC responds as expected.

---

## Updated Summary of Findings

### Database Issues
- ❌ Database has **NO "auto" fan mode** (only low/medium/high available)
- ❌ Database has **NO "auto" mode** for HVAC mode (only cool/dry/fan_only/heat/off)
- ✅ Database has 48 HEAT codes (16 per fan speed: low/medium/high)
- ❓ OFF codes transmit successfully but AC doesn't respond (or maybe works, needs verification)

### Code Behavior
- ⚠️  **CRITICAL BUG:** State updates before IR transmission verified
- ⚠️  When IR send fails, HA UI shows incorrect state (state published regardless of success)
- ⚠️  State can get "stuck" when commands fail - requires explicit mode change to reset
- ✅ Temperature-only changes now trigger IR sends (bug fixed in earlier session)
- ✅ MQTT delivery confirmation works (5s timeout)
- ✅ Database queries properly logged with full context
- ✅ When state is correct, IR codes lookup and transmit successfully

### Test Results Summary
**Phase 1 (State Stuck at OFF):**
- ❌ Test 1: mode=heat + fan=auto → Failed (no auto fan in DB, state stuck)
- ❌ Test 2: Temp changes → Sent OFF codes (mode stuck at OFF)
- ❌ Test 3: fan=medium → Sent OFF code (mode stuck at OFF)

**Phase 2 (After State Reset):**
- ✅ Test 4a: OFF → ON (heat) → State successfully transitioned
- ✅ Test 4b: fan=high (in heat mode) → **WORKING!** AC responded correctly ✅

### Physical AC Behavior (Verified)
- ✅ **Mode HEAT + Fan HIGH + Temp 22** = **WORKS PERFECTLY** ✅
- ✅ Mode HEAT works when internal state is correct
- ✅ Fan speed HIGH works and triggers AC correctly
- ✅ MQTT publish and IR code transmission functional end-to-end
- ❓ Fan speed MEDIUM - likely works (user said it worked earlier)
- ❌ Fan speed LOW - suspected not working (needs verification with correct state)
- ❓ OFF codes transmit but AC response unclear (needs separate test)

### Key Insight
**THE SYSTEM WORKS when internal state is correct!** The primary blocker is the state sync bug that prevents clean state transitions when commands fail.

---
