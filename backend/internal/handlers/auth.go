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
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey []byte

func init() {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Println("CRITICAL: JWT_SECRET_KEY not found in environment variables during init.")
		return
	}
}
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	var payload types.RegisterationPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Println("Invalid registration request payload:", err)
		utils.SendJSONError(w, "Invalid request payload. Ensure all fields are valid JSON strings.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if payload.Firstname == "" || payload.Lastname == "" || payload.Username == "" || payload.Password == "" || payload.Email == "" {
		utils.SendJSONError(w, "All fields (firstname, lastname, username, email, password) are required.", http.StatusBadRequest)
		return
	}
	if len(payload.Password) < 8 {
		utils.SendJSONError(w, "Password must be at least 8 characters long.", http.StatusBadRequest)
		return
	}
	if !isValidEmail(payload.Email) {
		utils.SendJSONError(w, "Invalid email address.", http.StatusBadRequest)
		return
	}

	if len(payload.Username) < 3 { // Add to Frontend Validation
		utils.SendJSONError(w, "Username must be at least 3 characters long.", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Failed to hash password:", err)
		utils.SendJSONError(w, "Failed to complete password hashing.", http.StatusInternalServerError)
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
			utils.SendJSONError(w, "Username or email already exists.", http.StatusConflict)
			return
		}
		log.Printf("Failed to register user in DB: %v\n", err)
		utils.SendJSONError(w, fmt.Sprintf("Failed to register user: %v", err), http.StatusInternalServerError)
		return
	}

	var userIDHex string
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		userIDHex = oid.Hex()
	} else {
		log.Println("Failed to retrieve generated user ID for token. InsertedID was not an ObjectID.")
		utils.SendJSONError(w, "Failed to retrieve generated user ID for token.", http.StatusInternalServerError)
		return
	}

	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Println("CRITICAL: JWT_SECRET_KEY not found in environment variables.")
		utils.SendJSONError(w, "User registered, but server configuration error prevented token generation.", http.StatusInternalServerError)
		return
	}
	jwtKey = []byte(secret)

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &types.Claims{
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
		utils.SendJSONError(w, "User registered, but server configuration error prevented token generation.", http.StatusInternalServerError)
		return
	}

	// Set the JWT as an HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "authToken", // Name of the cookie
		Value:    tokenString, // The JWT token string
		Expires:  expirationTime,
		HttpOnly: true,                 // Make it HTTP-only
		Secure:   false,                // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode, // Recommended for CSRF protection
		Path:     "/",                  // Make the cookie available to all paths
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	// Send a success message without the token in the body
	response := map[string]interface{}{
		"message":    "User Registered Successfully",
		"insertedID": userIDHex,
		// Do NOT include the token in the response body
	}
	json.NewEncoder(w).Encode(response)
	log.Printf("User registered: %s (Firstname: %s, Lastname: %s, Email: %s, UserID: %s)\n", newUser.Username, newUser.Firstname, newUser.Lastname, newUser.Email, userIDHex)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	var payload types.LoginPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Println("Invalid login request payload:", err)
		utils.SendJSONError(w, "Invalid request payload. Ensure email and password are provided as valid JSON strings.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if payload.Email == "" || payload.Password == "" {
		utils.SendJSONError(w, "Username/Email and password are required.", http.StatusBadRequest)
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
	if err == mongo.ErrNoDocuments {
		utils.SendJSONError(w, "Invalid username/email or password.", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Database error during login: %v\n", err)
		utils.SendJSONError(w, "Internal server error during login.", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(payload.Password))
	if err != nil {
		utils.SendJSONError(w, "Invalid username/email or password.", http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Println("CRITICAL: JWT_SECRET_KEY not found in environment variables.")
		utils.SendJSONError(w, "Server configuration error prevented token generation.", http.StatusInternalServerError)
		return
	}
	jwtKey = []byte(secret)

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &types.Claims{
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
		log.Printf("Failed to sign JWT token: %v\n", err)
		utils.SendJSONError(w, "Server configuration error prevented token generation.", http.StatusInternalServerError)
		return
	}

	// Set the JWT as an HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "authToken", // Name of the cookie
		Value:    tokenString, // The JWT token string
		Expires:  expirationTime,
		HttpOnly: true,                 // Make it HTTP-only
		Secure:   false,                // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode, // Recommended for CSRF protection
		Path:     "/",                  // Make the cookie available to all paths
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Send a success message without the token in the body
	response := map[string]string{
		"message": "Login Successful",
		// Do NOT include the token in the response body
	}
	json.NewEncoder(w).Encode(response)
	log.Printf("User logged in: %s (Email: %s)\n", foundUser.Username, foundUser.Email)
}
