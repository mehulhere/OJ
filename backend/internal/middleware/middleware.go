package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"backend/internal/types"
	"backend/internal/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var jwtKey []byte

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("WARNING: Error loading .env file in middleware.go.")
	}
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Println("CRITICAL: JWT_SECRET_KEY not found in environment variables during init in middleware.go.")
		return
	}
	jwtKey = []byte(secret)
}

// This is the middleware for the CORS
func WithCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// This is a middleware to validate JWT from HTTP-only cookie
func jwtAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the cookie named "authToken"
		cookie, err := r.Cookie("authToken")
		if err != nil {
			if err == http.ErrNoCookie {
				// If the cookie is not set, return an unauthorized status
				utils.SendJSONError(w, "Unauthorized: No auth token cookie provided", http.StatusUnauthorized)
				return
			}
			// For any other type of error, return a bad request status
			log.Printf("Error getting auth token cookie: %v", err)
			utils.SendJSONError(w, "Bad Request: Error reading auth token cookie", http.StatusBadRequest)
			return
		}

		// Get the JWT string from the cookie
		tokenString := cookie.Value

		// Declare the claims
		claims := &types.Claims{}

		// Parse the JWT
		tkn, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Ensure the token is signed with HS256
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// Return the jwtKey for validation
			// Ensure jwtKey is initialized (e.g., in main or init)
			if jwtKey == nil {
				// This case should ideally not happen if main loads the key correctly
				log.Println("CRITICAL: jwtKey is nil during token validation.")
				return nil, fmt.Errorf("server configuration error")
			}
			return jwtKey, nil
		})

		if err != nil {
			log.Printf("JWT validation failed: %v", err)
			// Handle token parsing or validation errors (e.g., expired, invalid signature)
			utils.SendJSONError(w, "Unauthorized: Invalid or expired auth token", http.StatusUnauthorized)
			return
		}

		// Check if the token is valid (like not expired)
		if !tkn.Valid {
			log.Println("JWT token is not valid after parsing.")
			utils.SendJSONError(w, "Unauthorized: Invalid auth token", http.StatusUnauthorized)
			return
		}

		// Token is valid, call the next handler
		// Optionally, you can add the claims to the request context here
		// ctx := context.WithValue(r.Context(), "claims", claims)
		// next.ServeHTTP(w, r.WithContext(ctx))
		next.ServeHTTP(w, r)
	}
}

// JWTAuthMiddleware checks for a valid JWT token in the request cookie
func JWTAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the authToken cookie
		cookie, err := r.Cookie("authToken")
		if err != nil {
			if err == http.ErrNoCookie {
				utils.SendJSONError(w, "Authentication required. Please log in.", http.StatusUnauthorized)
				return
			}
			utils.SendJSONError(w, "Error reading authentication token.", http.StatusBadRequest)
			return
		}

		// Get the JWT string from the cookie
		tokenStr := cookie.Value

		// Initialize a new instance of `Claims`
		claims := &types.Claims{}

		// Parse the JWT string
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				utils.SendJSONError(w, "Invalid authentication token signature.", http.StatusUnauthorized)
				return
			}
			utils.SendJSONError(w, "Error parsing authentication token.", http.StatusBadRequest)
			return
		}
		if !token.Valid {
			utils.SendJSONError(w, "Invalid authentication token.", http.StatusUnauthorized)
			return
		}

		// Token is valid, call the next handler
		next.ServeHTTP(w, r)
	}
}

// AdminAuthMiddleware checks if the user is authenticated and is an admin
func AdminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the authToken cookie
		cookie, err := r.Cookie("authToken")
		if err != nil {
			if err == http.ErrNoCookie {
				utils.SendJSONError(w, "Authentication required. Please log in.", http.StatusUnauthorized)
				return
			}
			utils.SendJSONError(w, "Error reading authentication token.", http.StatusBadRequest)
			return
		}

		// Get the JWT string from the cookie
		tokenStr := cookie.Value

		// Initialize a new instance of `Claims`
		claims := &types.Claims{}

		// Parse the JWT string
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				utils.SendJSONError(w, "Invalid authentication token signature.", http.StatusUnauthorized)
				return
			}
			utils.SendJSONError(w, "Error parsing authentication token.", http.StatusBadRequest)
			return
		}
		if !token.Valid {
			utils.SendJSONError(w, "Invalid authentication token.", http.StatusUnauthorized)
			return
		}

		// Check if the user is an admin
		if !claims.IsAdmin {
			utils.SendJSONError(w, "Access denied. Admin privileges required.", http.StatusForbidden)
			return
		}

		// Add claims to the request context
		ctx := context.WithValue(r.Context(), "claims", claims)

		// Token is valid and user is admin, call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
