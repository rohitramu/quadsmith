CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE resources (
    id TEXT PRIMARY KEY,
    resource_type TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    role TEXT DEFAULT 'USER',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE manufacturers (
    id TEXT PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
    name TEXT UNIQUE NOT NULL,
    website_url TEXT
);

CREATE TABLE components (
    id TEXT PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
    manufacturer_id TEXT REFERENCES manufacturers(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    release_year INT,
    release_month INT,
    release_day INT,
    weight_g DECIMAL,
    data JSONB NOT NULL
);

CREATE TABLE flight_stacks (
    id TEXT PRIMARY KEY,
    flight_controller_id TEXT REFERENCES components(id),
    esc_id TEXT REFERENCES components(id),
    loadout_name TEXT
);

CREATE TABLE vtx_configurations (
    id TEXT PRIMARY KEY,
    vtx_id TEXT REFERENCES components(id),
    loadout_name TEXT
);

CREATE TABLE receiver_configurations (
    id TEXT PRIMARY KEY,
    receiver_id TEXT REFERENCES components(id),
    loadout_name TEXT
);

CREATE TABLE builds (
    id TEXT PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    crash_resistance_rating INT,
    description TEXT,
    is_verified BOOLEAN DEFAULT FALSE,
    misc_weight_g DECIMAL,
    
    frame_id TEXT REFERENCES components(id),
    motor_id TEXT REFERENCES components(id),
    propeller_id TEXT REFERENCES components(id),
    camera_id TEXT REFERENCES components(id),
    gps_id TEXT REFERENCES components(id),
    
    flight_stack_id TEXT REFERENCES flight_stacks(id),
    vtx_config_id TEXT REFERENCES vtx_configurations(id),
    receiver_config_id TEXT REFERENCES receiver_configurations(id)
);

CREATE TABLE tags (
    id TEXT PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
    name TEXT UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE build_tags (
    build_id TEXT REFERENCES builds(id) ON DELETE CASCADE,
    tag_id TEXT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (build_id, tag_id)
);
