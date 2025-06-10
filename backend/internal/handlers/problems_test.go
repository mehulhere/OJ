package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend/internal/database"
	"backend/internal/middleware"
	"backend/internal/models"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	// Load environment variables for tests if not already loaded
	err := godotenv.Load("../../.env") // Adjust path as needed
	if err != nil {
		fmt.Println("Warning: Error loading .env file in tests. Ensure environment variables are set.")
	}

	// Note: Database connection is required for these tests.
	// Ensure MongoDB is running and MONGO_URI is set.
	// Consider adding a test database setup/teardown.
}

func TestGetProblemsHandler_Success(t *testing.T) {
	req, err := http.NewRequest("GET", "/problems", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(middleware.WithCORS(GetProblemsHandler)) // Test with CORS

	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v. Body: %s",
			status, http.StatusOK, rr.Body.String())
	}

	// Check the Content-Type header
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong Content-Type: got %v want %v",
			contentType, expectedContentType)
	}

	// Check if the response body is a valid JSON array
	var problems []models.ProblemListItem
	err = json.Unmarshal(rr.Body.Bytes(), &problems)
	if err != nil {
		t.Fatalf("Failed to parse response body as JSON array: %v. Body: %s", err, rr.Body.String())
	}

	// Optionally, check if the array is not empty if you expect problems to exist
	// if len(problems) == 0 {
	// 	t.Error("Expected problems list to be non-empty")
	// }

	t.Log("TestGetProblemsHandler_Success passed")
}

func TestGetProblemHandler_Success(t *testing.T) {
	// Define a unique problem ID for this test run to ensure idempotency
	// and allow cleanup.
	testProblemIDString := fmt.Sprintf("test-problem-%d", time.Now().UnixNano())

	// Setup: Insert a test problem into the database.
	problemsCollection := database.GetCollection("OJ", "problems")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	problemToInsert := models.Problem{
		ID:              primitive.NewObjectID(), // Generate a new MongoDB ObjectID
		ProblemID:       testProblemIDString,     // Custom string ID used by the handler path
		Title:           "Test Problem for GetProblemHandler",
		Statement:       "This is a test description.",
		Difficulty:      "Easy",
		Tags:            []string{"test", "dummy"},
		ConstraintsText: "N > 0",
		TimeLimitMs:     1000,
		MemoryLimitMB:   256,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err := problemsCollection.InsertOne(ctx, problemToInsert)
	if err != nil {
		t.Fatalf("Setup: Failed to insert test problem: %v", err)
	}

	// Cleanup: Ensure the test problem is deleted after the test.
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, delErr := problemsCollection.DeleteOne(cleanupCtx, primitive.M{"problem_id": testProblemIDString})
		if delErr != nil {
			t.Logf("Cleanup: Failed to delete test problem %s: %v", testProblemIDString, delErr)
		} else {
			t.Logf("Cleanup: Successfully deleted test problem %s", testProblemIDString)
		}
	})

	req, err := http.NewRequest("GET", "/problems/"+testProblemIDString, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(middleware.WithCORS(GetProblemHandler)) // Test with CORS

	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v. Body: %s",
			status, http.StatusOK, rr.Body.String())
	}

	// Check the Content-Type header
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong Content-Type: got %v want %v",
			contentType, expectedContentType)
	}

	// Check if the response body is a valid JSON object representing a problem
	var problem struct {
		models.Problem                    // Embed all fields from models.Problem
		SampleTestCases []models.TestCase `json:"sample_test_cases,omitempty"`
	}
	err = json.Unmarshal(rr.Body.Bytes(), &problem)
	if err != nil {
		t.Fatalf("Failed to parse response body as JSON object: %v. Body: %s", err, rr.Body.String())
	}

	// Optionally, check some fields of the retrieved problem
	if problem.ProblemID != testProblemIDString {
		t.Errorf("Retrieved problem ProblemID does not match: got %s, want %s", problem.ProblemID, testProblemIDString)
	}
	if problem.Title != problemToInsert.Title {
		t.Errorf("Retrieved problem Title does not match: got %s, want %s", problem.Title, problemToInsert.Title)
	}

	t.Logf("TestGetProblemHandler_Success passed for problem ID: %s", testProblemIDString)
}

func TestGetProblemHandler_NotFound(t *testing.T) {
	// Use an ID that is unlikely to exist
	problemID := "nonexistent-problem-12345"

	req, err := http.NewRequest("GET", "/problems/"+problemID, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(middleware.WithCORS(GetProblemHandler))

	handler.ServeHTTP(rr, req)

	// Check the status code - expect Not Found
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code for not found problem: got %v want %v. Body: %s",
			status, http.StatusNotFound, rr.Body.String())
	}

	// Optionally, check the error message
	var response map[string]string
	json.Unmarshal(rr.Body.Bytes(), &response)
	if msg, ok := response["message"]; ok && !strings.Contains(msg, "Problem not found") {
		t.Errorf("Unexpected error message for not found problem: %v", msg)
	}

	t.Logf("TestGetProblemHandler_NotFound passed for problem ID: %s", problemID)
}
