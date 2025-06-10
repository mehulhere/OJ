package handlers

import (
	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Helper function to create a test JWT token
func createTestAuthToken(t *testing.T, userID string, isAdmin bool) string {
	claims := &types.Claims{
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		Firstname: "Test",
		Lastname:  "User",
		IsAdmin:   isAdmin,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		t.Fatalf("Failed to create test token: %v", err)
	}
	return tokenString
}

// Helper function to create a test problem
func createTestProblem(t *testing.T) string {
	problemID := fmt.Sprintf("TEST_PROB_%d", time.Now().UnixNano())

	problem := models.Problem{
		ProblemID:       problemID,
		Title:           "Test Problem",
		Statement:       "This is a test problem",
		Difficulty:      "Easy",
		TimeLimitMs:     1000,
		MemoryLimitMB:   128,
		Tags:            []string{"test"},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		ConstraintsText: "1 <= n <= 100",
	}

	problemsCollection := database.GetCollection("OJ", "problems")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := problemsCollection.InsertOne(ctx, problem)
	if err != nil {
		t.Fatalf("Failed to create test problem: %v", err)
	}

	// Get the inserted ID
	problemDBID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		t.Fatalf("Failed to get problem ID")
	}

	// Create a test case for the problem
	testCase := models.TestCase{
		ProblemDBID:    problemDBID,
		Input:          "1 2",
		ExpectedOutput: "3",
		IsSample:       true,
		Points:         10,
		SequenceNumber: 1,
		CreatedAt:      time.Now(),
	}

	testCasesCollection := database.GetCollection("OJ", "test_cases")
	_, err = testCasesCollection.InsertOne(ctx, testCase)
	if err != nil {
		t.Fatalf("Failed to create test case: %v", err)
	}

	// Add problem_id to test case document for compatibility with the handler
	_, err = testCasesCollection.UpdateOne(
		ctx,
		bson.M{"problem_db_id": problemDBID},
		bson.M{"$set": bson.M{"problem_id": problemID}},
	)
	if err != nil {
		t.Fatalf("Failed to update test case with problem_id: %v", err)
	}

	return problemID
}

// Helper function to clean up test data
func cleanupTestData(t *testing.T, submissionID primitive.ObjectID, problemID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Remove submission from database
	submissionsCollection := database.GetCollection("OJ", "submissions")
	_, err := submissionsCollection.DeleteOne(ctx, bson.M{"_id": submissionID})
	if err != nil {
		t.Logf("Warning: Failed to delete test submission: %v", err)
	}

	// Remove problem and test cases
	problemsCollection := database.GetCollection("OJ", "problems")
	_, err = problemsCollection.DeleteOne(ctx, bson.M{"problem_id": problemID})
	if err != nil {
		t.Logf("Warning: Failed to delete test problem: %v", err)
	}

	testCasesCollection := database.GetCollection("OJ", "test_cases")
	_, err = testCasesCollection.DeleteMany(ctx, bson.M{"problem_id": problemID})
	if err != nil {
		t.Logf("Warning: Failed to delete test cases: %v", err)
	}

	// Remove submission directory
	submissionDir := filepath.Join("./submissions", submissionID.Hex())
	err = os.RemoveAll(submissionDir)
	if err != nil {
		t.Logf("Warning: Failed to remove submission directory: %v", err)
	}
}

// Test submission creation and file storage
func TestSubmitSolutionHandler_FileStorage(t *testing.T) {
	// Create a test user ID
	userID := primitive.NewObjectID()

	// Create a test problem
	problemID := createTestProblem(t)

	// Create test submission data
	submissionData := struct {
		ProblemID string `json:"problem_id"`
		Language  string `json:"language"`
		Code      string `json:"code"`
	}{
		ProblemID: problemID,
		Language:  "python",
		Code:      "def add(a, b):\n    return a + b\n\na, b = map(int, input().split())\nprint(add(a, b))",
	}

	// Convert submission data to JSON
	payloadBytes, err := json.Marshal(submissionData)
	if err != nil {
		t.Fatalf("Failed to marshal submission data: %v", err)
	}

	// Create a request with the submission data
	req, err := http.NewRequest("POST", "/submit", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add auth token cookie
	req.AddCookie(&http.Cookie{
		Name:  "authToken",
		Value: createTestAuthToken(t, userID.Hex(), false),
	})

	req.Header.Set("Content-Type", "application/json")

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(SubmitSolutionHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Handler returned wrong status code: got %v want %v. Body: %s",
			status, http.StatusCreated, rr.Body.String())
	}

	// Parse the response to get the submission ID
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	submissionIDStr, ok := response["submission_id"].(string)
	if !ok || submissionIDStr == "" {
		t.Fatalf("Missing or invalid submission_id in response: %v", response)
	}

	submissionID, err := primitive.ObjectIDFromHex(submissionIDStr)
	if err != nil {
		t.Fatalf("Invalid submission ID format: %v", err)
	}

	// Wait for submission processing to complete
	time.Sleep(2 * time.Second)

	// Check if submission directory was created
	submissionDir := filepath.Join("./submissions", submissionID.Hex())
	if _, err := os.Stat(submissionDir); os.IsNotExist(err) {
		t.Errorf("Submission directory was not created: %s", submissionDir)
	}

	// Check if code file was created
	codeFilePath := filepath.Join(submissionDir, "code.py")
	if _, err := os.Stat(codeFilePath); os.IsNotExist(err) {
		t.Errorf("Code file was not created: %s", codeFilePath)
	}

	// Check if code content is correct
	codeContent, err := ioutil.ReadFile(codeFilePath)
	if err != nil {
		t.Errorf("Failed to read code file: %v", err)
	} else if string(codeContent) != submissionData.Code {
		t.Errorf("Code content doesn't match. Expected: %s, Got: %s", submissionData.Code, string(codeContent))
	}

	// Check if test case status file was created
	testCaseStatusPath := filepath.Join(submissionDir, "testcasesStatus.txt")
	if _, err := os.Stat(testCaseStatusPath); os.IsNotExist(err) {
		t.Errorf("Test case status file was not created: %s", testCaseStatusPath)
	}

	// Clean up test data
	cleanupTestData(t, submissionID, problemID)
}

// Test submission queue processing
func TestSubmissionQueueProcessing(t *testing.T) {
	// Create a test user ID
	userID := primitive.NewObjectID()

	// Create a test problem
	problemID := createTestProblem(t)

	// Create test submission in the database directly
	submission := models.Submission{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		ProblemID:   problemID,
		Language:    "python",
		Status:      models.StatusPending,
		SubmittedAt: time.Now(),
	}

	submissionsCollection := database.GetCollection("OJ", "submissions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := submissionsCollection.InsertOne(ctx, submission)
	if err != nil {
		t.Fatalf("Failed to create test submission: %v", err)
	}

	// Create submission directory
	submissionDir := filepath.Join("./submissions", submission.ID.Hex())
	err = os.MkdirAll(submissionDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create submission directory: %v", err)
	}

	// Create code file
	codeFilePath := filepath.Join(submissionDir, "code.py")
	codeContent := "def add(a, b):\n    return a + b\n\na, b = map(int, input().split())\nprint(add(a, b))"
	err = ioutil.WriteFile(codeFilePath, []byte(codeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create code file: %v", err)
	}

	// Process the submission
	processSubmission(submission.ID)

	// Check if submission status was updated
	var updatedSubmission models.Submission
	err = submissionsCollection.FindOne(ctx, bson.M{"_id": submission.ID}).Decode(&updatedSubmission)
	if err != nil {
		t.Fatalf("Failed to retrieve updated submission: %v", err)
	}

	// Check if status was updated from pending
	if updatedSubmission.Status == models.StatusPending {
		t.Errorf("Submission status was not updated. Still pending.")
	}

	// Check if test cases passed/total was updated
	if updatedSubmission.TestCasesTotal == 0 {
		t.Errorf("Test cases total was not updated")
	}

	// Check if test case status file was created
	testCaseStatusPath := filepath.Join(submissionDir, "testcasesStatus.txt")
	if _, err := os.Stat(testCaseStatusPath); os.IsNotExist(err) {
		t.Errorf("Test case status file was not created: %s", testCaseStatusPath)
	}

	// Clean up test data
	cleanupTestData(t, submission.ID, problemID)
}

// Test submission detail retrieval with file content
func TestGetSubmissionDetailsHandler(t *testing.T) {
	// Create a test user ID
	userID := primitive.NewObjectID()

	// Create a test problem
	problemID := createTestProblem(t)

	// Create test submission in the database directly
	submission := models.Submission{
		ID:              primitive.NewObjectID(),
		UserID:          userID,
		ProblemID:       problemID,
		Language:        "python",
		Status:          models.StatusAccepted,
		ExecutionTimeMs: 100,
		MemoryUsedKB:    1024,
		TestCasesPassed: 1,
		TestCasesTotal:  1,
		SubmittedAt:     time.Now(),
	}

	submissionsCollection := database.GetCollection("OJ", "submissions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := submissionsCollection.InsertOne(ctx, submission)
	if err != nil {
		t.Fatalf("Failed to create test submission: %v", err)
	}

	// Create submission directory and files
	submissionDir := filepath.Join("./submissions", submission.ID.Hex())
	err = os.MkdirAll(submissionDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create submission directory: %v", err)
	}

	// Create code file
	codeFilePath := filepath.Join(submissionDir, "code.py")
	codeContent := "def add(a, b):\n    return a + b\n\na, b = map(int, input().split())\nprint(add(a, b))"
	err = ioutil.WriteFile(codeFilePath, []byte(codeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create code file: %v", err)
	}

	// Create test case status file
	testCaseStatusPath := filepath.Join(submissionDir, "testcasesStatus.txt")
	testCaseStatus := "Test Case 1: PASSED"
	err = ioutil.WriteFile(testCaseStatusPath, []byte(testCaseStatus), 0644)
	if err != nil {
		t.Fatalf("Failed to create test case status file: %v", err)
	}

	// Create request to get submission details
	req, err := http.NewRequest("GET", "/submissions/"+submission.ID.Hex(), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add auth token cookie
	req.AddCookie(&http.Cookie{
		Name:  "authToken",
		Value: createTestAuthToken(t, userID.Hex(), false),
	})

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(GetSubmissionDetailsHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v. Body: %s",
			status, http.StatusOK, rr.Body.String())
	}

	// Parse the response
	var response struct {
		ID              string `json:"id"`
		Code            string `json:"code"`
		TestCaseStatus  string `json:"test_case_status"`
		TestCasesPassed int    `json:"test_cases_passed"`
		TestCasesTotal  int    `json:"test_cases_total"`
	}

	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	// Check if code was included in response
	if response.Code != codeContent {
		t.Errorf("Code in response doesn't match. Expected: %s, Got: %s", codeContent, response.Code)
	}

	// Check if test case status was included in response
	if response.TestCaseStatus != testCaseStatus {
		t.Errorf("Test case status in response doesn't match. Expected: %s, Got: %s", testCaseStatus, response.TestCaseStatus)
	}

	// Check if test cases passed/total match
	if response.TestCasesPassed != submission.TestCasesPassed || response.TestCasesTotal != submission.TestCasesTotal {
		t.Errorf("Test cases passed/total don't match. Expected: %d/%d, Got: %d/%d",
			submission.TestCasesPassed, submission.TestCasesTotal, response.TestCasesPassed, response.TestCasesTotal)
	}

	// Clean up test data
	cleanupTestData(t, submission.ID, problemID)
}
