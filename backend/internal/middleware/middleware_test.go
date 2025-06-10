package middleware

import (
	"backend/internal/types"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestAdminAuthMiddleware tests the AdminAuthMiddleware function
func TestAdminAuthMiddleware(t *testing.T) {
	// Initialize jwtKey for testing
	jwtKey = []byte("test-secret-key")

	// Test cases
	tests := []struct {
		name           string
		setupCookie    func(*http.Request)
		expectedStatus int
		isAdmin        bool
	}{
		{
			name: "Valid admin token",
			setupCookie: func(req *http.Request) {
				claims := &types.Claims{
					UserID:   "admin123",
					Username: "admin",
					Email:    "admin@example.com",
					IsAdmin:  true,
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString(jwtKey)
				req.AddCookie(&http.Cookie{
					Name:  "authToken",
					Value: tokenString,
				})
			},
			expectedStatus: http.StatusOK,
			isAdmin:        true,
		},
		{
			name: "Valid non-admin token",
			setupCookie: func(req *http.Request) {
				claims := &types.Claims{
					UserID:   "user123",
					Username: "user",
					Email:    "user@example.com",
					IsAdmin:  false,
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString(jwtKey)
				req.AddCookie(&http.Cookie{
					Name:  "authToken",
					Value: tokenString,
				})
			},
			expectedStatus: http.StatusForbidden,
			isAdmin:        false,
		},
		{
			name:           "No auth token",
			setupCookie:    func(req *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
			isAdmin:        false,
		},
		{
			name: "Expired token",
			setupCookie: func(req *http.Request) {
				claims := &types.Claims{
					UserID:   "admin123",
					Username: "admin",
					Email:    "admin@example.com",
					IsAdmin:  true,
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString(jwtKey)
				req.AddCookie(&http.Cookie{
					Name:  "authToken",
					Value: tokenString,
				})
			},
			expectedStatus: http.StatusBadRequest,
			isAdmin:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test handler that will be wrapped by the middleware
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if claims were added to context
				claims, ok := r.Context().Value("claims").(*types.Claims)
				if !ok {
					t.Error("Claims not added to context")
				} else if claims.IsAdmin != tc.isAdmin {
					t.Errorf("Expected IsAdmin to be %v, got %v", tc.isAdmin, claims.IsAdmin)
				}
				w.WriteHeader(http.StatusOK)
			})

			// Wrap the test handler with the middleware
			handler := AdminAuthMiddleware(testHandler)

			// Create a test request
			req := httptest.NewRequest("GET", "/admin/test", nil)
			tc.setupCookie(req)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Call the handler
			handler.ServeHTTP(w, req)

			// Check the status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
