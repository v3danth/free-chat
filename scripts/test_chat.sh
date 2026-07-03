 
curl -X POST http://localhost:8080/auth/login \\n  -H "Content-Type: application/json" \\n  -d '{"email":"john@example.com","password":"secret123"}'
npx wscat -c "ws://localhost:8080/ws?token="
curl -s -X POST "$BASE_URL/auth/guest" \\n  -H "Content-Type: application/json" \\n  -d '{\n    "username": "guest_user_3",\n    "gender": "male",\n    "age": 22,\n    "about": "I am a guest"\n  }'
BASE_URL="http://localhost:8080"
curl -s -X POST "$BASE_URL/auth/guest" \\n  -H "Content-Type: application/json" \\n  -d '{\n    "username": "guest_user_3",\n    "gender": "male",\n    "age": 22,\n    "about": "I am a guest"\n  }'
curl -s -X POST "$BASE_URL/auth/guest" \\n  -H "Content-Type: application/json" \\n  -d '{\n    "username": "guest_user_4",\n    "gender": "male",\n    "age": 22,\n    "about": "I am a guest"\n  }'
