package main

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"interviewTask/internal/authentication"
	"net/http"
	"strings"
)

type contextKey string

const (
	userContextKey = contextKey("userId")
	roleContextKey = contextKey("role")
)

func (app *application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the token from the Authorization header.
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			app.logger.PrintInfo(fmt.Sprintf("authrization header is %s ", authHeader), nil)
			app.invalidCredentialsResponse(w, r)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			app.logger.PrintInfo(fmt.Sprintf("parts is %s %s %s %s ", parts[0], parts[1], parts[2], parts[3]), nil)

			app.invalidCredentialsResponse(w, r)
			return
		}
		tokenStr := parts[1]

		// Validate the token.
		token, err := auth.ValidateToken(tokenStr)
		if err != nil || !token.Valid {
			app.logger.PrintInfo(fmt.Sprintf("token is %s ", token), nil)
			app.invalidCredentialsResponse(w, r)
			return
		}

		// Extract claims.
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			app.logger.PrintInfo("claims check ", nil)
			app.invalidCredentialsResponse(w, r)
			return
		}

		// Extract the user ID (as "sub") and role.
		sub, ok := claims["sub"].(float64)
		if !ok {
			app.logger.PrintInfo("claims sub", nil)

			app.invalidCredentialsResponse(w, r)
			return
		}
		role, _ := claims["role"].(string)

		// Set the userID and role in the request context for downstream handlers.
		ctx := context.WithValue(r.Context(), userContextKey, int64(sub))
		ctx = context.WithValue(ctx, roleContextKey, role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Retrieve the role from the context.
			role, ok := r.Context().Value(roleContextKey).(string)
			if !ok || role != requiredRole {
				app.accessDeniedResonse(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
