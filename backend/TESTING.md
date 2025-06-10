# OJ Backend Testing Guide

This document provides information about the testing approach for the Online Judge backend system.

## Test Environment Setup

To run the tests, you need:

1. A local MongoDB instance running on localhost:27017
2. Environment variables set in a `.env` file in the `backend` directory:
   ```
   MONGO_URI=mongodb://localhost:27017
   JWT_SECRET_KEY=your_secret_key
   PORT=8080
   ```

## Running Tests

To run all tests:

```bash
cd backend
go test ./...
```

To run specific tests:

```bash
# Run all submission-related tests
go test -v ./internal/handlers -run "TestSubmit|TestSubmissionQueue|TestGetSubmission"

# Run a specific test
go test -v ./internal/handlers -run TestSubmitSolutionHandler_FileStorage
```

## Test Coverage

The following key components are covered by tests:

### Submission System Tests

The file-based submission system is tested with the following test cases:

1. **TestSubmitSolutionHandler_FileStorage**: Tests that the submission handler:
   - Properly stores code in a file instead of the database
   - Creates the necessary directory structure for submissions
   - Queues the submission for processing
   - Returns appropriate response codes and submission ID

2. **TestSubmissionQueueProcessing**: Tests that the submission queue:
   - Properly processes queued submissions
   - Updates the submission status in the database
   - Tracks test case results
   - Creates and populates the test case status file

3. **TestGetSubmissionDetailsHandler**: Tests that the submission details endpoint:
   - Properly retrieves submission details from the database
   - Includes code from the file system
   - Includes test case status from the status file
   - Enforces access controls (only the submitter or admin can view submissions)

## File-Based Submission Structure

For each submission, the system creates:

```
./submissions/<submission_id>/
  ├── code.<extension>         # The submitted code file
  ├── output_1.txt             # Output from test case 1
  ├── output_2.txt             # Output from test case 2
  └── testcasesStatus.txt      # Detailed results of each test case
```

This approach keeps the database lean by storing code and outputs in the file system rather than in the database.

## Adding New Tests

When adding new features to the submission system, follow these patterns to create tests:

1. Use helper functions like `createTestProblem()` and `cleanupTestData()` to manage test data
2. Create proper test assertions to verify file system operations
3. Ensure test cleanup to avoid test data accumulation

## Testing the Queue System

The submission queue is designed to process submissions asynchronously. The tests use direct function calls to `processSubmission()` for deterministic testing, but in production, the queue is processed by a background goroutine. 
 