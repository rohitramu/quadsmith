-- ============================================================================
-- 2. BASE INFRASTRUCTURE & SHARED TABLES
-- ============================================================================

CREATE TABLE IF NOT EXISTS vtx_ecosystem (
    id VARCHAR(50) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS rf_protocol (
    id VARCHAR(50) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS antenna_polarization (
    id VARCHAR(50) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS antenna_connector (
    id VARCHAR(50) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS battery_connector (
    id VARCHAR(50) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS company (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    website_url VARCHAR(2048)
);

CREATE TABLE IF NOT EXISTS hardware_component (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    model_name VARCHAR(150) NOT NULL,
    release_year INT,
    release_month INT,
    release_day INT,
    weight_g NUMERIC(6, 2)
);

CREATE TABLE IF NOT EXISTS hardware_component_company (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    hardware_component_id INT NOT NULL REFERENCES hardware_component(id) ON DELETE CASCADE,
    company_id INT NOT NULL REFERENCES company(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS hardware_component_link (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    hardware_component_id INT NOT NULL REFERENCES hardware_component(id) ON DELETE CASCADE,
    company_id INT REFERENCES company(id) ON DELETE SET NULL,
    url VARCHAR(2048) NOT NULL,
    link_type VARCHAR(50) NOT NULL CHECK (link_type IN ('Purchase', 'Datasheet', 'STL', 'Manual', 'Firmware', 'Video')),
    title VARCHAR(150) NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS battery_chemistry (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    nominal_voltage_per_cell_v NUMERIC(4, 2) NOT NULL,
    max_voltage_per_cell_v NUMERIC(4, 2) NOT NULL,
    min_voltage_per_cell_v NUMERIC(4, 2) NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS tag (
    id VARCHAR(50) PRIMARY KEY CHECK (id ~ '^[a-z](-?[a-z0-9])+$'),
    description TEXT
);

CREATE TABLE IF NOT EXISTS incompatibility_issue (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    incompatibility_reason VARCHAR(50) NOT NULL CHECK (incompatibility_reason IN ('Physical', 'Noise', 'Electronic', 'Firmware', 'Unknown')),
    description TEXT NOT NULL,
    suggested_fix VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS incompatibility_hardware_component (
    issue_id INT NOT NULL REFERENCES incompatibility_issue(id) ON DELETE CASCADE,
    hardware_component_id INT NOT NULL REFERENCES hardware_component(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, hardware_component_id)
);

-- ============================================================================
-- 3. HARDWARE SUBTYPES (Supertype/Subtype Pattern)
-- ============================================================================

CREATE TABLE IF NOT EXISTS frame (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    wheelbase_mm INT NOT NULL,
    max_prop_size_mm NUMERIC(5, 2) NOT NULL,
    fc_stack_mount_mm TEXT[] NOT NULL,
    motor_mount_pattern TEXT[] NOT NULL,
    camera_mount_width_mm NUMERIC(4, 2) NOT NULL,
    vtx_mount_mm TEXT[],
    max_battery_length_mm NUMERIC(5, 2),
    max_battery_width_mm NUMERIC(5, 2),
    max_battery_height_mm NUMERIC(5, 2)
);

CREATE TABLE IF NOT EXISTS motor (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    stator_size VARCHAR(20) NOT NULL,
    kv_rating INT NOT NULL,
    input_voltage_min_v NUMERIC(4, 2) NOT NULL,
    input_voltage_max_v NUMERIC(4, 2) NOT NULL,
    mount_pattern_mm VARCHAR(30) NOT NULL,
    mount_bolt_size VARCHAR(50) NOT NULL CHECK (mount_bolt_size IN ('M1.4', 'M2', 'M3')),
    shaft_type VARCHAR(50) NOT NULL CHECK (shaft_type IN ('5mm', 'T-Mount 1.5mm', 'T-Mount 2mm')),
    max_current_a NUMERIC(5, 2)
);

CREATE TABLE IF NOT EXISTS propeller (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    diameter_mm NUMERIC(5, 2) NOT NULL,
    pitch_mm NUMERIC(5, 2) NOT NULL,
    blade_count INT NOT NULL,
    mount_type VARCHAR(50) NOT NULL CHECK (mount_type IN ('5mm Hole', 'T-Mount M2')),
    recommended_stator TEXT[]
);

CREATE TABLE IF NOT EXISTS flight_controller (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    mount_pattern_mm VARCHAR(30) NOT NULL,
    mcu_processor VARCHAR(50) NOT NULL CHECK (mcu_processor IN ('F405', 'F411', 'F722', 'H743')),
    gyro_sensor VARCHAR(50),
    input_voltage_min_v NUMERIC(4, 2) NOT NULL,
    input_voltage_max_v NUMERIC(4, 2) NOT NULL,
    bec_outputs JSONB,
    uart_connections JSONB,
    esc_interface VARCHAR(50) NOT NULL CHECK (esc_interface IN ('8-pin Plug', 'Direct Solder')),
    supported_firmware VARCHAR(50)[] CHECK (supported_firmware <@ ARRAY['Betaflight', 'INAV', 'ArduPilot', 'KISS', 'EmuFlight', 'FlightOne']::VARCHAR[])
);

CREATE TABLE IF NOT EXISTS esc (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    is_integrated BOOLEAN DEFAULT FALSE,
    form_factor VARCHAR(50) NOT NULL CHECK (form_factor IN ('4-in-1', 'Individual')),
    mount_pattern_mm VARCHAR(30),
    continuous_current_a NUMERIC(5, 2) NOT NULL,
    burst_current_a NUMERIC(5, 2),
    input_voltage_min_v NUMERIC(4, 2) NOT NULL,
    input_voltage_max_v NUMERIC(4, 2) NOT NULL,
    firmware_protocol VARCHAR(50) NOT NULL CHECK (firmware_protocol IN ('BLHeli_S', 'BLHeli_32', 'AM32', 'Bluejay'))
);

CREATE TABLE IF NOT EXISTS battery (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    chemistry_id INT NOT NULL REFERENCES battery_chemistry(id),
    cell_count_s INT NOT NULL CHECK (cell_count_s > 0),
    cell_count_p INT NOT NULL DEFAULT 1 CHECK (cell_count_p > 0),
    capacity_mah INT NOT NULL,
    continuous_c_rating INT,
    connector_type VARCHAR(50) NOT NULL REFERENCES battery_connector(id),
    length_mm NUMERIC(5, 2),
    width_mm NUMERIC(5, 2),
    height_mm NUMERIC(5, 2)
);

CREATE TABLE IF NOT EXISTS vtx (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    vtx_ecosystem VARCHAR(50) NOT NULL REFERENCES vtx_ecosystem(id),
    video_connection_standard VARCHAR(50),
    mount_pattern_mm VARCHAR(30),
    input_voltage_min_v NUMERIC(4, 2) NOT NULL,
    input_voltage_max_v NUMERIC(4, 2) NOT NULL,
    max_power_mw INT,
    antenna_count INT NOT NULL DEFAULT 1,
    antenna_connector VARCHAR(50) REFERENCES antenna_connector(id),
    included_antenna_polarization VARCHAR(50) REFERENCES antenna_polarization(id)
);

CREATE TABLE IF NOT EXISTS camera (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    vtx_ecosystem VARCHAR(50) NOT NULL REFERENCES vtx_ecosystem(id),
    video_connection_standard VARCHAR(50),
    width_mm NUMERIC(5, 2) NOT NULL,
    height_mm NUMERIC(5, 2) NOT NULL,
    depth_mm NUMERIC(5, 2) NOT NULL,
    input_voltage_min_v NUMERIC(4, 2) NOT NULL,
    input_voltage_max_v NUMERIC(4, 2) NOT NULL,
    mounting_screw_size VARCHAR(50) NOT NULL CHECK (mounting_screw_size IN ('M1.4', 'M2', 'M3')),
    aspect_ratio VARCHAR(50) NOT NULL CHECK (aspect_ratio IN ('4:3', '16:9', 'Switchable'))
);

CREATE TABLE IF NOT EXISTS receiver (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    rf_protocol VARCHAR(50) NOT NULL REFERENCES rf_protocol(id),
    output_protocol VARCHAR(50) NOT NULL CHECK (output_protocol IN ('CRSF', 'SBUS', 'F.Port')),
    input_voltage_min_v NUMERIC(4, 2) NOT NULL,
    input_voltage_max_v NUMERIC(4, 2) NOT NULL,
    antenna_count INT NOT NULL DEFAULT 1,
    antenna_connector VARCHAR(50) REFERENCES antenna_connector(id),
    included_antenna_polarization VARCHAR(50) REFERENCES antenna_polarization(id)
);

CREATE TABLE IF NOT EXISTS gps_module (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    chipset VARCHAR(50) NOT NULL,
    has_compass BOOLEAN DEFAULT FALSE,
    input_voltage_min_v NUMERIC(4, 2) NOT NULL,
    input_voltage_max_v NUMERIC(4, 2) NOT NULL
);

CREATE TABLE IF NOT EXISTS antenna (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    frequency_band VARCHAR(50) NOT NULL CHECK (frequency_band IN ('5.8GHz', '2.4GHz', '900MHz', '1.3GHz')),
    antenna_polarization VARCHAR(50) NOT NULL REFERENCES antenna_polarization(id),
    connector VARCHAR(50) NOT NULL REFERENCES antenna_connector(id),
    antenna_style VARCHAR(50) NOT NULL,
    gain_dbi NUMERIC(4, 2),
    cable_length_mm NUMERIC(5, 2)
);

CREATE TABLE IF NOT EXISTS transmitter (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    rf_protocol VARCHAR(50) NOT NULL REFERENCES rf_protocol(id),
    external_bay_type VARCHAR(50) NOT NULL CHECK (external_bay_type IN ('JR Micro', 'Lite/Nano', 'None')),
    operating_system VARCHAR(50) NOT NULL CHECK (operating_system IN ('EdgeTX', 'OpenTX', 'Ethos')),
    antenna_count INT NOT NULL DEFAULT 1,
    antenna_connector VARCHAR(50) REFERENCES antenna_connector(id),
    included_antenna_polarization VARCHAR(50) REFERENCES antenna_polarization(id)
);

CREATE TABLE IF NOT EXISTS goggles (
    id INT PRIMARY KEY REFERENCES hardware_component(id) ON DELETE CASCADE,
    vtx_ecosystem VARCHAR(50) NOT NULL REFERENCES vtx_ecosystem(id),
    antenna_count INT NOT NULL DEFAULT 1,
    antenna_connector VARCHAR(50) REFERENCES antenna_connector(id),
    included_antenna_polarization VARCHAR(50) REFERENCES antenna_polarization(id)
);

-- ============================================================================
-- 4. SUB-ASSEMBLIES & LOADOUTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS vtx_configuration (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    vtx_id INT NOT NULL REFERENCES vtx(id) ON DELETE RESTRICT,
    antenna_ids INT[], -- Foreign keys referencing antenna(id), allows duplicates
    loadout_name VARCHAR(150)
);

CREATE TABLE IF NOT EXISTS receiver_configuration (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    receiver_id INT NOT NULL REFERENCES receiver(id) ON DELETE RESTRICT,
    antenna_ids INT[], -- Foreign keys referencing antenna(id), allows duplicates
    loadout_name VARCHAR(150)
);

-- ============================================================================
-- 5. CORE APPLICATION ENTITY (BUILDS WITH COMPATIBILITY ENVELOPES)
-- ============================================================================

CREATE TABLE IF NOT EXISTS build (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    crash_resistance_rating INT CHECK (crash_resistance_rating BETWEEN 1 AND 10),
    description TEXT,
    is_verified BOOLEAN DEFAULT TRUE,

    tag_ids VARCHAR(50)[], -- References tag(id)

    misc_weight_g NUMERIC(6, 2), -- covers screws, wires, mounts, etc.

    -- Hard Hardware Component Foreign Keys
    frame_id INT NOT NULL REFERENCES frame(id) ON DELETE RESTRICT,
    motor_id INT NOT NULL REFERENCES motor(id) ON DELETE RESTRICT,
    propeller_id INT NOT NULL REFERENCES propeller(id) ON DELETE RESTRICT,
    fc_id INT NOT NULL REFERENCES flight_controller(id) ON DELETE RESTRICT,
    esc_id INT REFERENCES esc(id) ON DELETE RESTRICT, -- Nullable for AIO boards
    camera_id INT NOT NULL REFERENCES camera(id) ON DELETE RESTRICT,
    gps_id INT REFERENCES gps_module(id) ON DELETE SET NULL,

    -- Sub-Assembly Foreign Keys
    vtx_config_id INT NOT NULL REFERENCES vtx_configuration(id) ON DELETE RESTRICT,
    receiver_config_id INT NOT NULL REFERENCES receiver_configuration(id) ON DELETE RESTRICT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- 6. INDEXES FOR PERFORMANCE
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_hardware_component_link_comp ON hardware_component_link(hardware_component_id);
CREATE INDEX IF NOT EXISTS idx_hardware_component_company_comp ON hardware_component_company(hardware_component_id);
CREATE INDEX IF NOT EXISTS idx_build_frame ON build(frame_id);
CREATE INDEX IF NOT EXISTS idx_build_motor ON build(motor_id);
CREATE INDEX IF NOT EXISTS idx_build_propeller ON build(propeller_id);
CREATE INDEX IF NOT EXISTS idx_build_fc ON build(fc_id);
CREATE INDEX IF NOT EXISTS idx_build_esc ON build(esc_id);
CREATE INDEX IF NOT EXISTS idx_build_vtx_config ON build(vtx_config_id);
CREATE INDEX IF NOT EXISTS idx_build_rx_config ON build(receiver_config_id);
CREATE INDEX IF NOT EXISTS idx_build_tag_ids ON build USING GIN (tag_ids);