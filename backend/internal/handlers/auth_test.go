package handlers

import (
	"backend/internal/database"
	"backend/internal/middleware"
	"backend/internal/types"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// No need to redeclare jwtKey here, it's accessible from the main package.

// TestMain will be executed before any tests in this package.
func TestMain(m *testing.M) {
	// Load environment variables.
	// The path is relative to the location of auth_test.go
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatalf("Error loading .env file in TestMain: %v. Make sure .env is in backend folder.", err)
	}

	// Initialize JWT Key from environment after loading .env
	// This ensures jwtKey is set before any other init() or test tries to use it.
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		log.Fatalf("FATAL: JWT_SECRET_KEY not set in environment for tests. Cannot run auth tests.")
	}
	// Set the package-level jwtKey from auth.go
	// This assignment should ideally be to the package variable defined in auth.go.
	// If jwtKey is not exported, this direct assignment might need rethinking,
	// but since they are in the same package, it should work.
	jwtKey = []byte(jwtSecret)
	if jwtKey == nil { // Double check, though previous fatalf should catch empty secret
		log.Fatalf("FATAL: jwtKey is nil after attempting to set from JWT_SECRET_KEY.")
	}
	log.Println("TestMain: JWT_SECRET_KEY loaded and jwtKey initialized.")

	// Database Setup
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Println("MONGO_URI not found in .env, ConnectDB will use default.")
		// ConnectDB handles empty string by using default "mongodb://localhost:27017"
	}

	if err := database.ConnectDB(mongoURI); err != nil {
		log.Fatalf("Failed to connect to MongoDB in TestMain: %v", err)
	}
	log.Println("TestMain: Database connection established.")

	// Run the tests
	exitCode := m.Run()

	// Database Teardown (optional, but good practice)
	// log.Println("TestMain: Closing database connection...") // For debugging
	database.DisconnectDB()
	log.Println("TestMain: Database connection closed.")

	os.Exit(exitCode)
}

func init() {
	// Environment variables are now loaded by TestMain.
	// The primary responsibility of this init can be reduced or re-evaluated.
	// However, ensuring jwtKey is set is still critical if TestMain's order isn't guaranteed
	// before other package init() functions (like auth.go's init).
	// TestMain runs before tests in *this* package, but package var initialization order is complex.

	// Let's ensure jwtKey is set from the env, as TestMain should have loaded it.
	// This might be redundant if TestMain's jwtKey setting is sufficient and correctly timed.
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		// This panic might be too aggressive if TestMain is expected to handle this.
		// Consider logging or relying on TestMain's fatal exit.
		// For now, keeping it to ensure visibility if TestMain's setup has issues.
		fmt.Println("Warning: JWT_SECRET_KEY not found by auth_test.go init(). TestMain should handle this.")
		// Panic here might prevent tests from running if TestMain hasn't set it yet.
		// Let's rely on TestMain's fatal error for JWT_SECRET_KEY.
		// If TestMain ran, jwtKey (the package var) should already be set.
	}
	// The actual setting of jwtKey is now in TestMain to centralize it
	// and ensure it happens before DB connection that might indirectly depend on it via other handlers.
	// if jwtKey == nil && secret != "" { // jwtKey is the package var from auth.go
	// 	 jwtKey = []byte(secret)
	// }

	// The original logic in auth_test.go's init for jwtKey:
	// secret := os.Getenv("JWT_SECRET_KEY")
	// if secret == "" {
	// 	panic("FATAL: JWT_SECRET_KEY not set for tests. Cannot run auth tests.")
	// }
	// if jwtKey == nil {
	// 	jwtKey = []byte(secret)
	// }
	// This is now effectively handled and centralized in TestMain.
	// We can leave this init() empty or remove it if TestMain covers all pre-test setup for this package.
	fmt.Println("auth_test.go init() completed.")
}

// Helper function to parse cookies from a ResponseRecorder
func getCookie(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	cookies := rr.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func TestRegisterHandler_Success(t *testing.T) {
	// Note: This test requires a clean state (user with this email/username doesn't exist)
	// or a proper test database setup/teardown.

	regPayload := types.RegisterationPayload{
		Firstname: "Test",
		Lastname:  "User",
		Username:  fmt.Sprintf("testuser_%d", time.Now().UnixNano()),         // Use unique username
		Email:     fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()), // Use unique email
		Password:  "password123",
	}
	payloadBytes, _ := json.Marshal(regPayload)
	req, err := http.NewRequest("POST", "/register", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(middleware.WithCORS(RegisterHandler)) // Test with CORS

	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v. Body: %s",
			status, http.StatusCreated, rr.Body.String())
	}

	// Check for the authToken cookie
	authTokenCookie := getCookie(rr, "authToken")
	if authTokenCookie == nil {
		t.Fatal("authToken cookie not found after registration")
	}

	// Verify cookie attributes
	if !authTokenCookie.HttpOnly {
		t.Error("authToken cookie not marked HttpOnly")
	}
	// Check Secure based on your local setup (false for HTTP)
	// if !authTokenCookie.Secure { // Uncomment if testing with HTTPS and Secure: true
	// 	t.Error("authToken cookie not marked Secure")
	// }
	if authTokenCookie.Path != "/" {
		t.Errorf("authToken cookie Path is incorrect: got %v want /", authTokenCookie.Path)
	}
	if authTokenCookie.Value == "" {
		t.Error("authToken cookie value is empty")
	}

	// Optionally, verify the response body message and insertedID
	var responseBody map[string]interface{} // Declared locally
	err = json.Unmarshal(rr.Body.Bytes(), &responseBody)
	if err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	message, ok := responseBody["message"].(string)
	if !ok || message != "User Registered Successfully" {
		t.Errorf("Unexpected success message: got %v", message)
	}

	insertedID, ok := responseBody["insertedID"].(string)
	if !ok || insertedID == "" {
		t.Errorf("Unexpected or empty insertedID: got %v", insertedID)
	}

	t.Log("TestRegisterHandler_Success passed")
}

func TestRegisterHandler_ValidationErrors(t *testing.T) {
	// Test case: missing required fields
	regPayload := types.RegisterationPayload{
		Firstname: "Test",
		Lastname:  "User",
		Username:  "testuser",
		Email:     "", // Missing email
		Password:  "password123",
	}
	payloadBytes, _ := json.Marshal(regPayload)
	req, err := http.NewRequest("POST", "/register", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(middleware.WithCORS(RegisterHandler))

	handler.ServeHTTP(rr, req)

	// Check the status code - expect Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for missing email: got %v want %v. Body: %s",
			status, http.StatusBadRequest, rr.Body.String())
	}

	// Check for specific error message (optional but good)
	var errorResponse map[string]string // Declared locally
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	if msg, ok := errorResponse["message"]; ok && !strings.Contains(msg, "All fields") {
		t.Errorf("Unexpected error message for missing email: %v", msg)
	}

	// Add more test cases for other validation errors (short password, invalid email format, short username)

	t.Log("TestRegisterHandler_ValidationErrors passed (missing email)")
}

func TestRegisterHandler_DuplicateUser(t *testing.T) {
	// This test will first register a user, then attempt to register the same user again
	// to check for the duplicate error.
	uniqueSuffix := time.Now().UnixNano() // Ensure unique user for each full test run
	regPayload := types.RegisterationPayload{
		Firstname: "Duplicate",
		Lastname:  "Test",
		Username:  fmt.Sprintf("duplicatetest_%d", uniqueSuffix),
		Email:     fmt.Sprintf("duplicate_%d@example.com", uniqueSuffix),
		Password:  "password123",
	}
	payloadBytes, _ := json.Marshal(regPayload)

	// First attempt - should succeed
	req1, err := http.NewRequest("POST", "/register", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	handler := http.HandlerFunc(middleware.WithCORS(RegisterHandler))
	handler.ServeHTTP(rr1, req1)

	if status := rr1.Code; status != http.StatusCreated {
		t.Fatalf("First registration attempt failed: got status %v, want %v. Body: %s", status, http.StatusCreated, rr1.Body.String())
	}

	// Second attempt with the same payload - should fail with Conflict
	// Need to re-create the reader for the second request as the body of req1 was already read.
	payloadBytes2, _ := json.Marshal(regPayload) // Re-marshal or ensure reader is fresh
	req2, err := http.NewRequest("POST", "/register", bytes.NewReader(payloadBytes2))
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	// handler is reusable
	handler.ServeHTTP(rr2, req2)

	// Check the status code - expect Conflict
	if status := rr2.Code; status != http.StatusConflict {
		t.Errorf("handler returned wrong status code for duplicate user: got %v want %v. Body: %s",
			status, http.StatusConflict, rr2.Body.String())
	}

	// Check for specific error message
	var errorResponse map[string]string
	json.Unmarshal(rr2.Body.Bytes(), &errorResponse)
	if msg, ok := errorResponse["message"]; ok && !strings.Contains(strings.ToLower(msg), "already exists") {
		t.Errorf("Unexpected error message for duplicate user: got '%s', expected to contain 'already exists'", msg)
	}

	t.Log("TestRegisterHandler_DuplicateUser passed")
}

// Note: For TestLoginHandler_Success, you would need to ensure a user exists in the DB first.
// This test assumes a user "testuser@example.com" with password "password123" exists.
func TestLoginHandler_Success(t *testing.T) {
	// Setup: Register the user first to ensure they exist.
	// Use a unique email/username for this test instance to avoid conflicts if tests run repeatedly
	// or if cleanup isn't perfect. However, for this specific test, we often use a known user.
	// Let's stick to the "testuser@example.com" for now and ensure it's created.
	// If this user might already exist from a previous failed run without DB clear, this could be an issue.
	// A robust way would be to attempt to delete this user before trying to create them,
	// or use truly unique credentials for each test run of TestLoginHandler_Success.
	// For now, we'll assume TestMain gives us a clean enough state for unique indexes to work.

	loginEmail := fmt.Sprintf("logintestuser_%d@example.com", time.Now().UnixNano())
	loginUsername := fmt.Sprintf("logintestuser_%d", time.Now().UnixNano())
	loginPassword := "password123"

	regPayload := types.RegisterationPayload{
		Firstname: "Login",
		Lastname:  "Tester",
		Username:  loginUsername,
		Email:     loginEmail,
		Password:  loginPassword,
	}
	regPayloadBytes, _ := json.Marshal(regPayload)
	regReq, err := http.NewRequest("POST", "/register", bytes.NewReader(regPayloadBytes))
	if err != nil {
		t.Fatalf("Setup: Failed to create registration request: %v", err)
	}
	regReq.Header.Set("Content-Type", "application/json")
	rrReg := httptest.NewRecorder()
	registerHandler := http.HandlerFunc(middleware.WithCORS(RegisterHandler))
	registerHandler.ServeHTTP(rrReg, regReq)

	if status := rrReg.Code; status != http.StatusCreated {
		// If registration fails (e.g. user already exists from a previous run and DB wasn't cleaned),
		// we can try to proceed, assuming the user might exist. But it's better to fail early if setup isn't clean.
		// However, for a login test, if the user already exists, that's fine for the login part.
		// Let's check if it's a conflict error (409), which means the user is already there.
		if status != http.StatusConflict { // If it's not 201 and not 409, then it's an unexpected error
			t.Fatalf("Setup: Registration failed with unexpected status: got %v, want %v or %v. Body: %s",
				status, http.StatusCreated, http.StatusConflict, rrReg.Body.String())
		}
		log.Printf("TestLoginHandler_Success: Setup user %s may have already existed (status %d), proceeding with login test.", loginEmail, status)
	} else {
		log.Printf("TestLoginHandler_Success: Successfully registered user %s for login test.", loginEmail)
	}

	// Now, attempt to login with the (now hopefully) existing user
	loginPayload := types.LoginPayload{
		Email:    loginEmail,    // Use the email of the user we just tried to register
		Password: loginPassword, // Use the password of the user we just tried to register
	}
	payloadBytes, _ := json.Marshal(loginPayload)
	req, err := http.NewRequest("POST", "/login", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(middleware.WithCORS(LoginHandler)) // Test with CORS

	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v. Body: %s",
			status, http.StatusOK, rr.Body.String())
	}

	// Check for the authToken cookie
	authTokenCookie := getCookie(rr, "authToken")
	if authTokenCookie == nil {
		t.Fatal("authToken cookie not found after login")
	}

	// Verify cookie attributes
	if !authTokenCookie.HttpOnly {
		t.Error("authToken cookie not marked HttpOnly")
	}
	// Check Secure based on your local setup (false for HTTP)
	// if !authTokenCookie.Secure { // Uncomment if testing with HTTPS and Secure: true
	// 	t.Error("authToken cookie not marked Secure")
	// }
	if authTokenCookie.Path != "/" {
		t.Errorf("authToken cookie Path is incorrect: got %v want /", authTokenCookie.Path)
	}
	if authTokenCookie.Value == "" {
		t.Error("authToken cookie value is empty")
	}

	// Optionally, verify the response body (should not contain the token)
	var responseBody map[string]interface{} // Declared locally
	err = json.Unmarshal(rr.Body.Bytes(), &responseBody)
	if err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	message, ok := responseBody["message"].(string)
	if !ok || message != "Login Successful" {
		t.Errorf("Unexpected success message: got %v", message)
	}

	if _, exists := responseBody["token"]; exists {
		t.Error("Token found in response body, should be in cookie")
	}

	t.Log("TestLoginHandler_Success passed")
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	loginPayload := types.LoginPayload{
		Email:    "nonexistent@example.com", // User that doesn't exist
		Password: "wrongpassword",
	}
	payloadBytes, _ := json.Marshal(loginPayload)
	req, err := http.NewRequest("POST", "/login", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(middleware.WithCORS(LoginHandler))

	handler.ServeHTTP(rr, req)

	// Check the status code - expect Unauthorized
	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code for invalid credentials: got %v want %v. Body: %s",
			status, http.StatusUnauthorized, rr.Body.String())
	}

	// Check for specific error message
	var errorResponse map[string]string // Declared locally
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	if msg, ok := errorResponse["message"]; ok && !strings.Contains(msg, "Invalid username/email or password") {
		t.Errorf("Unexpected error message for invalid credentials: %v", msg)
	}

	t.Log("TestLoginHandler_InvalidCredentials passed")
}

func TestLoginHandler_MissingFields(t *testing.T) {
	loginPayload := types.LoginPayload{
		Email:    "testuser@example.com",
		Password: "", // Missing password
	}
	payloadBytes, _ := json.Marshal(loginPayload)
	req, err := http.NewRequest("POST", "/login", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(middleware.WithCORS(LoginHandler))

	handler.ServeHTTP(rr, req)

	// Check the status code - expect Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for missing password: got %v want %v. Body: %s",
			status, http.StatusBadRequest, rr.Body.String())
	}

	// Check for specific error message
	var errorResponse map[string]string // Declared locally
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	if msg, ok := errorResponse["message"]; ok && !strings.Contains(msg, "Username/Email and password are required") {
		t.Errorf("Unexpected error message for missing password: %v", msg)
	}

	t.Log("TestLoginHandler_MissingFields passed")
}
