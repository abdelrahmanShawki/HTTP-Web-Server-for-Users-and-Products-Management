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
	os.Setenv("JWT_SECRET", "testsecret")
	app := &testApplication{
		application: application{
			logger: dLogger.Logger,
		},
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Role OK"))
	})

	middleware := app.AuthMiddleware(app.RequireRole("admin")(nextHandler))

	tTable := []struct {
		name           string
		tokenRole      string
		tokenID        int
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Non-admin user",
			tokenRole:      "user",
			tokenID:        456,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "access denied",
		},
		{
			name:           "Admin user",
			tokenRole:      "admin",
			tokenID:        789,
			expectedStatus: http.StatusOK,
			expectedBody:   "Role OK",
		},
	}

	for _, tCase := range tTable {
		t.Run(tCase.name, func(t *testing.T) {
			token, err := auth.GenerateToken(int64(tCase.tokenID), tCase.tokenRole)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tCase.expectedStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tCase.expectedBody)
		})
	}
}

// /-----------////////
func TestRecoverMiddleware(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := jsonlog.New(&logBuffer, jsonlog.LevelInfo)

	app := &testApplication{
		application: application{
			logger: logger,
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
