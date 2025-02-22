ALTER TABLE users
    ADD CONSTRAINT chk_user_role CHECK (role IN ('user', 'admin'));

ALTER TABLE products
    ADD CONSTRAINT chk_price_positive CHECK (price > 0),
    ADD CONSTRAINT chk_inventory_nonnegative CHECK (inventory_count >= 0);

ALTER TABLE credit_cards
    ADD CONSTRAINT chk_expiry_date CHECK (expiry_date > CURRENT_DATE);

ALTER TABLE orders
    ADD CONSTRAINT chk_total_amount CHECK (total_amount >= 0);

ALTER TABLE order_products
    ADD CONSTRAINT chk_quantity_positive CHECK (quantity > 0),
    ADD CONSTRAINT chk_price_at_purchase_positive CHECK (price_at_purchase > 0);