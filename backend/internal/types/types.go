package types

import "github.com/golang-jwt/jwt/v5"

// This is the struct for the JWT token
type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	IsAdmin   bool   `json:"is_admin"`
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
