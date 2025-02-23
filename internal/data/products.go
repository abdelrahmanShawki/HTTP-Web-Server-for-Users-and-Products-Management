package data

import (
	"context"
	"database/sql"
	"fmt"
	"interviewTask/internal/validator"
	"time"
)

// Product represents a product in the catalog.
type Product struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Price          float64   `json:"price"`
	InventoryCount int       `json:"inventory_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ProductSale represents aggregated sales information for a product.
type ProductSale struct {
	ProductID     int64   `json:"product_id"`
	Name          string  `json:"name"`
	TotalQuantity int     `json:"total_quantity"`
	TotalRevenue  float64 `json:"total_revenue"`
}

// ProductModel wraps a sql.DB connection pool.
type ProductModel struct {
	DB *sql.DB
}

// GetAll retrieves all products.
func (m ProductModel) GetAll() ([]Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, name, description, price, inventory_count, created_at, updated_at
		FROM products
		ORDER BY created_at`
	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		err = rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.InventoryCount, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

// GetByID retrieves a single product by its ID.
func (m ProductModel) GetByID(id int64) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, name, description, price, inventory_count, created_at, updated_at
		FROM products
		WHERE id = $1`
	var p Product
	err := m.DB.QueryRowContext(ctx, query, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.InventoryCount, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return &p, nil
}

// Create inserts a new product into the database.
func (m ProductModel) Create(p *Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO products (name, description, price, inventory_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	return m.DB.QueryRowContext(ctx, query, p.Name, p.Description, p.Price, p.InventoryCount).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

// Update modifies an existing product.
func (m ProductModel) Update(p *Product) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		UPDATE products
		SET name = $1, description = $2, price = $3, inventory_count = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at`
	return m.DB.QueryRowContext(ctx, query, p.Name, p.Description, p.Price, p.InventoryCount, p.ID).
		Scan(&p.UpdatedAt)
}

// Delete removes a product from the database.
func (m ProductModel) Delete(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM products WHERE id = $1`
	_, err := m.DB.ExecContext(ctx, query, id)
	return err
}

// SalesFiltering retrieves product sales data filtered by a time period or username.
// Note: This method assumes you have an orders and order_products table with relevant fields.
func (m ProductModel) SalesFiltering(from, to time.Time, username string) ([]ProductSale, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build the query dynamically based on provided filters.
	query := `
	SELECT 
		p.id,
		p.name,
		COALESCE(SUM(op.quantity), 0) AS total_quantity,
		COALESCE(SUM(op.quantity * op.price_at_purchase), 0) AS total_revenue
	FROM products p
	LEFT JOIN order_products op ON p.id = op.product_id
	LEFT JOIN orders o ON op.order_id = o.id
	LEFT JOIN users u ON o.user_id = u.id
	WHERE o.created_at BETWEEN $1 AND $2`
	args := []interface{}{from, to}

	if username != "" {
		username = validator.SanitizeString(username)
		query += ` AND u.first_name ILIKE $3`
		args = append(args, fmt.Sprintf("%%%s%%", username)) // %%%% to match any string contnais the username
		// used with Ilike operator , Ilike perform case-insensitive pattern match
	}

	query += ` GROUP BY p.id, p.name ORDER BY total_revenue DESC`

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sales []ProductSale
	for rows.Next() {
		var ps ProductSale
		err = rows.Scan(&ps.ProductID, &ps.Name, &ps.TotalQuantity, &ps.TotalRevenue)
		if err != nil {
			return nil, err
		}
		sales = append(sales, ps)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sales, nil
}

func ValidateProduct(v *validator.Validator, product *Product) {
	v.Check(product.Name != "", "name", "must be provided")
	v.Check(len(product.Name) <= 255, "name", "must not exceed 255 characters")
	v.Check(product.Description != "", "description", "must be provided")
	v.Check(len(product.Description) > 40 && len(product.Description) < 1200, "description", "description must be between 40 and 1200 character long")
	v.Check(product.Price > 0, "price", "must be a positive value")
	v.Check(product.InventoryCount >= 0, "inventory_count", "must be a non-negative value")
}
