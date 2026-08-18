#!/bin/bash

# Script to run the JWT HTTP example

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "========================================"
echo "  JWT HTTP Example"
echo "========================================"
echo ""
echo "Starting HTTP server on port 8080..."
echo ""
echo "Available endpoints:"
echo "  POST   http://localhost:8080/api/login      - Login and get tokens"
echo "  POST   http://localhost:8080/api/refresh    - Refresh access token"
echo "  GET    http://localhost:8080/api/health     - Health check (public)"
echo "  GET    http://localhost:8080/api/profile    - Get user profile (protected)"
echo "  GET    http://localhost:8080/api/users      - List users (protected)"
echo "  GET    http://localhost:8080/api/admin/users - Admin endpoint (admin only)"
echo "  GET    http://localhost:8080/api/content    - Optional auth endpoint"
echo ""
echo "Test users:"
echo "  Email: admin@example.com, Password: admin123 (role: admin)"
echo "  Email: user@example.com,  Password: user123  (role: user)"
echo ""
echo "Example curl commands:"
echo ""
echo "# Login:"
echo "curl -X POST http://localhost:8080/api/login \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"email\":\"admin@example.com\",\"password\":\"admin123\"}'"
echo ""
echo "# Access protected endpoint (replace <TOKEN> with access_token from login):"
echo "curl http://localhost:8080/api/profile \\"
echo "  -H 'Authorization: Bearer <TOKEN>'"
echo ""
echo "========================================"
echo ""

cd "$PROJECT_ROOT" || exit 1
go run pkg/jwt/examples/http/main.go
