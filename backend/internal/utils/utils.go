package utils

import (
	"backend/internal/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	var result types.ExecutionResult
	var err error
	var cmd *exec.Cmd

	// Create temporary directory for execution
	tempDir, err := os.MkdirTemp("", "codejudge-exec-*")
	if err != nil {
		return result, fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file for input if needed
	inputPath := ""
	if req.Input != "" {
		inputPath = filepath.Join(tempDir, "input.txt")
		if err := os.WriteFile(inputPath, []byte(req.Input), 0644); err != nil {
			return result, fmt.Errorf("failed to write input file: %v", err)
		}
	}

	// Set up command based on language
	switch strings.ToLower(req.Language) {
	case "python":
		// For Python, write code to file and execute
		scriptPath := filepath.Join(tempDir, "script.py")
		if err := os.WriteFile(scriptPath, []byte(req.Code), 0644); err != nil {
			return result, fmt.Errorf("failed to write Python code file: %v", err)
		}
		cmd = exec.CommandContext(ctx, "python3", scriptPath)

	case "javascript":
		// For JavaScript, write code to file and execute with Node.js
		scriptPath := filepath.Join(tempDir, "script.js")
		if err := os.WriteFile(scriptPath, []byte(req.Code), 0644); err != nil {
			return result, fmt.Errorf("failed to write JavaScript code file: %v", err)
		}
		cmd = exec.CommandContext(ctx, "node", scriptPath)

	case "cpp":
		// For C++, write code, compile, and execute
		sourcePath := filepath.Join(tempDir, "source.cpp")
		execPath := filepath.Join(tempDir, "executable")

		if err := os.WriteFile(sourcePath, []byte(req.Code), 0644); err != nil {
			return result, fmt.Errorf("failed to write C++ code file: %v", err)
		}

		// Compile first
		compileCmd := exec.Command("g++", "-o", execPath, sourcePath)
		compileOutput, err := compileCmd.CombinedOutput()
		if err != nil {
			return types.ExecutionResult{
				Output:          string(compileOutput),
				ExecutionTimeMs: 0,
				MemoryUsedKB:    0,
				Status:          "compilation_error",
			}, fmt.Errorf("compilation error: %v", err)
		}

		cmd = exec.CommandContext(ctx, execPath)

	case "java":
		// For Java, write code, compile, and execute
		// This is simplified - should handle class name extraction
		mainClass := "Main" // Assuming the main class is named "Main"
		sourcePath := filepath.Join(tempDir, mainClass+".java")

		if err := os.WriteFile(sourcePath, []byte(req.Code), 0644); err != nil {
			return result, fmt.Errorf("failed to write Java code file: %v", err)
		}

		// Compile first
		compileCmd := exec.Command("javac", sourcePath)
		compileOutput, err := compileCmd.CombinedOutput()
		if err != nil {
			return types.ExecutionResult{
				Output:          string(compileOutput),
				ExecutionTimeMs: 0,
				MemoryUsedKB:    0,
				Status:          "compilation_error",
			}, fmt.Errorf("compilation error: %v", err)
		}

		cmd = exec.CommandContext(ctx, "java", "-cp", tempDir, mainClass)

	default:
		return result, fmt.Errorf("unsupported language: %s", req.Language)
	}

	// Set up input/output
	if req.Input != "" {
		cmd.Stdin = strings.NewReader(req.Input)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = tempDir

	// Measure execution time
	startTime := time.Now()

	// Run the command
	runErr := cmd.Run()

	executionTime := time.Since(startTime).Milliseconds()

	// Prepare result
	result.ExecutionTimeMs = int(executionTime)
	result.MemoryUsedKB = 0 // We don't measure memory yet

	// Combine stdout and stderr for the output
	if stderr.Len() > 0 {
		stderrStr := stderr.String()
		result.Output = stderrStr

		// For Python, better classify errors
		if strings.ToLower(req.Language) == "python" {
			// Check for common Python errors that should be treated as compilation errors
			if strings.Contains(stderrStr, "SyntaxError") ||
				strings.Contains(stderrStr, "IndentationError") ||
				strings.Contains(stderrStr, "TabError") ||
				strings.Contains(stderrStr, "NameError") || // Common typos like 'prinft' instead of 'print'
				strings.Contains(stderrStr, "ImportError") ||
				strings.Contains(stderrStr, "ModuleNotFoundError") {
				result.Status = "compilation_error"
			} else {
				result.Status = "runtime_error"
			}
		} else {
			result.Status = "runtime_error"
		}
	} else {
		result.Output = stdout.String()
		result.Status = "success"
	}

	// Handle timeout and other errors
	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "time_limit_exceeded"
		return result, fmt.Errorf("context deadline exceeded")
	} else if runErr != nil {
		// Check if it's a compilation error (should be caught earlier)
		// or runtime error
		result.Status = "runtime_error"
		return result, runErr
	}

	return result, nil
}
