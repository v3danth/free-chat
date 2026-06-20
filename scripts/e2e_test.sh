#!/bin/bash

set -e  # stop on first error

BASE_URL="http://localhost:8080"

echo "Starting E2E tests..."

echo ""
echo "1. Creating guest user..."
GUEST_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/guest" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "guest_user_1",
    "gender": "male",
    "age": 22,
    "about": "I am a guest"
  }')

echo "$GUEST_RESPONSE"
echo ""

echo "2. Registering user..."
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "email": "john@example.com",
    "password": "secret123",
    "gender": "male",
    "age": 25,
    "about": "I am registered"
  }')

echo "$REGISTER_RESPONSE"
echo ""

echo "3. Logging in user..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "secret123"
  }')

echo "$LOGIN_RESPONSE"
echo ""

echo "E2E tests completed successfully!"
