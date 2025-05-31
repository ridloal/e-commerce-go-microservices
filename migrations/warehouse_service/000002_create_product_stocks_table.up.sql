CREATE TABLE IF NOT EXISTS product_stocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL, -- This ID comes from the Product Service
    quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved_quantity INT NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_warehouse_product UNIQUE (warehouse_id, product_id),
    CONSTRAINT chk_reserved_not_greater_than_quantity CHECK (reserved_quantity <= quantity)
);

CREATE INDEX IF NOT EXISTS idx_product_stocks_warehouse_id ON product_stocks(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_product_stocks_product_id ON product_stocks(product_id);

-- Seed Data for Product Stocks --
-- Product1 (Laptop) in Warehouse1 (Jakarta)
INSERT INTO product_stocks (warehouse_id, product_id, quantity, reserved_quantity) VALUES
('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a21', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a31', 50, 5);

-- Product1 (Laptop) in Warehouse2 (Surabaya)
INSERT INTO product_stocks (warehouse_id, product_id, quantity, reserved_quantity) VALUES
('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a31', 20, 0);

-- Product2 (Mouse) in Warehouse1 (Jakarta)
INSERT INTO product_stocks (warehouse_id, product_id, quantity, reserved_quantity) VALUES
('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a21', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a32', 200, 10);

-- Product3 (Keyboard) in Warehouse1 (Jakarta)
INSERT INTO product_stocks (warehouse_id, product_id, quantity, reserved_quantity) VALUES
('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a21', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 100, 0);

-- Product3 (Keyboard) in Warehouse2 (Surabaya)
INSERT INTO product_stocks (warehouse_id, product_id, quantity, reserved_quantity) VALUES
('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 75, 2);