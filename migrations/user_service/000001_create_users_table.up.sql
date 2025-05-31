CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone_number VARCHAR(50) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_phone_number ON users(phone_number);

-- Seed Data for Users --
-- Note: Replace 'YOUR_BCRYPT_HASH_FOR_password123' with an actual bcrypt hash.
-- You can generate one using a simple Go program or an online tool.
-- Example for "password123" might be something like: $2a$10$xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
INSERT INTO users (id, email, phone_number, password_hash) VALUES
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'user1@example.com', '081234567890', '$2a$10$hLifD9w1W34viAe7dR6c8upQl.qCrM8uziBWI73Cdi.u540XOwO9C'),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'user2@example.com', '081234567891', '$2a$10$kx5SnuiV6VzN8nJiBjFGqeiNqd4shzucQYaU4vtQEJdmscTXl/nfS');