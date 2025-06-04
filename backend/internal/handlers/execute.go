package handlers

import (
	"backend/internal/types"
	"backend/internal/utils"
	"context"
	"encoding/json"
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

	result := types.ExecuteCodeResult{
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
			// Use utils.SendJSONError for server-internal issues before forming a full 'result'
			utils.SendJSONError(w, "Server error creating python execution environment.", http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(tempDir)

		scriptPath := filepath.Join(tempDir, "script.py")
		if err := os.WriteFile(scriptPath, []byte(payload.Code), 0644); err != nil {
			log.Println("Failed to write python code to temp file:", err)
			utils.SendJSONError(w, "Server error preparing python code for execution.", http.StatusInternalServerError)
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
			utils.SendJSONError(w, "Server error creating javascript execution environment.", http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(tempDir)

		scriptPath := filepath.Join(tempDir, "script.js")
		if err := os.WriteFile(scriptPath, []byte(payload.Code), 0644); err != nil {
			log.Println("Failed to write javascript code to temp file:", err)
			utils.SendJSONError(w, "Server error preparing javascript code for execution.", http.StatusInternalServerError)
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
	// are handled by 'utils.SendJSONError' which sends appropriate non-200 statuses.
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error encoding execution result: %v", err)
		// At this point, headers are sent. Can't send a new JSON error.
	}
	log.Printf("Code execution: lang=%s, status=%s, time=%dms, stdout_len=%d, stderr_len=%d, error_msg_len=%d",
		payload.Language, result.Status, result.ExecutionTimeMs, len(result.Stdout), len(result.Stderr), len(result.Error))
}
