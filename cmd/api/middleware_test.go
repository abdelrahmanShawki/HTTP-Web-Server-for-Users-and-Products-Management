package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"interviewTask/internal/assert"
	auth "interviewTask/internal/authentication"
	"interviewTask/internal/jsonlog"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func (app *testApplication) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "invalid credentials", http.StatusUnauthorized)
}

func (app *testApplication) accessDeniedResonse(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "access denied", http.StatusForbidden)
}

func (app *testApplication) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, "server error", http.StatusInternalServerError)
}

type dummyLogger struct {
	*jsonlog.Logger
}

func newDummyLogger() *dummyLogger {
	logger := jsonlog.New(io.Discard, jsonlog.LevelInfo)
	return &dummyLogger{Logger: logger}
}

// Override PrintInfo to do nothing.
func (l *dummyLogger) PrintInfo(msg string, data map[string]string) {}

// Override PrintError to do nothing.
func (l *dummyLogger) PrintError(err error, data map[string]string) {}

type testApplication struct {
	application
}

func TestAuthMiddleware(t *testing.T) {
	dLogger := newDummyLogger()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		t.Fatalf("JWT_SECRET not configured")
	}

	app := &testApplication{
		application: application{
			logger: dLogger.Logger,
		},
	}

	// Dummy next handler that simply writes "OK"
	dummyNextHandler := http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		writer.WriteHeader(http.StatusOK)
		writer.Write([]byte("OK"))
	})

	middleware := app.AuthMiddleware(dummyNextHandler)

	// Generate a valid token for the valid token test.
	validToken, err := auth.GenerateToken(123, "user")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Generate an expired token (expiration time in the past).
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  123,
		"role": "user",
		"exp":  time.Now().Add(-time.Hour).Unix(),
		"iat":  time.Now().Add(-2 * time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	// Generate a token missing the "sub" claim.
	tokenMissingSub, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": "user",
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to generate token missing sub: %v", err)
	}

	// Generate a token with "sub" of wrong type (string instead of a number).
	tokenSubWrongType, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "123", // wrong type!
		"role": "user",
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to generate token with wrong type for sub: %v", err)
	}

	// Generate a token missing the "role" claim.
	tokenMissingRole, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": 123,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to generate token missing role: %v", err)
	}

	// Define test cases including the new edge cases.
	tTable := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Malformed auth header",
			authHeader:     "BadHeader",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Expired token",
			authHeader:     "Bearer " + expiredToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Token missing sub claim",
			authHeader:     "Bearer " + tokenMissingSub,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Token with sub wrong type",
			authHeader:     "Bearer " + tokenSubWrongType,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Token missing role (should pass auth)",
			authHeader:     "Bearer " + tokenMissingRole,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tCase := range tTable {
		t.Run(tCase.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tCase.authHeader != "" {
				req.Header.Set("Authorization", tCase.authHeader)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			// Directly compare the integer status codes.
			assert.Equal(t, tCase.expectedStatus, rr.Code)

			// For the OK case, also check that the response body contains "OK". - > next handler write it go to line 68
			if tCase.expectedStatus == http.StatusOK {
				assert.Contains(t, rr.Body.String(), "OK")
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	dLogger := newDummyLogger()
	app := &testApplication{
		application: application{
			logger: dLogger.Logger,
		},
	}
	// dummy next handler writes "Role OK"
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Role OK"))
	})
	// Wrap with RequireRole middleware that requires "admin" role.
	requireAdmin := app.RequireRole("admin")(nextHandler)

	// Create a request with a valid token for a non-admin user.
	token, err := auth.GenerateToken(456, "user")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	// To simulate the effect of the AuthMiddleware, manually add role to the context.
	// In real usage, AuthMiddleware would add the role to the context.
	ctx := req.Context()
	ctx = contextWithRole(ctx, "user")
	req = req.WithContext(ctx)

	requireAdmin.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusText(http.StatusForbidden), http.StatusText(rr.Code))

	// Now test with an admin token.
	adminToken, err := auth.GenerateToken(789, "admin")
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	// Manually add the role "admin" to context.
	ctx = req.Context()
	ctx = contextWithRole(ctx, "admin")
	req = req.WithContext(ctx)

	rr = httptest.NewRecorder()
	requireAdmin.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusText(http.StatusOK), http.StatusText(rr.Code))
	assert.Contains(t, rr.Body.String(), "Role OK")
}

// contextWithRole is a helper to add a role to a context (like AuthMiddleware does).
func contextWithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleContextKey, role)
}

// /-----------////////
func TestRecoverMiddleware(t *testing.T) {
	dLogger := newDummyLogger()
	app := &testApplication{
		application: application{
			logger: dLogger.Logger,
		},
	}

	// Create a handler that panics.
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})
	recoverMiddleware := app.RecoverMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	recoverMiddleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusText(http.StatusInternalServerError), http.StatusText(rr.Code))
	// Optionally, check that the response body contains the error message.
	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	body = bytes.TrimSpace(body)
	assert.Contains(t, string(body), "server error")
}
