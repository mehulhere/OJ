package handlers

import (
	"backend/internal/types"
	"backend/internal/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func ExecuteCodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendJSONError(w, "Method not allowed. Only POST is accepted.", http.StatusMethodNotAllowed)
		return
	}

	var payload types.ExecuteCodePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Invalid execute code request payload:", err)
		utils.SendJSONError(w, "Invalid request payload for code execution.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if payload.Code == "" {
		utils.SendJSONError(w, "Code cannot be empty.", http.StatusBadRequest)
		return
	}
	if payload.Language == "" {
		utils.SendJSONError(w, "Language must be specified.", http.StatusBadRequest)
		return
	}

	// Process test cases
	var testCases []string
	if len(payload.TestCases) > 0 {
		testCases = payload.TestCases
		log.Printf("Executing code with %d test cases", len(testCases))
	} else if payload.Stdin != "" {
		// Backward compatibility: use single stdin as a test case
		testCases = []string{payload.Stdin}
		stdinSummary := payload.Stdin
		if len(stdinSummary) > 100 {
			stdinSummary = stdinSummary[:100] + "... (truncated)"
		}
		log.Printf("Executing code with single test case: %s", stdinSummary)
	} else {
		testCases = []string{""} // Empty test case if none provided
	}

	result := types.ExecuteCodeResult{
		// Initialize status, it will be overwritten by actual execution outcomes
		Status:  "processing",
		Results: make([]types.TestCaseResult, 0, len(testCases)),
	}

	// Default error in case a language isn't handled, or early exit
	result.Error = "Language not supported or internal error before execution: " + payload.Language

	// Create temp directory for code execution
	tempDir, err := os.MkdirTemp("", "codejudge-"+payload.Language+"-*")
	if err != nil {
		log.Println("Failed to create temp dir:", err)
		utils.SendJSONError(w, "Server error creating execution environment.", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	// Write code to file based on language
	var scriptPath string
	switch payload.Language {
	case "python":
		scriptPath = filepath.Join(tempDir, "script.py")
	case "javascript":
		scriptPath = filepath.Join(tempDir, "script.js")
	case "cpp":
		scriptPath = filepath.Join(tempDir, "script.cpp")
	case "java":
		scriptPath = filepath.Join(tempDir, "Main.java")
	default:
		utils.SendJSONError(w, "Unsupported language: "+payload.Language, http.StatusBadRequest)
		return
	}

	if err := os.WriteFile(scriptPath, []byte(payload.Code), 0644); err != nil {
		log.Println("Failed to write code to temp file:", err)
		utils.SendJSONError(w, "Server error preparing code for execution.", http.StatusInternalServerError)
		return
	}

	// Compile code if needed (for C++ and Java)
	if payload.Language == "cpp" || payload.Language == "java" {
		var compileCmd *exec.Cmd
		var compileOutput bytes.Buffer

		if payload.Language == "cpp" {
			execPath := filepath.Join(tempDir, "executable")
			compileCmd = exec.Command("g++", "-o", execPath, scriptPath)
		} else { // Java
			compileCmd = exec.Command("javac", scriptPath)
		}

		compileCmd.Stdout = &compileOutput
		compileCmd.Stderr = &compileOutput

		if err := compileCmd.Run(); err != nil {
			result.Status = "compilation_error"
			result.Stderr = compileOutput.String()
			result.Error = ""

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(result)
			return
		}
	}

	// Execute each test case
	var totalExecutionTime int64
	maxExecutionTime := int64(0)
	hasError := false

	for _, testInput := range testCases {
		testResult := executeTestCase(payload.Language, scriptPath, tempDir, testInput)

		// Track execution stats
		totalExecutionTime += testResult.ExecutionTimeMs
		if testResult.ExecutionTimeMs > maxExecutionTime {
			maxExecutionTime = testResult.ExecutionTimeMs
		}

		// Track if any test case had an error
		if testResult.Status != "success" {
			hasError = true
		}

		// Add to results array
		result.Results = append(result.Results, testResult)
	}

	// Set overall result based on the first test case for backward compatibility
	if len(result.Results) > 0 {
		result.Stdout = result.Results[0].Stdout
		result.Stderr = result.Results[0].Stderr
		result.Status = result.Results[0].Status
		result.ExecutionTimeMs = maxExecutionTime
		result.Error = ""
	}

	// If all test cases passed, set overall status to success
	if !hasError {
		result.Status = "success"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error encoding execution result: %v", err)
	}
	log.Printf("Code execution: lang=%s, status=%s, time=%dms, test_cases=%d",
		payload.Language, result.Status, maxExecutionTime, len(testCases))
}

// Helper function to execute a single test case
func executeTestCase(language, scriptPath, tempDir, input string) types.TestCaseResult {
	result := types.TestCaseResult{
		Status: "processing",
	}

	var cmd *exec.Cmd
	var timeout time.Duration

	switch language {
	case "python":
		cmd = exec.Command("python3", scriptPath)
		timeout = 2 * time.Second
	case "javascript":
		cmd = exec.Command("node", scriptPath)
		timeout = 10 * time.Second
	case "cpp":
		execPath := filepath.Join(tempDir, "executable")
		cmd = exec.Command(execPath)
		timeout = 2 * time.Second
	case "java":
		cmd = exec.Command("java", "-cp", tempDir, "Main")
		timeout = 5 * time.Second
	default:
		result.Status = "error"
		result.Error = "Unsupported language"
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	runErr := cmd.Run()
	executionTime := time.Since(startTime).Milliseconds()

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.ExecutionTimeMs = executionTime

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "time_limit_exceeded"
		if result.Stderr == "" {
			result.Stderr = fmt.Sprintf("Execution timed out after %d seconds.", int(timeout.Seconds()))
		} else {
			result.Stderr += fmt.Sprintf("\nExecution timed out after %d seconds.", int(timeout.Seconds()))
		}
	} else if runErr != nil {
		result.Status = "runtime_error"
		if result.Stderr == "" {
			result.Stderr = runErr.Error()
		}
	} else {
		result.Status = "success"
	}

	return result
}
