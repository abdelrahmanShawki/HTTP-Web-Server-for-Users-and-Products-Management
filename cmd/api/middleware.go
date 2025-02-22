package main

import (
	"context"
	"github.com/golang-jwt/jwt/v4"
	"interviewTask/internal/authentication"
	"net/http"
	"strings"
)

func (app *application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the token from the Authorization header.
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.invalidCredentialsResponse(w, r)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			app.invalidCredentialsResponse(w, r)
			return
		}
		tokenStr := parts[1]

		// Validate the token.
		token, err := auth.ValidateToken(tokenStr)
		if err != nil || !token.Valid {
			app.invalidCredentialsResponse(w, r)
			return
		}

		// Extract claims.
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			app.invalidCredentialsResponse(w, r)
			return
		}

		// Extract the user ID (as "sub") and role.
		sub, ok := claims["sub"].(float64)
		if !ok {
			app.invalidCredentialsResponse(w, r)
			return
		}
		role, _ := claims["role"].(string)

		// Set the userID and role in the request context for downstream handlers.
		ctx := context.WithValue(r.Context(), "userID", int64(sub))
		ctx = context.WithValue(ctx, "role", role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Retrieve the role from the context.
			role, ok := r.Context().Value("role").(string)
			if !ok || role != requiredRole {
				app.accessDeniedResonse(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
