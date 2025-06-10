package handlers

import (
	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/types"
	"backend/internal/utils"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetProblemsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendJSONError(w, "Method not allowed. Only GET is accepted.", http.StatusMethodNotAllowed)
		return
	}

	problemsCollection := database.GetCollection("OJ", "problems")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := problemsCollection.Find(ctx, primitive.M{})
	if err != nil {
		log.Println("Error fetching problems from DB:", err)
		utils.SendJSONError(w, "Failed to retrieve problems.", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var problems []models.ProblemListItem
	for cursor.Next(ctx) {
		var problem models.Problem
		if err := cursor.Decode(&problem); err != nil {
			log.Println("Error decoding problem:", err)
			continue
		}
		problems = append(problems, models.ProblemListItem{
			ID:         problem.ID,
			ProblemID:  problem.ProblemID,
			Title:      problem.Title,
			Difficulty: problem.Difficulty,
			Tags:       problem.Tags,
		})
	}

	if err := cursor.Err(); err != nil {
		log.Println("Error with problems cursor:", err)
		utils.SendJSONError(w, "Error processing problems list.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(problems); err != nil {
		log.Println("Error encoding problems to JSON:", err)
		// If headers are already written, this specific utils.SendJSONError might not be effective.
		// Consider more centralized error handling for such cases.
	}
	log.Println("Successfully retrieved problems list. Count:", len(problems))
}

// getProblemHandler (singular problem)
func GetProblemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendJSONError(w, "Method not allowed. Only GET is accepted.", http.StatusMethodNotAllowed)
		return
	}

	pathSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathSegments) < 2 || pathSegments[0] != "problems" {
		utils.SendJSONError(w, "Invalid problem URL format. Expected /problems/{id}", http.StatusBadRequest)
		return
	}
	problemIDFromURL := pathSegments[len(pathSegments)-1]

	if problemIDFromURL == "" {
		utils.SendJSONError(w, "Problem ID is required in the URL path.", http.StatusBadRequest)
		return
	}

	problemsCollection := database.GetCollection("OJ", "problems")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var problemData models.Problem

	filter := primitive.M{"problem_id": problemIDFromURL}
	err := problemsCollection.FindOne(ctx, filter).Decode(&problemData)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			objectID, idErr := primitive.ObjectIDFromHex(problemIDFromURL)
			if idErr != nil {
				utils.SendJSONError(w, "Problem not found (and ID is not a valid ObjectID).", http.StatusNotFound)
				return
			}
			filter = primitive.M{"_id": objectID}
			err = problemsCollection.FindOne(ctx, filter).Decode(&problemData)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					utils.SendJSONError(w, "Problem not found.", http.StatusNotFound)
					return
				}
				log.Println("Error fetching single problem by ObjectID from DB:", err)
				utils.SendJSONError(w, "Failed to retrieve problem.", http.StatusInternalServerError)
				return
			}
		} else {
			log.Println("Error fetching single problem by problem_id from DB:", err)
			utils.SendJSONError(w, "Failed to retrieve problem.", http.StatusInternalServerError)
			return
		}
	}

	// Fetch sample test cases
	var fetchedSampleTestCases []models.TestCase
	testCasesCollection := database.GetCollection("OJ", "testcases")
	findOptions := options.Find()
	findOptions.SetSort(primitive.D{{Key: "sequence_number", Value: 1}})
	findOptions.SetLimit(2)
	testCaseFilter := primitive.M{"problem_db_id": problemData.ID}

	cursor, err := testCasesCollection.Find(ctx, testCaseFilter, findOptions)
	if err != nil {
		log.Println("Error fetching sample test cases from DB for problem "+problemData.ID.Hex()+":", err)
		// fetchedSampleTestCases will remain empty or nil
	} else {
		defer cursor.Close(ctx)
		if err = cursor.All(ctx, &fetchedSampleTestCases); err != nil {
			log.Println("Error decoding sample test cases for problem "+problemData.ID.Hex()+":", err)
			fetchedSampleTestCases = nil // Ensure it's nil if decoding fails
		}
	}

	// Define a response structure that embeds problemData and adds sample test cases
	// This way, models.Problem struct remains clean.
	type problemResponse struct {
		models.Problem                    // Embed all fields from models.Problem
		SampleTestCases []models.TestCase `json:"sample_test_cases,omitempty"`
	}

	responsePayload := problemResponse{
		Problem:         problemData,
		SampleTestCases: fetchedSampleTestCases,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(responsePayload); err != nil {
		log.Println("Error encoding single problem (with samples) to JSON:", err)
	}
	log.Println("Successfully retrieved problem:", responsePayload.Title, "with", len(responsePayload.SampleTestCases), "sample test cases.")
}

// CreateProblemHandler handles the creation of a new problem
func CreateProblemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	// Get claims from context to identify the admin user
	claims, ok := r.Context().Value("claims").(*types.Claims)
	if !ok {
		utils.SendJSONError(w, "Failed to retrieve user information.", http.StatusInternalServerError)
		return
	}

	var problem models.Problem
	err := json.NewDecoder(r.Body).Decode(&problem)
	if err != nil {
		log.Println("Invalid problem payload:", err)
		utils.SendJSONError(w, "Invalid request payload. Please check your input.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate required fields
	if problem.Title == "" || problem.Statement == "" || problem.Difficulty == "" {
		utils.SendJSONError(w, "Title, statement, and difficulty are required fields.", http.StatusBadRequest)
		return
	}

	// Set creation and update timestamps
	now := time.Now()
	problem.CreatedAt = now
	problem.UpdatedAt = now

	// Set author to the current user
	problem.Author = claims.Username

	// Insert the problem into the database
	problemsCollection := database.GetCollection("OJ", "problems")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := problemsCollection.InsertOne(ctx, problem)
	if err != nil {
		log.Printf("Failed to create problem in DB: %v\n", err)
		utils.SendJSONError(w, "Failed to create problem.", http.StatusInternalServerError)
		return
	}

	// Get the inserted ID
	insertedID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		log.Println("Failed to get inserted problem ID")
		utils.SendJSONError(w, "Problem created but failed to retrieve its ID.", http.StatusInternalServerError)
		return
	}

	// Fetch the created problem to return it
	var createdProblem models.Problem
	err = problemsCollection.FindOne(ctx, primitive.M{"_id": insertedID}).Decode(&createdProblem)
	if err != nil {
		log.Printf("Failed to fetch created problem: %v\n", err)
		utils.SendJSONError(w, "Problem created but failed to retrieve its details.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdProblem)
	log.Printf("Problem created: %s (ID: %s) by admin: %s\n", createdProblem.Title, createdProblem.ID.Hex(), claims.Username)
}
