package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"interviewTask/internal/validator"
)

// User represents a user record in the database.
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Password  password  `json:"-"` // omit from JSON responses
	Role      string    `json:"role"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// password encapsulates both the plaintext and hashed password.
type password struct {
	plaintext *string
	hash      []byte
}

// Set hashes the given plaintext password and stores it.
func (p *password) Set(plain string) error {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	p.plaintext = &plain
	p.hash = h
	return nil
}

// Matches compares a provided plaintext password against the stored hash.
func (p *password) Matches(plain string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plain))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UserModel wraps a sql.DB connection pool and provides methods for user data access.
type UserModel struct {
	DB *sql.DB
}

// Insert adds a new user to the database and updates the User struct with its ID and timestamps.
func (m UserModel) Insert(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO users (email, password_hash, role, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`
	err := m.DB.QueryRowContext(ctx, query,
		user.Email,
		string(user.Password.hash),
		user.Role,
		user.FirstName,
		user.LastName,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		// Check for duplicate email error (PostgreSQL error code 23505).
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrDuplicateEmail
		}
		return err
	}
	return nil
}

// GetByEmail retrieves a user from the database by their email address.
func (m UserModel) GetByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, email, password_hash, role, first_name, last_name, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var user User
	var passwordHash string
	err := m.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&passwordHash,
		&user.Role,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	user.Password.hash = []byte(passwordHash)
	return &user, nil
}

func (m UserModel) GetByID(id int64) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		SELECT id, email, password_hash, role, first_name, last_name, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var user User
	var passwordHash string
	err := m.DB.QueryRowContext(ctx, query, id).
		Scan(&user.ID, &user.Email, &passwordHash, &user.Role, &user.FirstName, &user.LastName, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	user.Password.hash = []byte(passwordHash)
	return &user, nil
}

// ValidateUser uses the provided validator to enforce rules on the User fields.
func ValidateUser(v *validator.Validator, user *User) {
	// Ensure email is provided and matches a valid email format.
	v.Check(user.Email != "", "email", "must be provided")
	v.Check(validator.Matches(user.Email, validator.EmailRX), "email", "must be a valid email address")

	// Validate password length if the plaintext version is set.
	if user.Password.plaintext != nil {
		v.Check(len(*user.Password.plaintext) >= 8, "password", "must be at least 8 characters long")
		v.Check(len(*user.Password.plaintext) <= 72, "password", "must not exceed 72 characters")
	}

	// Check that role is either "user" or "admin".
	v.Check(user.Role == "user" || user.Role == "admin", "role", "must be either user or admin")

	if user.FirstName != "" {
		v.Check(len(user.FirstName) <= 100, "first_name", "must not exceed 100 characters")
	}
	if user.LastName != "" {
		v.Check(len(user.LastName) <= 100, "last_name", "must not exceed 100 characters")
	}
}
