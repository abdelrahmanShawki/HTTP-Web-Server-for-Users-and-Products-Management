package data

import (
	"context"
	"database/sql"
	"time"
)

// CreditCard represents a credit card record.
type CreditCard struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	CardToken      string    `json:"card_token"`
	ExpiryDate     time.Time `json:"expiry_date"`
	CardholderName string    `json:"cardholder_name"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreditCardModel wraps a sql.DB connection pool.
type CreditCardModel struct {
	DB *sql.DB
}

// Insert adds a new credit card record to the database.
// It uses a context with timeout to avoid hanging queries and returns
// the generated ID and creation timestamp via the CreditCard struct.
func (m CreditCardModel) Insert(card *CreditCard) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO credit_cards (user_id, card_token, expiry_date, cardholder_name, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`
	err := m.DB.QueryRowContext(ctx, query, card.UserID, card.CardToken, card.ExpiryDate, card.CardholderName).
		Scan(&card.ID, &card.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

// Delete removes a credit card record identified by its ID and associated userID.
// It returns a specific ErrRecordNotFound if no record is deleted.
func (m CreditCardModel) Delete(id, userID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		DELETE FROM credit_cards
		WHERE id = $1 AND user_id = $2
	`
	result, err := m.DB.ExecContext(ctx, query, id, userID)
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
