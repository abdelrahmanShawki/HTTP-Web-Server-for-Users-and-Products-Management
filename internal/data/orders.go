package data

import (
	"context"
	"database/sql"
	"time"
)

// Order represents an order placed by a user.
type Order struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	TotalAmount     float64   `json:"total_amount"`
	StripePaymentID string    `json:"stripe_payment_id"`
	CreatedAt       time.Time `json:"created_at"`
}

// OrderProduct represents a record in the order_products table.
type OrderProduct struct {
	OrderID         int64   `json:"order_id"`
	ProductID       int64   `json:"product_id"`
	Quantity        int     `json:"quantity"`
	PriceAtPurchase float64 `json:"price_at_purchase"`
}

//the next Dtos decription :   (composition)

// PurchaseHistory represents a full order with its product details. works as a DTO
type PurchaseHistory struct {
	OrderID         int64                `json:"order_id"`
	UserID          int64                `json:"user_id"`
	TotalAmount     float64              `json:"total_amount"`
	StripePaymentID string               `json:"stripe_payment_id"`
	CreatedAt       time.Time            `json:"created_at"`
	Products        []OrderProductDetail `json:"products"`
}

// OrderProductDetail works as a DTO ,
type OrderProductDetail struct {
	ProductID          int64     `json:"product_id"`
	Quantity           int       `json:"quantity"`
	PriceAtPurchase    float64   `json:"price_at_purchase"`
	ProductName        string    `json:"product_name"`
	ProductDescription string    `json:"product_description"`
	ProductPrice       float64   `json:"product_price"`
	InventoryCount     int       `json:"inventory_count"`
	ProductCreatedAt   time.Time `json:"product_created_at"`
}

type OrdersModel struct {
	DB *sql.DB
}

// Create inserts a new order and its associated order_products records atomically.
func (m OrdersModel) Create(order *Order, orderProducts []OrderProduct) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Begin a transaction.
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Insert the order record.
	orderQuery := `
		INSERT INTO orders (user_id, total_amount, stripe_payment_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	err = tx.QueryRowContext(ctx, orderQuery, order.UserID, order.TotalAmount, order.StripePaymentID).
		Scan(&order.ID, &order.CreatedAt)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert each order_products record.
	orderProductQuery := `
		INSERT INTO order_products (order_id, product_id, quantity, price_at_purchase)
		VALUES ($1, $2, $3, $4)`

	for _, op := range orderProducts {
		_, err = tx.ExecContext(ctx, orderProductQuery, order.ID, op.ProductID, op.Quantity, op.PriceAtPurchase)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// GetPurchaseHistory retrieves all orders and their associated product details for a given user.
// It uses a join query to combine orders, order_products, and products data.
func (m OrdersModel) GetPurchaseHistory(userID int64) ([]PurchaseHistory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
	SELECT 
		o.id, o.user_id, o.total_amount, o.stripe_payment_id, o.created_at,
		op.product_id, op.quantity, op.price_at_purchase,
		p.name, p.description, p.price, p.inventory_count, p.created_at as product_created_at
	FROM orders o
	JOIN order_products op ON o.id = op.order_id
	JOIN products p ON op.product_id = p.id
	WHERE o.user_id = $1
	ORDER BY o.created_at DESC, op.product_id;
	`

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Use a map to group products by order.
	historyMap := make(map[int64]*PurchaseHistory)

	for rows.Next() {
		var orderID int64
		var userID int64
		var totalAmount float64
		var stripePaymentID string
		var orderCreatedAt time.Time

		var productID int64
		var quantity int
		var priceAtPurchase float64
		var productName string
		var productDescription string
		var productPrice float64
		var inventoryCount int
		var productCreatedAt time.Time

		err = rows.Scan(
			&orderID, &userID, &totalAmount, &stripePaymentID, &orderCreatedAt,
			&productID, &quantity, &priceAtPurchase,
			&productName, &productDescription, &productPrice, &inventoryCount, &productCreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Check if we've already created a PurchaseHistory for this order.
		history, exists := historyMap[orderID]
		if !exists {
			history = &PurchaseHistory{
				OrderID:         orderID,
				UserID:          userID,
				TotalAmount:     totalAmount,
				StripePaymentID: stripePaymentID,
				CreatedAt:       orderCreatedAt,
				Products:        []OrderProductDetail{},
			}
			historyMap[orderID] = history
		}

		// Append product details.
		productDetail := OrderProductDetail{
			ProductID:          productID,
			Quantity:           quantity,
			PriceAtPurchase:    priceAtPurchase,
			ProductName:        productName,
			ProductDescription: productDescription,
			ProductPrice:       productPrice,
			InventoryCount:     inventoryCount,
			ProductCreatedAt:   productCreatedAt,
		}
		history.Products = append(history.Products, productDetail)
	} // loop ends here .

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Convert the map to a slice.
	result := make([]PurchaseHistory, 0, len(historyMap))
	for _, h := range historyMap {
		result = append(result, *h)
	}

	return result, nil
}

// UpdateStatusByStripePaymentID updates the status of an order based on its Stripe PaymentIntent ID.
func (m OrdersModel) UpdateStatusByStripePaymentID(paymentIntentID, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE stripe_payment_id = $2
	`
	result, err := m.DB.ExecContext(ctx, query, status, paymentIntentID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}
