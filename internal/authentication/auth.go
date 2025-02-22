package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))
var TokenExpiry = time.Hour * 72

func GenerateToken(userID int64, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,                             // subject: user ID
		"role": role,                               // include role for authorization checks
		"exp":  time.Now().Add(TokenExpiry).Unix(), // expiry time
		"iat":  time.Now().Unix(),                  // issued at
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC (HS256).
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
}
