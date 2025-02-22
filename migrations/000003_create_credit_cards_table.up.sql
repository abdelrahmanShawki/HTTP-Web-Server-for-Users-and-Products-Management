CREATE TABLE IF NOT EXISTS credit_cards (
        id SERIAL PRIMARY KEY,
        user_id INTEGER NOT NULL,
        card_token VARCHAR(255) NOT NULL,
        expiry_date DATE NOT NULL,
        cardholder_name VARCHAR(255),
        created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);