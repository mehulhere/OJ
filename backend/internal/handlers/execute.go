package handlers

import (
	"backend/internal/types"
	"backend/internal/utils"
	"context"
	"encoding/json"
	"log"
	"net/http"
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

	// Execute each test case using the external executor service
	const defaultTimeLimitMs = 5000
	var maxExecutionTime int64
	hasError := false

	for _, testInput := range testCases {
		// Prepare execution request
		execReq := types.ExecutionRequest{
			Language:    payload.Language,
			Code:        payload.Code,
			Input:       testInput,
			TimeLimitMs: defaultTimeLimitMs,
		}

		// Give a little buffer over the requested time limit
		execCtx, cancel := context.WithTimeout(r.Context(), time.Duration(defaultTimeLimitMs+2000)*time.Millisecond)
		execResult, err := utils.ExecuteCode(execCtx, execReq)
		cancel()

		// Map executor result to API response structure
		testCaseRes := types.TestCaseResult{
			Stdout:          "",
			Stderr:          "",
			ExecutionTimeMs: int64(execResult.ExecutionTimeMs),
			Status:          execResult.Status,
		}

		if execResult.Status == "success" {
			testCaseRes.Stdout = execResult.Output
		} else {
			testCaseRes.Stderr = execResult.Output
		}

		if err != nil {
			testCaseRes.Error = err.Error()
		}

		// Track execution stats
		if testCaseRes.ExecutionTimeMs > maxExecutionTime {
			maxExecutionTime = testCaseRes.ExecutionTimeMs
		}

		// Track if any test case had an error
		if testCaseRes.Status != "success" {
			hasError = true
		}

		// Add to results array
		result.Results = append(result.Results, testCaseRes)
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
