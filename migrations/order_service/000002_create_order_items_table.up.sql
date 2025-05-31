CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL, -- Merujuk ke ID produk dari Product Service
    quantity INT NOT NULL CHECK (quantity > 0),
    price_at_purchase DECIMAL(10, 2) NOT NULL, -- Harga produk saat dibeli
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- updated_at tidak umum untuk order_items, karena item biasanya tidak diubah setelah order dibuat
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);

-- Seed Data for Order Items for Order1 --
-- Item 1: Product1 (Laptop)
INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase) VALUES
('d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a41', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a31', 1, 1200.50);

-- Item 2: Product2 (Mouse)
INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase) VALUES
('d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a41', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a32', 2, 25.99);