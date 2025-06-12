package utils

import (
	"backend/internal/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

// ExecuteCode delegates code execution to a long-lived language-specific
// executor container running on localhost.
//
// Language  →  Port
//
//	python      :8001
//	javascript  :8002
//	cpp         :8003
//	java        :8004
//
// The container exposes POST /execute with body types.ExecutionRequest
// and returns types.ExecutionResult.
func ExecuteCode(ctx context.Context, req types.ExecutionRequest) (types.ExecutionResult, error) {
	fmt.Println("Executing code in DOCKER in ", req.Language)
	langToPort := map[string]string{
		"python":     "8001",
		"javascript": "8002",
		"js":         "8002",
		"cpp":        "8003",
		"c++":        "8003",
		"java":       "8004",
	}

	port, ok := langToPort[strings.ToLower(req.Language)]
	if !ok {
		return types.ExecutionResult{}, fmt.Errorf("unsupported language: %s", req.Language)
	}

	endpoint := fmt.Sprintf("http://localhost:%s/execute", port)

	payload, err := json.Marshal(req)
	if err != nil {
		return types.ExecutionResult{}, fmt.Errorf("marshal exec-request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return types.ExecutionResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return types.ExecutionResult{}, fmt.Errorf("executor call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var e map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return types.ExecutionResult{}, fmt.Errorf("executor %d: %s", resp.StatusCode, e["message"])
	}

	var result types.ExecutionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return types.ExecutionResult{}, fmt.Errorf("decode exec-result: %w", err)
	}

	return result, nil
}
