curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"john@example.com","password":"secret123"}'

npx wscat -c "ws://localhost:8080/ws?token=YOUR_JWT"
