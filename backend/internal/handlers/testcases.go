package handlers

import (
	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/utils"

	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func AddTestCaseHandler(w http.ResponseWriter, r *http.Request) { // Only allowed for admins
	if r.Method != http.MethodPost {
		utils.SendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	var payload models.AddTestCasePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Invalid add test case request payload:", err)
		utils.SendJSONError(w, "Invalid request payload for test case.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate input (Points and SequenceNumber might have defaults if not provided, or be required)
	if payload.ProblemDBID == "" || payload.Input == "" {
		utils.SendJSONError(w, "ProblemDBID and Input are required for a test case.", http.StatusBadRequest)
		return
	}
	// You might want to add validation for payload.Points and payload.SequenceNumber (e.g., >= 0)

	problemObjectID, err := primitive.ObjectIDFromHex(payload.ProblemDBID)
	if err != nil {
		utils.SendJSONError(w, "Invalid ProblemDBID format. Must be a valid ObjectID hex string.", http.StatusBadRequest)
		return
	}

	problemsCollection := database.GetCollection("OJ", "problems")
	ctxCheck, cancelCheck := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCheck()
	var existingProblem models.Problem
	err = problemsCollection.FindOne(ctxCheck, primitive.M{"_id": problemObjectID}).Decode(&existingProblem)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			utils.SendJSONError(w, "Problem with the given ProblemDBID not found.", http.StatusNotFound)
			return
		}
		log.Println("Error checking for existing problem:", err)
		utils.SendJSONError(w, "Error verifying problem existence.", http.StatusInternalServerError)
		return
	}

	newTestCase := models.TestCase{
		ProblemDBID:    problemObjectID,
		Input:          payload.Input,
		ExpectedOutput: payload.ExpectedOutput,
		IsSample:       payload.IsSample,
		Points:         payload.Points,
		Notes:          payload.Notes,
		SequenceNumber: payload.SequenceNumber,
		CreatedAt:      time.Now(),
	}

	testCasesCollection := database.GetCollection("OJ", "testcases")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := testCasesCollection.InsertOne(ctx, newTestCase)
	if err != nil {
		log.Println("Failed to insert test case into DB:", err)
		utils.SendJSONError(w, "Failed to add test case.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]interface{}{
		"message":      "Test case added successfully",
		"test_case_id": result.InsertedID,
	}
	json.NewEncoder(w).Encode(response)
	log.Printf("Test case added for problem %s with ID: %v. Points: %d, Sequence: %d\n", payload.ProblemDBID, result.InsertedID, payload.Points, payload.SequenceNumber)
}
