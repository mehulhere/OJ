package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// No need to redeclare jwtKey here, it's accessible from the main package.

func init() {
	// Load environment variables for tests if not already loaded
	err := godotenv.Load("../../../.env") // Adjust path as needed
	if err != nil {
		fmt.Println("Warning: Error loading .env file in tests. Ensure environment variables are set.")
	}

	// Ensure jwtKey is set for tests - setting the package-level jwtKey from main.go
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		panic("FATAL: JWT_SECRET_KEY not set for tests. Cannot run auth tests.")
	}
	// Only set if it hasn't been set already (e.g., by main's init)
	if jwtKey == nil {
		jwtKey = []byte(secret)
	}

	// Note: Database connection might be needed for some tests, but for basic handler tests,
	// we might mock it or assume a running DB. For these auth tests, a running DB is assumed.
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
	regPayload := RegisterationPayload{
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
	handler := http.HandlerFunc(withCORS(registerHandler)) // Test with CORS

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
	regPayload := RegisterationPayload{
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
	handler := http.HandlerFunc(registerHandler)

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
	// Note: This test requires a user to already exist with the provided email/username.
	// You might need to register a user first in a setup phase.
	regPayload := RegisterationPayload{
		Firstname: "Existing",
		Lastname:  "User",
		Username:  "existing_test_user",   // Use an existing username
		Email:     "existing@example.com", // Use an existing email
		Password:  "password123",
	}
	payloadBytes, _ := json.Marshal(regPayload)
	req, err := http.NewRequest("POST", "/register", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(registerHandler)

	handler.ServeHTTP(rr, req)

	// Check the status code - expect Conflict
	if status := rr.Code; status != http.StatusConflict {
		t.Errorf("handler returned wrong status code for duplicate user: got %v want %v. Body: %s",
			status, http.StatusConflict, rr.Body.String())
	}

	// Check for specific error message
	var errorResponse map[string]string // Declared locally
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	if msg, ok := errorResponse["message"]; ok && !strings.Contains(msg, "already exists") {
		t.Errorf("Unexpected error message for duplicate user: %v", msg)
	}

	t.Log("TestRegisterHandler_DuplicateUser passed")
}

// Note: For TestLoginHandler_Success, you would need to ensure a user exists in the DB first.
// This test assumes a user "testuser@example.com" with password "password123" exists.
func TestLoginHandler_Success(t *testing.T) {
	loginPayload := LoginPayload{
		Email:    "testuser@example.com", // Use existing user's email or username
		Password: "password123",          // Use existing user's password
	}
	payloadBytes, _ := json.Marshal(loginPayload)
	req, err := http.NewRequest("POST", "/login", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(withCORS(loginHandler)) // Test with CORS

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
	loginPayload := LoginPayload{
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
	handler := http.HandlerFunc(loginHandler)

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
	loginPayload := LoginPayload{
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
	handler := http.HandlerFunc(loginHandler)

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
