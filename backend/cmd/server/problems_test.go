package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/internal/models"

	"github.com/joho/godotenv"
)

func init() {
	// Load environment variables for tests if not already loaded
	err := godotenv.Load("../../../.env") // Adjust path as needed
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
	handler := http.HandlerFunc(withCORS(getProblemsHandler)) // Test with CORS

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
	// Note: This test requires a problem with a known problem_id or ObjectID to exist in the DB.
	// You might need to insert a test problem as part of your test setup.
	// Replace "test-problem-1" with an actual problem_id or ObjectID from your DB.
	problemID := "test-problem-1"
	// problemID := "60f1a4b9d2f0b9f8a9b2c3d4" // Example ObjectID

	req, err := http.NewRequest("GET", "/problems/"+problemID, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(withCORS(getProblemHandler)) // Test with CORS

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
	// if problem.ProblemID != problemID && problem.ID.Hex() != problemID {
	// 	t.Errorf("Retrieved problem does not match requested ID/problem_id")
	// }

	t.Logf("TestGetProblemHandler_Success passed for problem ID/ObjectID: %s", problemID)
}

func TestGetProblemHandler_NotFound(t *testing.T) {
	// Use an ID that is unlikely to exist
	problemID := "nonexistent-problem-12345"

	req, err := http.NewRequest("GET", "/problems/"+problemID, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getProblemHandler)

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
