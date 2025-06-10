#!/bin/bash

# Check if the correct number of arguments is provided
if [ "$#" -ne 4 ]; then
    echo "Usage: $0 <firstname> <lastname> <username> <password>"
    exit 1
fi

# Navigate to the backend directory
cd backend

# Run the admin creation script
go run cmd/admin/create_admin.go "$1" "$2" "$3" "$4"

# Return to the original directory
cd -

echo "Admin user creation process completed." 