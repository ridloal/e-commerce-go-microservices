CREATE TABLE IF NOT EXISTS warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    location TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_warehouses_name ON warehouses(name);
CREATE INDEX IF NOT EXISTS idx_warehouses_is_active ON warehouses(is_active);

-- Seed Data for Warehouses --
INSERT INTO warehouses (id, name, location, is_active) VALUES
('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a21', 'Main Warehouse Jakarta', 'Jakarta, Indonesia', TRUE),
('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'Secondary Warehouse Surabaya', 'Surabaya, Indonesia', TRUE);