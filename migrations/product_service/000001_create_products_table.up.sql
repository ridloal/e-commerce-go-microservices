CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock_quantity INT NOT NULL DEFAULT 0, -- Stok sederhana untuk tahap ini
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);

-- Seed Data for Products --
INSERT INTO products (id, name, description, price, stock_quantity) VALUES
('c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a31', 'Awesome Laptop 16GB RAM', 'A very powerful and awesome laptop for all your needs.', 1200.50, 0), -- Stock will be managed by warehouse service
('c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a32', 'Wireless Ergonomic Mouse', 'Comfortable mouse for long working hours.', 25.99, 0),
('c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'Mechanical RGB Keyboard', 'Clicky and responsive keyboard with RGB lights.', 75.00, 0);