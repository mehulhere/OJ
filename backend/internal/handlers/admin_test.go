package handlers

import (
	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/types"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// setupAdminTestContext sets up a request with admin context
func setupAdminTestContext() (*http.Request, *httptest.ResponseRecorder, context.Context) {
	req := httptest.NewRequest(http.MethodPost, "/admin/problems", nil)
	w := httptest.NewRecorder()

	// Create admin claims
	adminClaims := &types.Claims{
		UserID:    "admin123",
		Username:  "admin",
		Email:     "admin@example.com",
		Firstname: "Admin",
		Lastname:  "User",
		IsAdmin:   true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Add claims to request context
	ctx := context.WithValue(req.Context(), "claims", adminClaims)
	req = req.WithContext(ctx)

	return req, w, ctx
}

// setupTestProblem creates a test problem in the database
func setupTestProblem(t *testing.T) primitive.ObjectID {
	// Create a test problem
	problem := models.Problem{
		ID:              primitive.NewObjectID(),
		ProblemID:       "TEST-123",
		Title:           "Test Problem",
		Difficulty:      "Medium",
		Statement:       "This is a test problem statement",
		ConstraintsText: "1 <= n <= 100",
		TimeLimitMs:     2000,
		MemoryLimitMB:   256,
		Author:          "admin",
		Tags:            []string{"test", "array"},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Insert the problem into the database
	problemsCollection := database.GetCollection("OJ", "problems")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := problemsCollection.InsertOne(ctx, problem)
	if err != nil {
		t.Fatalf("Failed to insert test problem: %v", err)
	}

	return problem.ID
}

// cleanupTestProblem removes a test problem from the database
func cleanupTestProblem(t *testing.T, problemID primitive.ObjectID) {
	problemsCollection := database.GetCollection("OJ", "problems")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := problemsCollection.DeleteOne(ctx, bson.M{"_id": problemID})
	if err != nil {
		t.Fatalf("Failed to delete test problem: %v", err)
	}
}

// TestCreateProblemHandler tests the CreateProblemHandler function
func TestCreateProblemHandler(t *testing.T) {
	// Skip if not in test environment
	if database.DB == nil {
		t.Skip("Skipping test: No database connection")
	}

	// Setup request with admin context
	req, w, _ := setupAdminTestContext()

	// Create test problem payload
	problemPayload := map[string]interface{}{
		"problem_id":       "TEST-456",
		"title":            "New Test Problem",
		"difficulty":       "Easy",
		"statement":        "This is a new test problem statement",
		"constraints_text": "1 <= n <= 50",
		"time_limit_ms":    1000,
		"memory_limit_mb":  128,
		"tags":             []string{"test", "string"},
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(problemPayload)
	if err != nil {
		t.Fatalf("Failed to marshal problem payload: %v", err)
	}

	// Set request body with NopCloser to implement ReadCloser
	req.Body = io.NopCloser(bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	// Call handler
	CreateProblemHandler(w, req)

	// Check response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	// Verify response contains problem data
	var response models.Problem
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Title != "New Test Problem" {
		t.Errorf("Expected problem title 'New Test Problem', got '%s'", response.Title)
	}

	// Cleanup - delete the created problem
	cleanupTestProblem(t, response.ID)
}

// TestAddTestCaseHandler tests the AddTestCaseHandler function
func TestAddTestCaseHandler(t *testing.T) {
	// Skip if not in test environment
	if database.DB == nil {
		t.Skip("Skipping test: No database connection")
	}

	// Setup test problem
	problemID := setupTestProblem(t)
	defer cleanupTestProblem(t, problemID)

	// Setup request with admin context
	req, w, _ := setupAdminTestContext()

	// Create test case payload
	testCasePayload := map[string]interface{}{
		"problem_db_id":   problemID.Hex(),
		"input":           "5\n1 2 3 4 5",
		"expected_output": "15",
		"is_sample":       true,
		"points":          10,
		"sequence_number": 1,
		"notes":           "Sample test case",
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(testCasePayload)
	if err != nil {
		t.Fatalf("Failed to marshal test case payload: %v", err)
	}

	// Set request body with NopCloser to implement ReadCloser
	req.Body = io.NopCloser(bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	// Call handler
	AddTestCaseHandler(w, req)

	// Check response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	// Verify response contains test case ID
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["test_case_id"]; !ok {
		t.Errorf("Response does not contain test_case_id")
	}

	// Cleanup - delete the created test case
	testCasesCollection := database.GetCollection("OJ", "testcases")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = testCasesCollection.DeleteMany(ctx, bson.M{"problem_db_id": problemID})
	if err != nil {
		t.Fatalf("Failed to delete test cases: %v", err)
	}
}
