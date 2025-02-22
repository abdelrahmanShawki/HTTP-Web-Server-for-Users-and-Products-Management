CREATE TABLE IF NOT EXISTS orders (
              id SERIAL PRIMARY KEY,
              user_id INTEGER NOT NULL,
              total_amount NUMERIC(10,2) NOT NULL,
              stripe_payment_id VARCHAR(255) NOT NULL,
              status VARCHAR(20) NOT NULL DEFAULT 'pending',
              created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
              FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);