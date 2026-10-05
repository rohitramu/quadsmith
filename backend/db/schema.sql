CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE resources (
    uuid UUID PRIMARY KEY,
    id TEXT UNIQUE NOT NULL,
    resource_type TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE users (
    uuid UUID PRIMARY KEY,
    id TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    role TEXT DEFAULT 'USER',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE manufacturers (
    uuid UUID PRIMARY KEY REFERENCES resources(uuid) ON DELETE CASCADE,
    name TEXT UNIQUE NOT NULL,
    website_url TEXT
);

CREATE TABLE components (
    uuid UUID PRIMARY KEY REFERENCES resources(uuid) ON DELETE CASCADE,
    manufacturer_uuid UUID REFERENCES manufacturers(uuid) ON DELETE CASCADE,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    release_date JSONB,
    weight_g DECIMAL,
    data JSONB NOT NULL
);

CREATE TABLE flight_stacks (
    uuid UUID PRIMARY KEY,
    id TEXT UNIQUE NOT NULL,
    flight_controller_uuid UUID REFERENCES components(uuid),
    esc_uuid UUID REFERENCES components(uuid),
    loadout_name TEXT
);

CREATE TABLE vtx_configurations (
    uuid UUID PRIMARY KEY,
    id TEXT UNIQUE NOT NULL,
    vtx_uuid UUID REFERENCES components(uuid),
    loadout_name TEXT
);

CREATE TABLE receiver_configurations (
    uuid UUID PRIMARY KEY,
    id TEXT UNIQUE NOT NULL,
    receiver_uuid UUID REFERENCES components(uuid),
    loadout_name TEXT
);

CREATE TABLE builds (
    uuid UUID PRIMARY KEY REFERENCES resources(uuid) ON DELETE CASCADE,
    user_uuid UUID REFERENCES users(uuid) ON DELETE SET NULL,
    
    frame_uuid UUID REFERENCES components(uuid),
    motor_uuid UUID REFERENCES components(uuid),
    propeller_uuid UUID REFERENCES components(uuid),
    camera_uuid UUID REFERENCES components(uuid),
    gps_uuid UUID REFERENCES components(uuid),
    
    flight_stack_uuid UUID REFERENCES flight_stacks(uuid),
    vtx_config_uuid UUID REFERENCES vtx_configurations(uuid),
    receiver_config_uuid UUID REFERENCES receiver_configurations(uuid),
    
    data JSONB NOT NULL
);

CREATE TABLE tags (
    uuid UUID PRIMARY KEY REFERENCES resources(uuid) ON DELETE CASCADE,
    name TEXT UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE build_tags (
    build_uuid UUID REFERENCES builds(uuid) ON DELETE CASCADE,
    tag_uuid UUID REFERENCES tags(uuid) ON DELETE CASCADE,
    PRIMARY KEY (build_uuid, tag_uuid)
);

-- GIN Index for dynamic JSONB component querying
CREATE INDEX idx_components_data_gin ON components USING GIN (data jsonb_path_ops);
