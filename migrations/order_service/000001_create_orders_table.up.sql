CREATE TYPE order_status AS ENUM (
    'PENDING_PAYMENT',
    'AWAITING_SHIPMENT',
    'SHIPPED',
    'DELIVERED',
    'CANCELLED',
    'FAILED',
    'PAYMENT_TIMEOUT'
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