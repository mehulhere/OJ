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
	http.HandleFunc("/register", middleware.WithCORS(handlers.RegisterHandler))
	http.HandleFunc("/login", middleware.WithCORS(handlers.LoginHandler))
	http.HandleFunc("/api/auth-status", middleware.WithCORS(handlers.AuthStatusHandler))
	http.HandleFunc("/problems", middleware.WithCORS(handlers.GetProblemsHandler))
	http.HandleFunc("/problems/", middleware.WithCORS(handlers.GetProblemHandler))
	http.HandleFunc("/execute", middleware.JWTAuthMiddleware(middleware.WithCORS(handlers.ExecuteCodeHandler)))
	http.HandleFunc("/testcases", middleware.JWTAuthMiddleware(middleware.WithCORS(handlers.AddTestCaseHandler))) // Only for admins

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
