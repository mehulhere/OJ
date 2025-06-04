package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var DB *mongo.Client

func ConnectDB(uri string) error {

	// Set a Timeout of 10s
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	// Set uri and any other options
	clientOptions := options.Client().ApplyURI(uri)

	// Connect to mongo with these options
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v", err)
		return err
	}

	// Ping the MongoDB server to verify the connection
	err = client.Ping(ctx, readpref.Primary()) // readpref.Primary() reads the primary node of the database
	if err != nil {
		log.Printf("Failed to ping MongoDB: %v", err)
		if client != nil {
			disconnectErr := client.Disconnect(context.Background()) // Sent a new context without timeout
			if disconnectErr != nil {
				log.Printf("Failed to disconnect client after ping failure: %v", disconnectErr)
			}
		}
		return err // This returns the ping error
	}

	DB = client
	fmt.Println("Successfully connected to MongoDB!")

	// Ensure unique indexes
	err = EnsureUniqueIndex("OJ", "users", "username")
	if err != nil {
		log.Fatalf("Failed to ensure unique username index: %v", err)
	}

	err = EnsureUniqueIndex("OJ", "users", "email")
	if err != nil {
		log.Fatalf("Failed to ensure unique email index: %v", err)
	}

	return nil
}

// Function to ensure a unique index on a specified field
func EnsureUniqueIndex(dbName string, collectionName string, fieldKey string) error {
	if DB == nil {
		return fmt.Errorf("mongodb client is not initialized")
	}

	collection := DB.Database(dbName).Collection(collectionName)

	// Create a unique index on the specified field
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: fieldKey, Value: 1}}, // Index on the specified field and sort in ascending
		Options: options.Index().SetUnique(true),   // Uniqueness option
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		// Check if the error is due to the index already existing
		if mongo.IsDuplicateKeyError(err) {
			fmt.Printf("Unique index on '%s' already exists.\n", fieldKey)
			return nil // Index already exists, consider it successful
		}
		return fmt.Errorf("failed to create unique index on '%s': %v", fieldKey, err)
	}

	fmt.Printf("Successfully created unique index on '%s'.\n", fieldKey)

	return nil
}

func GetCollection(dbName string, collectionName string) *mongo.Collection {
	if DB == nil {
		log.Fatal("MongoDB client is not initiazed. Call Connect DB first.")
		return nil
	}

	collection := DB.Database(dbName).Collection(collectionName)
	return collection
}

func DisconnectDB() {
	if DB == nil {
		return
	}

	// Creating a 5s timeout for Disconnect too
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := DB.Disconnect(ctx)
	if err != nil {
		log.Fatalf("Failed to disconnect from MongoDB: %v", err)
	}
	fmt.Println("Connection to MongoDB closed.")
}
