package main

import (
	"backend/internal/database"
	"backend/internal/models"
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	// Load environment variables
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: Error loading .env file, trying to create a default one")

		// Create a default .env file
		defaultEnv := `MONGO_URI=mongodb://localhost:27017
JWT_SECRET_KEY=mysecretkey123fortests
PORT=8080`

		err = os.WriteFile(".env", []byte(defaultEnv), 0644)
		if err != nil {
			log.Printf("Error creating default .env file: %v", err)
		} else {
			log.Println("Created default .env file")
			err = godotenv.Load(".env")
			if err != nil {
				log.Fatalf("Error loading created .env file: %v", err)
			}
		}
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

	// Define test cases for TS001 problem
	problemID := "TS001"

	// First, get the problem DB ID
	problemsCollection := database.GetCollection("OJ", "problems")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var problem models.Problem
	err = problemsCollection.FindOne(ctx, bson.M{"problem_id": problemID}).Decode(&problem)
	if err != nil {
		log.Fatalf("Error finding problem %s: %v", problemID, err)
	}

	log.Printf("Found problem: %s - %s (ID: %s)", problem.ProblemID, problem.Title, problem.ID.Hex())

	// Define test cases
	testCases := []models.TestCase{
		{
			ProblemDBID:    problem.ID,
			Input:          "nums = [2,7,11,15], target = 9",
			ExpectedOutput: "[0,1]",
			IsSample:       true,
			Points:         10,
			Notes:          "Basic case",
			SequenceNumber: 1,
			CreatedAt:      time.Now(),
		},
		{
			ProblemDBID:    problem.ID,
			Input:          "nums = [3,2,4], target = 6",
			ExpectedOutput: "[1,2]",
			IsSample:       true,
			Points:         10,
			Notes:          "Numbers not in order",
			SequenceNumber: 2,
			CreatedAt:      time.Now(),
		},
		{
			ProblemDBID:    problem.ID,
			Input:          "nums = [3,3], target = 6",
			ExpectedOutput: "[0,1]",
			IsSample:       true,
			Points:         10,
			Notes:          "Same element",
			SequenceNumber: 3,
			CreatedAt:      time.Now(),
		},
		{
			ProblemDBID:    problem.ID,
			Input:          "nums = [1,2,3,4,5,6,7,8,9,10], target = 19",
			ExpectedOutput: "[8,9]",
			IsSample:       false,
			Points:         20,
			Notes:          "Larger array",
			SequenceNumber: 4,
			CreatedAt:      time.Now(),
		},
		{
			ProblemDBID:    problem.ID,
			Input:          "nums = [-1,-2,-3,-4,-5], target = -8",
			ExpectedOutput: "[2,4]",
			IsSample:       false,
			Points:         20,
			Notes:          "Negative numbers",
			SequenceNumber: 5,
			CreatedAt:      time.Now(),
		},
	}

	// First, clear existing test cases
	testCasesCollection := database.GetCollection("OJ", "test_cases")
	_, err = testCasesCollection.DeleteMany(ctx, bson.M{"problem_db_id": problem.ID})
	if err != nil {
		log.Fatalf("Error deleting existing test cases: %v", err)
	}

	// Insert test cases
	for _, tc := range testCases {
		result, err := testCasesCollection.InsertOne(ctx, tc)
		if err != nil {
			log.Fatalf("Error inserting test case: %v", err)
		}
		log.Printf("Inserted test case %s: %s", result.InsertedID, tc.Input)
	}

	log.Printf("Successfully added %d test cases for problem %s", len(testCases), problemID)
}
