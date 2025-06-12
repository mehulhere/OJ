package handlers

import (
	"backend/internal/database"
	"backend/internal/models"
	"backend/internal/types"
	"backend/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Submission queue for processing submissions
var (
	submissionQueue     = make(chan primitive.ObjectID, 100) // Buffer for 100 submissions
	submissionQueueLock sync.Mutex
	isProcessing        = false
)

// Initialize the submissions directory
func init() {
	// Create the submissions directory if it doesn't exist
	submissionsDir := "./submissions"
	if err := os.MkdirAll(submissionsDir, 0755); err != nil {
		log.Printf("Failed to create submissions directory: %v", err)
	}
}

// SubmitSolutionHandler handles code submissions from users
func SubmitSolutionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from JWT token
	cookie, err := r.Cookie("authToken")
	if err != nil {
		utils.SendJSONError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	tokenStr := cookie.Value
	claims := &types.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		utils.SendJSONError(w, "Invalid authentication token", http.StatusUnauthorized)
		return
	}

	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		utils.SendJSONError(w, "Invalid user ID in token", http.StatusBadRequest)
		return
	}

	var submissionData models.SubmissionData

	if err := json.NewDecoder(r.Body).Decode(&submissionData); err != nil {
		utils.SendJSONError(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate submission data
	if submissionData.ProblemID == "" || submissionData.Language == "" || submissionData.Code == "" {
		utils.SendJSONError(w, "Problem ID, language, and code are required", http.StatusBadRequest)
		return
	}

	// Create submission record
	submission := models.Submission{
		UserID:      userID,
		ProblemID:   submissionData.ProblemID,
		Language:    submissionData.Language,
		Status:      models.StatusPending, // Initially set as pending
		SubmittedAt: time.Now(),
	}

	// Save to database
	submissionsCollection := database.GetCollection("OJ", "submissions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := submissionsCollection.InsertOne(ctx, submission)
	if err != nil {
		log.Printf("Failed to save submission: %v", err)
		utils.SendJSONError(w, "Failed to process submission", http.StatusInternalServerError)
		return
	}

	// Get the inserted ID
	submissionID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		log.Printf("Failed to get submission ID")
		utils.SendJSONError(w, "Failed to process submission", http.StatusInternalServerError)
		return
	}

	// Create directory for this submission
	submissionDir := filepath.Join("./submissions", submissionID.Hex())
	if err := os.MkdirAll(submissionDir, 0755); err != nil {
		log.Printf("Failed to create submission directory: %v", err)
		utils.SendJSONError(w, "Failed to process submission", http.StatusInternalServerError)
		return
	}

	// Determine file extension based on language
	var fileExtension string
	switch strings.ToLower(submissionData.Language) {
	case "python":
		fileExtension = ".py"
	case "javascript":
		fileExtension = ".js"
	case "cpp":
		fileExtension = ".cpp"
	case "java":
		fileExtension = ".java"
	default:
		fileExtension = ".txt"
	}

	// Write code to file
	codeFilePath := filepath.Join(submissionDir, "code"+fileExtension)
	if err := os.WriteFile(codeFilePath, []byte(submissionData.Code), 0644); err != nil {
		log.Printf("Failed to write code file: %v", err)
		utils.SendJSONError(w, "Failed to process submission", http.StatusInternalServerError)
		return
	}

	// Add submission to processing queue
	submissionQueueLock.Lock()
	submissionQueue <- submissionID
	if !isProcessing {
		isProcessing = true
		go processSubmissionQueue()
	}
	submissionQueueLock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Submission received and queued for evaluation",
		"submission_id": submissionID.Hex(),
	})
}

// Process submissions in the queue
func processSubmissionQueue() {
	for submissionID := range submissionQueue {
		// Process the submission
		processSubmission(submissionID)

		// Check if queue is empty
		submissionQueueLock.Lock()
		if len(submissionQueue) == 0 {
			isProcessing = false
			submissionQueueLock.Unlock()
			break
		}
		submissionQueueLock.Unlock()
	}
}

// Process a single submission
func processSubmission(submissionID primitive.ObjectID) {
	log.Printf("Processing submission: %s", submissionID.Hex())

	// Get submission details from database
	submissionsCollection := database.GetCollection("OJ", "submissions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var submission models.Submission
	err := submissionsCollection.FindOne(ctx, bson.M{"_id": submissionID}).Decode(&submission)
	if err != nil {
		log.Printf("Failed to retrieve submission %s: %v", submissionID.Hex(), err)
		return
	}

	// Get problem details to know test cases and time limits
	problemsCollection := database.GetCollection("OJ", "problems")
	var problem models.Problem
	err = problemsCollection.FindOne(ctx, bson.M{"problem_id": submission.ProblemID}).Decode(&problem)
	if err != nil {
		log.Printf("Failed to retrieve problem %s: %v", submission.ProblemID, err)
		updateSubmissionStatus(submissionID, models.StatusRuntimeError, 0, 0, 0, 0)
		return
	}

	// Get test cases for the problem
	testCasesCollection := database.GetCollection("OJ", "test_cases")
	cursor, err := testCasesCollection.Find(ctx, bson.M{"problem_db_id": problem.ID})
	if err != nil {
		log.Printf("Failed to retrieve test cases for problem %s: %v", submission.ProblemID, err)
		updateSubmissionStatus(submissionID, models.StatusRuntimeError, 0, 0, 0, 0)
		return
	}
	defer cursor.Close(ctx)

	var testCases []models.TestCase
	if err = cursor.All(ctx, &testCases); err != nil {
		log.Printf("Failed to decode test cases: %v", err)
		updateSubmissionStatus(submissionID, models.StatusRuntimeError, 0, 0, 0, 0)
		return
	}

	if len(testCases) == 0 {
		log.Printf("No test cases found for problem %s", submission.ProblemID)
		updateSubmissionStatus(submissionID, models.StatusRuntimeError, 0, 0, 0, 0)
		return
	}

	// Determine file path
	submissionDir := filepath.Join("./submissions", submissionID.Hex())
	var fileExtension string
	switch strings.ToLower(submission.Language) {
	case "python":
		fileExtension = ".py"
	case "javascript":
		fileExtension = ".js"
	case "cpp":
		fileExtension = ".cpp"
	case "java":
		fileExtension = ".java"
	default:
		fileExtension = ".txt"
	}
	codeFilePath := filepath.Join(submissionDir, "code"+fileExtension)

	// Create test case status file
	testCaseStatusPath := filepath.Join(submissionDir, "testcasesStatus.txt")
	testCaseStatusFile, err := os.Create(testCaseStatusPath)
	if err != nil {
		log.Printf("Failed to create test case status file: %v", err)
	}
	defer testCaseStatusFile.Close()

	// Execute each test case
	passedCount := 0
	totalCount := len(testCases)

	maxExecutionTime := 0
	maxMemoryUsed := 0

	for i, testCase := range testCases {
		// Create a context with timeout based on problem's time limit
		execCtx, execCancel := context.WithTimeout(context.Background(), time.Duration(problem.TimeLimitMs)*time.Millisecond)
		defer execCancel()

		// Execute code with test case input
		result, err := executeCode(execCtx, submission.Language, codeFilePath, testCase.Input, problem.TimeLimitMs)

		// Write test case result to file
		outputFilePath := filepath.Join(submissionDir, fmt.Sprintf("output_%d.txt", i+1))
		if err := os.WriteFile(outputFilePath, []byte(result.Output), 0644); err != nil {
			log.Printf("Failed to write output file: %v", err)
		}

		// Update max execution time and memory
		if result.ExecutionTimeMs > maxExecutionTime {
			maxExecutionTime = result.ExecutionTimeMs
		}
		if result.MemoryUsedKB > maxMemoryUsed {
			maxMemoryUsed = result.MemoryUsedKB
		}

		// Check result against expected output
		status := fmt.Sprintf("Test Case %d: ", i+1)

		if err != nil {
			if err.Error() == "context deadline exceeded" {
				status += "TIME_LIMIT_EXCEEDED"
			} else if strings.Contains(err.Error(), "memory limit") {
				status += "MEMORY_LIMIT_EXCEEDED"
			} else if strings.Contains(err.Error(), "compilation") {
				status += "COMPILATION_ERROR\n" + result.Output
			} else {
				status += "RUNTIME_ERROR\n" + result.Output
			}
		} else if result.Status == "compilation_error" {
			// Explicitly check for compilation errors from the result status
			status += "COMPILATION_ERROR\n" + result.Output
		} else if result.Status == "runtime_error" {
			status += "RUNTIME_ERROR\n" + result.Output
		} else if result.Status == "time_limit_exceeded" {
			status += "TIME_LIMIT_EXCEEDED"
		} else {
			// Compare output (trim whitespace)
			expectedOutput := strings.TrimSpace(testCase.ExpectedOutput)
			actualOutput := strings.TrimSpace(result.Output)

			if expectedOutput == actualOutput {
				status += "PASSED"
				passedCount++
			} else {
				status += "WRONG_ANSWER\n"
				status += fmt.Sprintf("Expected:\n%s\n\nActual:\n%s", expectedOutput, actualOutput)
			}
		}

		// Write status to test case status file
		testCaseStatusFile.WriteString(status + "\n\n")
	}

	// Determine final status
	var finalStatus models.SubmissionStatus
	if passedCount == totalCount {
		finalStatus = models.StatusAccepted
	} else {
		// Check if any test case had a specific error
		// Read the test case status file to find the error type
		testCaseStatusContent, err := os.ReadFile(testCaseStatusPath)
		if err == nil {
			content := string(testCaseStatusContent)
			fmt.Println(content)

			// Special handling for Python common errors - check for Python's error types directly in the output
			if strings.Contains(strings.ToLower(submission.Language), "python") {
				if strings.Contains(content, "NameError:") ||
					strings.Contains(content, "SyntaxError:") ||
					strings.Contains(content, "IndentationError:") ||
					strings.Contains(content, "TabError:") ||
					strings.Contains(content, "ImportError:") ||
					strings.Contains(content, "ModuleNotFoundError:") {
					finalStatus = models.StatusCompilationError
					log.Printf("Classified Python error as COMPILATION_ERROR: %s", submission.ID.Hex())
					updateSubmissionStatus(submissionID, finalStatus, maxExecutionTime, maxMemoryUsed, passedCount, totalCount)
					return
				}
			}

			if strings.Contains(content, "COMPILATION_ERROR") {
				finalStatus = models.StatusCompilationError
			} else if strings.Contains(content, "RUNTIME_ERROR") {
				finalStatus = models.StatusRuntimeError
			} else if strings.Contains(content, "TIME_LIMIT_EXCEEDED") {
				finalStatus = models.StatusTimeLimitExceeded
			} else if strings.Contains(content, "MEMORY_LIMIT_EXCEEDED") {
				finalStatus = models.StatusMemoryLimitExceeded
			} else {
				finalStatus = models.StatusWrongAnswer
			}
		} else {
			// Default to wrong answer if can't read the file
			finalStatus = models.StatusWrongAnswer
		}
	}

	// Update submission in database
	updateSubmissionStatus(submissionID, finalStatus, maxExecutionTime, maxMemoryUsed, passedCount, totalCount)

	log.Printf("Finished processing submission %s: %s (%d/%d passed)",
		submissionID.Hex(), finalStatus, passedCount, totalCount)
}

// Update submission status in database
func updateSubmissionStatus(submissionID primitive.ObjectID, status models.SubmissionStatus,
	executionTimeMs, memoryUsedKB, testCasesPassed, testCasesTotal int) {

	submissionsCollection := database.GetCollection("OJ", "submissions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":            status,
			"execution_time_ms": executionTimeMs,
			"memory_used_kb":    memoryUsedKB,
			"test_cases_passed": testCasesPassed,
			"test_cases_total":  testCasesTotal,
		},
	}

	_, err := submissionsCollection.UpdateOne(ctx, bson.M{"_id": submissionID}, update)
	if err != nil {
		log.Printf("Failed to update submission status: %v", err)
	}
}

// Execute code with given input
func executeCode(ctx context.Context, language, codePath, input string, timeLimitMs int) (types.ExecutionResult, error) {
	// This is a simplified version - in a real system, you would use a sandboxed execution environment
	// For now, we'll just use the existing ExecuteCodeHandler logic but adapted for our needs

	// Read the code file
	codeBytes, err := os.ReadFile(codePath)
	if err != nil {
		return types.ExecutionResult{}, fmt.Errorf("failed to read code file: %v", err)
	}

	// Create execution request
	execRequest := types.ExecutionRequest{
		Language:    language,
		Code:        string(codeBytes),
		Input:       input,
		TimeLimitMs: timeLimitMs,
	}

	// Execute the code
	result, err := utils.ExecuteCode(ctx, execRequest)
	return result, err
}

// GetSubmissionsHandler retrieves a list of submissions
func GetSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendJSONError(w, "Method not allowed. Only GET is accepted.", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from JWT token (for filtering by user if needed)
	var userID primitive.ObjectID
	var isAdmin bool

	cookie, err := r.Cookie("authToken")
	if err == nil {
		// Token exists, parse it
		tokenStr := cookie.Value
		claims := &types.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err == nil && token.Valid {
			userID, _ = primitive.ObjectIDFromHex(claims.UserID)
			isAdmin = claims.IsAdmin
		}
	}

	// Parse query parameters
	query := r.URL.Query()
	problemID := query.Get("problem_id")
	userIDParam := query.Get("user_id")
	filterByUser := query.Get("my_submissions") == "true"

	// Build the filter
	filter := bson.M{}

	// Filter by problem ID if provided
	if problemID != "" {
		filter["problem_id"] = problemID
	}

	// Filter by user ID if provided or if "my_submissions" is true
	if userIDParam != "" && isAdmin {
		// Only admins can see other users' submissions
		userObjID, err := primitive.ObjectIDFromHex(userIDParam)
		if err == nil {
			filter["user_id"] = userObjID
		}
	} else if filterByUser && !userID.IsZero() {
		// User wants to see their own submissions
		filter["user_id"] = userID
	}

	// If not admin and not filtering by own submissions, only show public submissions
	// (In a real system, you might have a "public" flag on submissions)

	// Set up pagination
	limit := 50
	page := 0
	if pageStr := query.Get("page"); pageStr != "" {
		if pageNum, err := utils.ParseInt(pageStr); err == nil && pageNum > 0 {
			page = pageNum - 1 // Convert to 0-indexed
		}
	}

	// Query database
	submissionsCollection := database.GetCollection("OJ", "submissions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set up options for sorting and pagination
	findOptions := options.Find().
		SetSort(bson.D{{Key: "submitted_at", Value: -1}}). // Sort by newest first
		SetSkip(int64(page * limit)).
		SetLimit(int64(limit))

	cursor, err := submissionsCollection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Failed to query submissions: %v", err)
		utils.SendJSONError(w, "Failed to retrieve submissions", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	// Decode submissions
	var submissions []models.Submission
	if err := cursor.All(ctx, &submissions); err != nil {
		log.Printf("Failed to decode submissions: %v", err)
		utils.SendJSONError(w, "Failed to process submissions data", http.StatusInternalServerError)
		return
	}

	// Fetch user and problem details to create list items
	var submissionItems []models.SubmissionListItem
	userCache := make(map[primitive.ObjectID]string)
	problemCache := make(map[string]string)

	usersCollection := database.GetCollection("OJ", "users")
	problemsCollection := database.GetCollection("OJ", "problems")

	for _, sub := range submissions {
		item := models.SubmissionListItem{
			ID:              sub.ID,
			UserID:          sub.UserID,
			ProblemID:       sub.ProblemID,
			Language:        sub.Language,
			Status:          sub.Status,
			ExecutionTimeMs: sub.ExecutionTimeMs,
			SubmittedAt:     sub.SubmittedAt,
		}

		// Get username (use cache to avoid repeated DB lookups)
		if username, found := userCache[sub.UserID]; found {
			item.Username = username
		} else {
			var user models.User
			if err := usersCollection.FindOne(ctx, bson.M{"_id": sub.UserID}).Decode(&user); err == nil {
				item.Username = user.Username
				userCache[sub.UserID] = user.Username
			}
		}

		// Get problem title
		if title, found := problemCache[sub.ProblemID]; found {
			item.ProblemTitle = title
		} else {
			var problem models.Problem
			if err := problemsCollection.FindOne(ctx, bson.M{"problem_id": sub.ProblemID}).Decode(&problem); err == nil {
				item.ProblemTitle = problem.Title
				problemCache[sub.ProblemID] = problem.Title
			}
		}

		submissionItems = append(submissionItems, item)
	}

	// Count total submissions for pagination info
	totalCount, err := submissionsCollection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("Failed to count submissions: %v", err)
		// Continue anyway, just won't have accurate pagination info
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submissions": submissionItems,
		"pagination": map[string]interface{}{
			"total":       totalCount,
			"page":        page + 1, // Convert back to 1-indexed for client
			"limit":       limit,
			"total_pages": (totalCount + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetSubmissionDetailsHandler retrieves details of a specific submission
func GetSubmissionDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendJSONError(w, "Method not allowed. Only GET is accepted.", http.StatusMethodNotAllowed)
		return
	}

	// Extract submission ID from URL
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		utils.SendJSONError(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	submissionIDStr := parts[len(parts)-1]

	submissionID, err := primitive.ObjectIDFromHex(submissionIDStr)
	if err != nil {
		utils.SendJSONError(w, "Invalid submission ID format", http.StatusBadRequest)
		return
	}

	// Get user ID from JWT token
	var userID primitive.ObjectID
	var isAdmin bool

	cookie, err := r.Cookie("authToken")
	if err == nil {
		// Token exists, parse it
		tokenStr := cookie.Value
		claims := &types.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err == nil && token.Valid {
			userID, _ = primitive.ObjectIDFromHex(claims.UserID)
			isAdmin = claims.IsAdmin
		}
	}

	// Query database
	submissionsCollection := database.GetCollection("OJ", "submissions")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var submission models.Submission
	err = submissionsCollection.FindOne(ctx, bson.M{"_id": submissionID}).Decode(&submission)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			utils.SendJSONError(w, "Submission not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to retrieve submission: %v", err)
			utils.SendJSONError(w, "Failed to retrieve submission details", http.StatusInternalServerError)
		}
		return
	}

	// Check if user has permission to view this submission
	if !isAdmin && submission.UserID != userID {
		utils.SendJSONError(w, "You don't have permission to view this submission", http.StatusForbidden)
		return
	}

	// Fetch associated user and problem details
	var user models.User
	var problem models.Problem

	usersCollection := database.GetCollection("OJ", "users")
	problemsCollection := database.GetCollection("OJ", "problems")

	_ = usersCollection.FindOne(ctx, bson.M{"_id": submission.UserID}).Decode(&user)
	_ = problemsCollection.FindOne(ctx, bson.M{"problem_id": submission.ProblemID}).Decode(&problem)

	// Read code file
	submissionDir := filepath.Join("./submissions", submissionID.Hex())
	var fileExtension string
	switch strings.ToLower(submission.Language) {
	case "python":
		fileExtension = ".py"
	case "javascript":
		fileExtension = ".js"
	case "cpp":
		fileExtension = ".cpp"
	case "java":
		fileExtension = ".java"
	default:
		fileExtension = ".txt"
	}

	codeFilePath := filepath.Join(submissionDir, "code"+fileExtension)
	code, err := os.ReadFile(codeFilePath)
	if err != nil {
		log.Printf("Failed to read code file: %v", err)
		code = []byte("// Code file not found")
	}

	// Read test case status if available
	testCaseStatusPath := filepath.Join(submissionDir, "testcasesStatus.txt")
	testCaseStatus, err := os.ReadFile(testCaseStatusPath)
	if err != nil {
		log.Printf("Failed to read test case status file: %v", err)
		testCaseStatus = []byte("Test case results not available")
	}

	// Create response object
	response := struct {
		models.Submission
		Username       string `json:"username,omitempty"`
		ProblemTitle   string `json:"problem_title,omitempty"`
		Code           string `json:"code,omitempty"`
		TestCaseStatus string `json:"test_case_status,omitempty"`
	}{
		Submission:     submission,
		Username:       user.Username,
		ProblemTitle:   problem.Title,
		Code:           string(code),
		TestCaseStatus: string(testCaseStatus),
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
