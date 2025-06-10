package utils

import (
	"backend/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Helper function to send JSON errors
func SendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

// ParseInt parses a string to an integer with error handling
func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// ExecuteCode executes code with the given input and returns the result
// This is a simplified implementation - in a production environment, you would use
// a sandboxed execution environment for security
func ExecuteCode(ctx context.Context, req types.ExecutionRequest) (types.ExecutionResult, error) {
	var _ *exec.Cmd // Using _ to ignore the unused variable
	var result types.ExecutionResult

	// Create temporary files for the code and input
	// In a real implementation, this would use proper temp files and cleanup

	// Set up command based on language
	switch strings.ToLower(req.Language) {
	case "python":
		// For Python, we can execute directly with the code file
		// cmd = exec.CommandContext(ctx, "python3", req.Code)
	case "javascript":
		// For JavaScript, we can use Node.js
		// cmd = exec.CommandContext(ctx, "node", req.Code)
	case "cpp":
		// For C++, we need to compile and then execute
		// This is simplified - real implementation would compile first
		// cmd = exec.CommandContext(ctx, "g++", "-o", "temp_executable", req.Code, "&&", "./temp_executable")
	case "java":
		// For Java, we need to compile and then execute
		// This is simplified - real implementation would handle class names
		// cmd = exec.CommandContext(ctx, "javac", req.Code, "&&", "java", "Main")
	default:
		return result, fmt.Errorf("unsupported language: %s", req.Language)
	}

	// In a real implementation, you would:
	// 1. Write the code to a file
	// 2. Compile if necessary
	// 3. Execute with proper resource limits
	// 4. Measure execution time and memory usage
	// 5. Clean up temporary files

	// For this simplified version, we'll just return a mock result
	// In a real implementation, you would capture stdout/stderr and measure resources

	startTime := time.Now()

	// This is where you would actually run the command and capture output
	// output, err := cmd.CombinedOutput()

	// For now, let's simulate execution
	time.Sleep(100 * time.Millisecond) // Simulate execution time
	executionTime := time.Since(startTime).Milliseconds()

	// For demonstration, return a mock result
	// In a real implementation, this would be the actual output and measurements
	result = types.ExecutionResult{
		Output:          "Sample output for " + req.Language + " code\nWith input: " + req.Input,
		ExecutionTimeMs: int(executionTime),
		MemoryUsedKB:    1024, // Mock memory usage
		Status:          "success",
	}

	return result, nil
}
