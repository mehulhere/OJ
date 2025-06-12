package handlers

import (
	"backend/internal/database"
	"backend/internal/types"
	"backend/internal/utils"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetLastCodeHandler returns the most recent code draft (latest submission)
// for the authenticated user for a specific problem (and optional language).
//
//	GET /last-code?problem_id=XYZ&language=python
func GetLastCodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendJSONError(w, "Method not allowed. Only GET is accepted.", http.StatusMethodNotAllowed)
		return
	}

	// Authenticate user via cookie
	cookie, err := r.Cookie("authToken")
	if err != nil {
		utils.SendJSONError(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	tokenStr := cookie.Value
	claims := &types.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		utils.SendJSONError(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	userID, _ := primitive.ObjectIDFromHex(claims.UserID)

	// Query params
	q := r.URL.Query()
	problemID := q.Get("problem_id")
	if problemID == "" {
		utils.SendJSONError(w, "problem_id query param is required", http.StatusBadRequest)
		return
	}
	lang := strings.ToLower(q.Get("language")) // optional

	// Build MongoDB filter
	filter := bson.M{
		"user_id":    userID,
		"problem_id": problemID,
	}
	if lang != "" {
		filter["language"] = lang
	}

	submissionsCollection := database.GetCollection("OJ", "submissions")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := submissionsCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("Failed query in GetLastCodeHandler: %v", err)
		utils.SendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	var subs []struct {
		ID        primitive.ObjectID `bson:"_id"`
		Language  string             `bson:"language"`
		Submitted time.Time          `bson:"submitted_at"`
	}
	if err := cursor.All(ctx, &subs); err != nil {
		log.Printf("Cursor decode error: %v", err)
		utils.SendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	if len(subs) == 0 {
		utils.SendJSONError(w, "No code found", http.StatusNotFound)
		return
	}
	// Sort by SubmittedAt desc (Go <1.21 stable no generics sort)
	sort.Slice(subs, func(i, j int) bool { return subs[i].Submitted.After(subs[j].Submitted) })
	latest := subs[0]

	// Read code file from submissions directory
	submissionDir := filepath.Join("./submissions", latest.ID.Hex())
	var ext string
	switch strings.ToLower(latest.Language) {
	case "python":
		ext = ".py"
	case "javascript":
		ext = ".js"
	case "cpp":
		ext = ".cpp"
	case "java":
		ext = ".java"
	default:
		ext = ".txt"
	}
	codePath := filepath.Join(submissionDir, "code"+ext)
	codeBytes, err := os.ReadFile(codePath)
	if err != nil {
		log.Printf("GetLastCode: cannot read code file: %v", err)
		utils.SendJSONError(w, "Code file not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"code":          string(codeBytes),
		"language":      latest.Language,
		"submission_id": latest.ID.Hex(),
	})
}
