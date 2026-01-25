-- HVAC Manager IR Code Database Schema
-- SQLite schema for storing pre-translated IR codes from SmartIR

-- Model metadata table
-- Stores information about supported AC models
CREATE TABLE IF NOT EXISTS models (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id TEXT NOT NULL UNIQUE,          -- e.g., "1109", "1116"
    manufacturer TEXT NOT NULL,              -- e.g., "Daikin"
    supported_models TEXT NOT NULL,          -- JSON array of model numbers, e.g., ["BRC4C158"]
    commands_encoding TEXT NOT NULL,         -- e.g., "Raw"
    supported_controller TEXT NOT NULL,      -- e.g., "MQTT"
    min_temperature INTEGER NOT NULL,        -- Minimum supported temperature
    max_temperature INTEGER NOT NULL,        -- Maximum supported temperature
    precision REAL NOT NULL,                 -- Temperature precision (1.0 or 0.5)
    operation_modes TEXT NOT NULL,           -- JSON array of supported modes
    fan_modes TEXT NOT NULL,                 -- JSON array of supported fan speeds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- IR codes table
-- Stores the actual IR codes for each state combination
CREATE TABLE IF NOT EXISTS ir_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id TEXT NOT NULL,                  -- References models.model_id
    mode TEXT NOT NULL,                      -- e.g., "cool", "heat", "fan_only", "dry", "off"
    temperature INTEGER,                     -- Temperature (NULL for "off" command)
    fan_speed TEXT,                          -- e.g., "low", "medium", "high" (NULL for "off")
    ir_code TEXT NOT NULL,                   -- Base64-encoded Tuya format IR code
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(model_id) ON DELETE CASCADE,
    -- Ensure unique combinations for each model
    UNIQUE(model_id, mode, temperature, fan_speed)
);

-- Index for fast lookups by state
CREATE INDEX IF NOT EXISTS idx_ir_codes_lookup 
ON ir_codes(model_id, mode, temperature, fan_speed);

-- Index for querying by mode
CREATE INDEX IF NOT EXISTS idx_ir_codes_mode 
ON ir_codes(model_id, mode);

-- Comments for documentation:
-- 
-- Usage Examples:
-- 1. Lookup specific code:
--    SELECT ir_code FROM ir_codes 
--    WHERE model_id='1109' AND mode='cool' AND temperature=21 AND fan_speed='medium';
--
-- 2. Get all codes for a mode:
--    SELECT temperature, fan_speed, ir_code FROM ir_codes 
--    WHERE model_id='1109' AND mode='cool' ORDER BY temperature, fan_speed;
--
-- 3. Get model capabilities:
--    SELECT operation_modes, fan_modes, min_temperature, max_temperature 
--    FROM models WHERE model_id='1109';
--
-- 4. Get "off" command:
--    SELECT ir_code FROM ir_codes 
--    WHERE model_id='1109' AND mode='off';
