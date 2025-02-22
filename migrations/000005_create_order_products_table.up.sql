CREATE TABLE IF NOT EXISTS order_products (
                  order_id INTEGER NOT NULL,
                  product_id INTEGER NOT NULL,
                  quantity INTEGER NOT NULL DEFAULT 1,
                  price_at_purchase NUMERIC(10,2) NOT NULL,
                  PRIMARY KEY (order_id, product_id),
                  FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
                  FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);