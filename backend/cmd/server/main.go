package main

import (
	"backend/internal/database"
	"backend/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// This is the struct for the JWT token
type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	jwt.RegisteredClaims
}

// This is the struct for the registration payload
type RegisterationPayload struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// This is the struct for the login payload
type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// This is the struct for the code execution payload
type ExecuteCodePayload struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Stdin    string `json:"stdin"` // Optional standard input
}

// This is the struct for the code execution result
type ExecuteCodeResult struct {
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	ExecutionTimeMs int64  `json:"execution_time_ms"`
	MemoryUsageKb   int64  `json:"memory_usage_kb"` // Placeholder, actual measurement is complex
	Error           string `json:"error,omitempty"` // For errors in the execution service itself
	Status          string `json:"status"`          // e.g., "success", "compile_error", "runtime_error", "timeout"
}

var jwtKey []byte

// Helper function to send JSON errors
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	var payload RegisterationPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Println("Invalid registration request payload:", err)
		sendJSONError(w, "Invalid request payload. Ensure all fields are valid JSON strings.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if payload.Firstname == "" || payload.Lastname == "" || payload.Username == "" || payload.Password == "" || payload.Email == "" {
		sendJSONError(w, "All fields (firstname, lastname, username, email, password) are required.", http.StatusBadRequest)
		return
	}
	if len(payload.Password) < 8 {
		sendJSONError(w, "Password must be at least 8 characters long.", http.StatusBadRequest)
		return
	}
	if !isValidEmail(payload.Email) {
		sendJSONError(w, "Invalid email address.", http.StatusBadRequest)
		return
	}

	if len(payload.Username) < 3 { // Add to Frontend Validation
		sendJSONError(w, "Username must be at least 3 characters long.", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Failed to hash password:", err)
		sendJSONError(w, "Failed to complete password hashing.", http.StatusInternalServerError)
		return
	}

	newUser := models.User{
		Firstname: payload.Firstname,
		Lastname:  payload.Lastname,
		Username:  payload.Username,
		Email:     payload.Email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	usersCollection := database.GetCollection("OJ", "users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := usersCollection.InsertOne(ctx, newUser)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			sendJSONError(w, "Username or email already exists.", http.StatusConflict)
			return
		}
		log.Printf("Failed to register user in DB: %v\n", err)
		sendJSONError(w, fmt.Sprintf("Failed to register user: %v", err), http.StatusInternalServerError)
		return
	}

	var userIDHex string
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		userIDHex = oid.Hex()
	} else {
		log.Println("Failed to retrieve generated user ID for token. InsertedID was not an ObjectID.")
		sendJSONError(w, "Failed to retrieve generated user ID for token.", http.StatusInternalServerError)
		return
	}

	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Println("CRITICAL: JWT_SECRET_KEY not found in environment variables.")
		sendJSONError(w, "User registered, but server configuration error prevented token generation.", http.StatusInternalServerError)
		return
	}
	jwtKey = []byte(secret)

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:    userIDHex,
		Username:  newUser.Username,
		Email:     newUser.Email,
		Firstname: newUser.Firstname,
		Lastname:  newUser.Lastname,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Printf("Failed to sign JWT token: %v\n", err)
		sendJSONError(w, "User registered, but failed to generate token.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]interface{}{
		"message":    "User Registered Successfully",
		"insertedID": userIDHex,
		"token":      tokenString,
	}
	json.NewEncoder(w).Encode(response)
	log.Printf("User registered: %s (Firstname: %s, Lastname: %s, Email: %s, UserID: %s)\n", newUser.Username, newUser.Firstname, newUser.Lastname, newUser.Email, userIDHex)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	var payload LoginPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Println("Invalid login request payload:", err)
		sendJSONError(w, "Invalid request payload. Ensure email and password are provided as valid JSON strings.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if payload.Email == "" || payload.Password == "" {
		sendJSONError(w, "Username/Email and password are required.", http.StatusBadRequest)
		return
	}

	usersCollection := database.GetCollection("OJ", "users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var foundUser models.User

	// Determine if the input is likely an email or a username
	// A simple check for '@' is used here, but a more robust validation (like isValidEmail) could be used.
	isEmail := strings.Contains(payload.Email, "@")

	var filter primitive.M
	if isEmail {
		filter = primitive.M{"email": payload.Email}
		err = usersCollection.FindOne(ctx, filter).Decode(&foundUser)
		if err == mongo.ErrNoDocuments {
			// Not found by email, now try by username
			filter = primitive.M{"username": payload.Email}
			err = usersCollection.FindOne(ctx, filter).Decode(&foundUser)
		}
	} else {
		// Not an email, search by username
		filter = primitive.M{"username": payload.Email}
		err = usersCollection.FindOne(ctx, filter).Decode(&foundUser)
	}

	if err != nil {
		if err == mongo.ErrNoDocuments {
			sendJSONError(w, "Invalid username/email or password.", http.StatusUnauthorized)
			return
		}
		log.Println("Error finding user:", err)
		sendJSONError(w, "Failed to process login due to a server error.", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(payload.Password))
	if err != nil {
		sendJSONError(w, "Invalid username/email or password. ", http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Println("CRITICAL: JWT_SECRET_KEY not found in environment variables during login.")
		sendJSONError(w, "Login successful, but server configuration error prevented token generation.", http.StatusInternalServerError)
		return
	}
	jwtKey = []byte(secret)

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:    foundUser.ID.Hex(),
		Username:  foundUser.Username,
		Email:     foundUser.Email,
		Firstname: foundUser.Firstname,
		Lastname:  foundUser.Lastname,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Println("Failed to sign JWT token during login:", err)
		sendJSONError(w, "Login successful, but failed to generate token.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message": "Login Successful",
		"token":   tokenString,
		"user": map[string]string{
			"user_id":   foundUser.ID.Hex(),
			"username":  foundUser.Username,
			"email":     foundUser.Email,
			"firstname": foundUser.Firstname,
			"lastname":  foundUser.Lastname,
		},
	}
	json.NewEncoder(w).Encode(response)
	log.Println("User logged in:", foundUser.Username)
}

func getProblemsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONError(w, "Method not allowed. Only GET is accepted.", http.StatusMethodNotAllowed)
		return
	}

	problemsCollection := database.GetCollection("OJ", "problems")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := problemsCollection.Find(ctx, primitive.M{})
	if err != nil {
		log.Println("Error fetching problems from DB:", err)
		sendJSONError(w, "Failed to retrieve problems.", http.StatusInternalServerError)
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
		sendJSONError(w, "Error processing problems list.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(problems); err != nil {
		log.Println("Error encoding problems to JSON:", err)
		// If headers are already written, this specific sendJSONError might not be effective.
		// Consider more centralized error handling for such cases.
	}
	log.Println("Successfully retrieved problems list. Count:", len(problems))
}

// getProblemHandler (singular problem)
func getProblemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONError(w, "Method not allowed. Only GET is accepted.", http.StatusMethodNotAllowed)
		return
	}

	pathSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathSegments) < 2 || pathSegments[0] != "problems" {
		sendJSONError(w, "Invalid problem URL format. Expected /problems/{id}", http.StatusBadRequest)
		return
	}
	problemIDFromURL := pathSegments[len(pathSegments)-1]

	if problemIDFromURL == "" {
		sendJSONError(w, "Problem ID is required in the URL path.", http.StatusBadRequest)
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
				sendJSONError(w, "Problem not found (and ID is not a valid ObjectID).", http.StatusNotFound)
				return
			}
			filter = primitive.M{"_id": objectID}
			err = problemsCollection.FindOne(ctx, filter).Decode(&problemData)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					sendJSONError(w, "Problem not found.", http.StatusNotFound)
					return
				}
				log.Println("Error fetching single problem by ObjectID from DB:", err)
				sendJSONError(w, "Failed to retrieve problem.", http.StatusInternalServerError)
				return
			}
		} else {
			log.Println("Error fetching single problem by problem_id from DB:", err)
			sendJSONError(w, "Failed to retrieve problem.", http.StatusInternalServerError)
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

func addTestCaseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	var payload models.AddTestCasePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Invalid add test case request payload:", err)
		sendJSONError(w, "Invalid request payload for test case.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate input (Points and SequenceNumber might have defaults if not provided, or be required)
	if payload.ProblemDBID == "" || payload.Input == "" {
		sendJSONError(w, "ProblemDBID and Input are required for a test case.", http.StatusBadRequest)
		return
	}
	// You might want to add validation for payload.Points and payload.SequenceNumber (e.g., >= 0)

	problemObjectID, err := primitive.ObjectIDFromHex(payload.ProblemDBID)
	if err != nil {
		sendJSONError(w, "Invalid ProblemDBID format. Must be a valid ObjectID hex string.", http.StatusBadRequest)
		return
	}

	problemsCollection := database.GetCollection("OJ", "problems")
	ctxCheck, cancelCheck := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCheck()
	var existingProblem models.Problem
	err = problemsCollection.FindOne(ctxCheck, primitive.M{"_id": problemObjectID}).Decode(&existingProblem)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			sendJSONError(w, "Problem with the given ProblemDBID not found.", http.StatusNotFound)
			return
		}
		log.Println("Error checking for existing problem:", err)
		sendJSONError(w, "Error verifying problem existence.", http.StatusInternalServerError)
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
		sendJSONError(w, "Failed to add test case.", http.StatusInternalServerError)
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

func executeCodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	var payload ExecuteCodePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Invalid execute code request payload:", err)
		sendJSONError(w, "Invalid request payload for code execution.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if payload.Code == "" {
		sendJSONError(w, "Code cannot be empty.", http.StatusBadRequest)
		return
	}
	if payload.Language == "" {
		sendJSONError(w, "Language must be specified.", http.StatusBadRequest)
		return
	}

	result := ExecuteCodeResult{
		// Initialize status, it will be overwritten by actual execution outcomes
		Status: "processing",
	}

	// Default error in case a language isn't handled, or early exit.
	// This will be cleared if a language handler successfully completes.
	result.Error = "Language not supported or internal error before execution: " + payload.Language

	if payload.Language == "python" {
		tempDir, err := os.MkdirTemp("", "codejudge-python-*")
		if err != nil {
			log.Println("Failed to create temp dir for python:", err)
			// Use sendJSONError for server-internal issues before forming a full 'result'
			sendJSONError(w, "Server error creating python execution environment.", http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(tempDir)

		scriptPath := filepath.Join(tempDir, "script.py")
		if err := os.WriteFile(scriptPath, []byte(payload.Code), 0644); err != nil {
			log.Println("Failed to write python code to temp file:", err)
			sendJSONError(w, "Server error preparing python code for execution.", http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "python3", scriptPath)
		cmd.Stdin = strings.NewReader(payload.Stdin)
		var out, errStream strings.Builder
		cmd.Stdout = &out
		cmd.Stderr = &errStream

		startTime := time.Now()
		runErr := cmd.Run()
		executionTime := time.Since(startTime).Milliseconds()

		result.Stdout = out.String()
		result.Stderr = errStream.String()
		result.ExecutionTimeMs = executionTime
		result.Error = "" // Clear pre-set error as execution proceeded

		if ctx.Err() == context.DeadlineExceeded {
			result.Status = "time_limit_exceeded"
			// Append to stderr, as TLE might not produce its own stderr
			if result.Stderr == "" {
				result.Stderr = "Execution timed out after 2 seconds."
			} else {
				result.Stderr += "\nExecution timed out after 2 seconds."
			}
		} else if runErr != nil {
			// This covers Python syntax errors, runtime errors, etc.
			result.Status = "runtime_error" // Or "syntax_error" - "runtime_error" is a general catch-all
			if result.Stderr == "" {        // If Python didn't output to stderr for some reason
				result.Stderr = runErr.Error()
			}
		} else {
			result.Status = "success"
		}

	} else if payload.Language == "javascript" {
		tempDir, err := os.MkdirTemp("", "codejudge-js-*")
		if err != nil {
			log.Println("Failed to create temp dir for javascript:", err)
			sendJSONError(w, "Server error creating javascript execution environment.", http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(tempDir)

		scriptPath := filepath.Join(tempDir, "script.js")
		if err := os.WriteFile(scriptPath, []byte(payload.Code), 0644); err != nil {
			log.Println("Failed to write javascript code to temp file:", err)
			sendJSONError(w, "Server error preparing javascript code for execution.", http.StatusInternalServerError)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "node", scriptPath)
		cmd.Stdin = strings.NewReader(payload.Stdin)
		var out, errStream strings.Builder
		cmd.Stdout = &out
		cmd.Stderr = &errStream

		startTime := time.Now()
		runErr := cmd.Run()
		executionTime := time.Since(startTime).Milliseconds()

		result.Stdout = out.String()
		result.Stderr = errStream.String()
		result.ExecutionTimeMs = executionTime
		result.Error = "" // Clear pre-set error

		if ctx.Err() == context.DeadlineExceeded {
			result.Status = "time_limit_exceeded"
			if result.Stderr == "" {
				result.Stderr = "Execution timed out after 10 seconds."
			} else {
				result.Stderr += "\nExecution timed out after 10 seconds."
			}
		} else if runErr != nil {
			result.Status = "runtime_error"
			if result.Stderr == "" {
				result.Stderr = runErr.Error()
			}
		} else {
			result.Status = "success"
		}
	} else {
		// If language not supported, result.Error is already set.
		// We might want a specific status for "language_not_supported" if we send 200 OK.
		// For now, the default "processing" status would be overwritten if we had a dedicated 'else' block.
		// Let's ensure result.Status reflects this if we are sending 200 OK.
		if result.Error != "" { // If error is still the default "Language not supported..."
			result.Status = "client_error_language_not_supported" // A more specific status
		}
	}

	w.Header().Set("Content-Type", "application/json")
	// Always send http.StatusOK if we have managed to form a 'result' object.
	// The 'status', 'stderr', and 'error' fields within 'result' will indicate the actual outcome.
	// Errors that prevent forming a 'result' (e.g., bad JSON payload, internal file write errors)
	// are handled by 'sendJSONError' which sends appropriate non-200 statuses.
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error encoding execution result: %v", err)
		// At this point, headers are sent. Can't send a new JSON error.
	}
	log.Printf("Code execution: lang=%s, status=%s, time=%dms, stdout_len=%d, stderr_len=%d, error_msg_len=%d",
		payload.Language, result.Status, result.ExecutionTimeMs, len(result.Stdout), len(result.Stderr), len(result.Error))
}

// This is the middleware for the CORS
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file. Ensure environment variables are set if .env is not used.")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("FATAL: MONGO_URI not found in environment variables.")
	}
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		log.Fatal("FATAL: JWT_SECRET_KEY not found in environment variables.")
	}

	err_db := database.ConnectDB(mongoURI)
	if err_db != nil {
		log.Fatalf("Could not connect to the database: %v", err_db)
	}
	log.Println("Successfully connected to MongoDB.")
	defer database.DisconnectDB()

	http.HandleFunc("/register", withCORS(registerHandler))
	http.HandleFunc("/login", withCORS(loginHandler))
	http.HandleFunc("/problems", withCORS(getProblemsHandler))
	http.HandleFunc("/problems/", withCORS(getProblemHandler))
	http.HandleFunc("/testcases", withCORS(addTestCaseHandler))
	http.HandleFunc("/execute", withCORS(executeCodeHandler))

	log.Println("Server listening on port 8080. Allowed origin for CORS: http://localhost:3000")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
