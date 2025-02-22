ALTER TABLE order_products DROP CONSTRAINT IF EXISTS chk_quantity_positive;
ALTER TABLE order_products DROP CONSTRAINT IF EXISTS chk_price_at_purchase_positive;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS chk_total_amount;

ALTER TABLE credit_cards DROP CONSTRAINT IF EXISTS chk_expiry_date;

ALTER TABLE products DROP CONSTRAINT IF EXISTS chk_price_positive;
ALTER TABLE products DROP CONSTRAINT IF EXISTS chk_inventory_nonnegative;

ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_user_role;