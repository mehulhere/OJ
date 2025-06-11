# OJ (Online Judge)

This is an Online Judge platform for competitive programming practice and contests. The platform supports multiple programming languages, real-time code execution, and automated evaluation of submissions.

## Features

- **User Authentication**: Register, login, and manage your account
- **Problem Solving**: Browse problems by difficulty and topic
- **Code Editor**: Built-in Monaco editor with syntax highlighting
- **Multiple Languages**: Support for Python, JavaScript, C++, and Java
- **Real-time Execution**: Test your code with custom inputs before submission
- **Automated Evaluation**: Submit solutions to be evaluated against test cases
- **Detailed Feedback**: Receive specific error messages and test case results
- **Submission History**: Track your progress and review past submissions

## Getting Started

### Prerequisites

- Go 1.24+ for backend
- Node.js 16+ for frontend
- MongoDB instance

### Setup

1. Clone the repository
2. Set up environment variables in `backend/.env`:
   ```
   MONGO_URI=mongodb://localhost:27017
   JWT_SECRET_KEY=your_secret_key
   PORT=8080
   ```
3. Start the backend:
   ```bash
   cd backend
   go run cmd/server/main.go
   ```
4. Start the frontend:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

## Creating an Admin User

To create an admin user, run the following command from the root directory of the project:

```bash
./create_admin.sh <firstname> <lastname> <username> <password>
```
Replace `<firstname>`, `<lastname>`, `<username>`, and `<password>` with the desired details for your admin user.

## Error Classification

The platform intelligently classifies errors to provide helpful feedback:

- **Compilation Errors**: Syntax errors, name errors, and other code structure issues
- **Runtime Errors**: Errors that occur during execution (e.g., division by zero)
- **Time Limit Exceeded**: Solutions that take too long to execute
- **Memory Limit Exceeded**: Solutions that use too much memory
- **Wrong Answer**: Solutions that produce incorrect output for test cases

## Architecture

- **Backend**: Go REST API with JWT authentication and MongoDB integration
- **Frontend**: Next.js application with React, TypeScript, and Tailwind CSS
- **Code Execution**: Sandboxed environment for safe code execution

## Contribution

Contributions are welcome! Please feel free to submit a Pull Request.