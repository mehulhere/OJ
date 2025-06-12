package main

import (
	"log"
	"net/http"
	"os"

	"backend/internal/database"
	"backend/internal/handlers"
	"backend/internal/middleware"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file in main.go")
	}

	// Get MongoDB URI from environment
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI not set in environment variables")
	}

	// Initialize MongoDB connection
	err = database.ConnectDB(mongoURI)
	if err != nil {
		log.Fatal(err)
	}
	defer database.DisconnectDB()

	// Set JWT key
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Fatal("JWT_SECRET_KEY not set in environment variables")
	}
	// jwtKey := []byte(secret)

	// Define routes

	// Handle OPTIONS requests globally
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all responses
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight for 24 hours

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// For non-OPTIONS requests that don't match any other routes
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// For the root path, show a simple response
		w.Write([]byte("Online Judge API Server"))
	})

	http.HandleFunc("/register", middleware.WithCORS(handlers.RegisterHandler))
	http.HandleFunc("/login", middleware.WithCORS(handlers.LoginHandler))
	http.HandleFunc("/api/auth-status", middleware.WithCORS(handlers.AuthStatusHandler))
	http.HandleFunc("/problems", middleware.WithCORS(handlers.GetProblemsHandler))
	http.HandleFunc("/problems/", middleware.WithCORS(handlers.GetProblemHandler))
	http.HandleFunc("/execute", middleware.WithCORS(middleware.JWTAuthMiddleware(handlers.ExecuteCodeHandler)))
	http.HandleFunc("/testcases", middleware.WithCORS(middleware.JWTAuthMiddleware(handlers.AddTestCaseHandler))) // Only for admins

	// Submission routes
	http.HandleFunc("/submissions", middleware.WithCORS(handlers.GetSubmissionsHandler))
	http.HandleFunc("/submissions/", middleware.WithCORS(handlers.GetSubmissionDetailsHandler))
	http.HandleFunc("/submit", middleware.WithCORS(middleware.JWTAuthMiddleware(handlers.SubmitSolutionHandler)))

	// Last code retrieval route
	http.HandleFunc("/last-code", middleware.WithCORS(middleware.JWTAuthMiddleware(handlers.GetLastCodeHandler)))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s...", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
