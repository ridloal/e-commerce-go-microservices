CREATE TYPE order_status AS ENUM (
    'PENDING_PAYMENT',
    'AWAITING_SHIPMENT',
    'SHIPPED',
    'DELIVERED',
    'CANCELLED',
    'FAILED',
    'PAYMENT_TIMEOUT',
    'PAYMENT_CONFIRMED'
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL, -- Merujuk ke ID pengguna dari User Service
    total_amount DECIMAL(12, 2) NOT NULL,
    status order_status NOT NULL DEFAULT 'PENDING_PAYMENT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

-- Seed Data for Orders --
-- Order1 by User1
INSERT INTO orders (id, user_id, total_amount, status) VALUES
('d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a41', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 0, 'PENDING_PAYMENT');
-- Total amount will be updated after items are inserted, or calculated by service. For seed, can pre-calculate.
-- Pre-calculated: Product1 (1200.50 * 1) + Product2 (25.99 * 2) = 1200.50 + 51.98 = 1252.48

UPDATE orders SET total_amount = 1252.48 WHERE id = 'd0eebc99-9c0b-4ef8-bb6d-6bb9bd380a41';