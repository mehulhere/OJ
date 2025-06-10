package main

import (
	"backend/internal/database"
	"backend/internal/models"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load environment variables
	err := godotenv.Load("../../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
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

	// Get admin details from command line arguments
	if len(os.Args) != 5 {
		fmt.Println("Usage: go run create_admin.go <firstname> <lastname> <username> <password>")
		os.Exit(1)
	}

	firstname := os.Args[1]
	lastname := os.Args[2]
	username := os.Args[3]
	password := os.Args[4]

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Create the admin user
	adminUser := models.User{
		Firstname: firstname,
		Lastname:  lastname,
		Username:  username,
		Email:     username + "@admin.com", // Using username as email for simplicity
		Password:  string(hashedPassword),
		IsAdmin:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Insert the admin user into the database
	usersCollection := database.GetCollection("OJ", "users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := usersCollection.InsertOne(ctx, adminUser)
	if err != nil {
		log.Fatal("Failed to create admin user:", err)
	}

	fmt.Printf("Admin user created successfully with ID: %v\n", result.InsertedID)
}
