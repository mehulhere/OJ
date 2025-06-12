package main

import (
	"bytes"
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

type ExecRequest struct {
	Code          string `json:"code"`
	Input         string `json:"input"`
	TimeLimitMs   int    `json:"time_limit_ms"`
	MemoryLimitKB int    `json:"memory_limit_kb"`
	Language      string `json:"language"`
}

type ExecResult struct {
	Output          string `json:"output"`
	ExecutionTimeMs int    `json:"execution_time_ms"`
	MemoryUsedKB    int    `json:"memory_used_kb"`
	Status          string `json:"status"`
}

func main() {
	http.HandleFunc("/execute", execHandler)
	log.Println("🔵 C++-executor listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func execHandler(w http.ResponseWriter, r *http.Request) {
	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(req.TimeLimitMs)*time.Millisecond)
	defer cancel()

	start := time.Now()

	out, status := runCode(ctx, req.Code, req.Input)
	execMs := int(time.Since(start).Milliseconds())

	res := ExecResult{
		Output:          out,
		Status:          status,
		ExecutionTimeMs: execMs,
		MemoryUsedKB:    0,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func runCode(ctx context.Context, code, input string) (output, status string) {
	log.Println("Running code...")
	dir, _ := os.MkdirTemp("", "exec-*")
	defer os.RemoveAll(dir)

	source := filepath.Join(dir, "source.cpp")
	exe := filepath.Join(dir, "main")
	_ = os.WriteFile(source, []byte(code), 0644)

	// Compile
	compileCmd := exec.Command("g++", "-o", exe, source)
	var compileOut bytes.Buffer
	compileCmd.Stderr = &compileOut
	if err := compileCmd.Run(); err != nil {
		return compileOut.String(), "compilation_error"
	}

	// Run
	cmd := exec.CommandContext(ctx, exe)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "time limit exceeded", "time_limit_exceeded"
		}
		if stderr.Len() > 0 {
			return stderr.String(), "runtime_error"
		}
		return err.Error(), "runtime_error"
	}

	if stderr.Len() > 0 {
		return stderr.String(), "runtime_error"
	}

	return stdout.String(), "success"
}
