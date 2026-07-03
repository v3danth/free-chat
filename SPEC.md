# FreeChat HTTP & WebSocket API Specification

**Base URL:** `http://localhost:8080`
**Protocol:** HTTP/1.1, WebSocket (RFC 6455)
**Content-Type:** `application/json` (unless otherwise specified)

> **Note on Error Responses:** The API currently utilizes Go's standard `http.Error` for failure states. Therefore, error responses return a `text/plain` body instead of JSON. Successful payloads return `application/json`.

---

## 1. Authentication API

### 1.1 Create Guest User
Allows a user to join without a password. Generates a temporary user and returns a JWT.

*   **Endpoint:** `POST /auth/guest`
*   **Auth Required:** No
*   **Content-Type:** `application/json`

**Request Body:**
```json
{
  "username": "string (2-32 chars, alphanumeric/underscore)",
  "gender": "male | female | non-binary | femboy | other | couple",
  "age": 0,
  "about": "string (optional)"
}
```

**Responses:**

*   `201 Created` (Success)
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "ID": 1,
    "UserType": "guest",
    "Status": "active",
    "Username": "coolguest",
    "Gender": "male",
    "Age": 25,
    "About": "hello",
    "Email": null,
    "PasswordHash": null,
    "LastSeenAt": null,
    "CreatedAt": "2023-10-27T10:00:00Z",
    "UpdatedAt": "2023-10-27T10:00:00Z"
  }
}
```
*   `409 Conflict` (Username already taken)
*   `400 Bad Request` (Invalid request body / missing username)

---

### 1.2 Register User
Creates a permanent user account. Does **not** return a token (must login afterward).

*   **Endpoint:** `POST /auth/register`
*   **Auth Required:** No
*   **Content-Type:** `application/json`

**Request Body:**
```json
{
  "username": "string (2-32 chars)",
  "email": "string (valid email)",
  "password": "string (min 6 chars)",
  "gender": "male",
  "age": 25,
  "about": "string"
}
```

**Responses:**

*   `201 Created` (Success - Returns User object, no token)
```json
{
  "ID": 2,
  "UserType": "registered",
  "Status": "active",
  "Username": "newuser",
  "Email": "user@example.com",
  "PasswordHash": null,
  "CreatedAt": "2023-10-27T10:00:00Z",
  "UpdatedAt": "2023-10-27T10:00:00Z"
}
```
*   `409 Conflict` (Username or email already registered)
*   `400 Bad Request` (Missing fields)

---

### 1.3 Login User
Authenticates a registered user and returns a JWT.

*   **Endpoint:** `POST /auth/login`
*   **Auth Required:** No
*   **Content-Type:** `application/json`

**Request Body:**
```json
{
  "email": "string",
  "password": "string"
}
```

**Responses:**

*   `200 OK` (Success)
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": { ... }
}
```
*   `401 Unauthorized` (Invalid credentials)

---

## 2. WebSocket API

### 2.1 Establish Connection
Upgrades an HTTP request to a persistent WebSocket connection.

*   **Endpoint:** `GET /ws`
*   **Auth Required:** Optional (See query params)

**Query Parameters:**
*   `token` (string): Valid JWT from Auth APIs.
*   `username` (string): If no token is provided, the backend will instantly create a guest account with this username and issue a token.

**Headers:**
*   `Connection: Upgrade`
*   `Upgrade: websocket`
*   `Sec-WebSocket-Key: <client_generated_key>`
*   `Sec-WebSocket-Version: 13`

**Responses:**

*   `101 Switching Protocols` (Success)
*   `401 Unauthorized` (Invalid token provided)
*   `409 Conflict` (Username already taken during direct WS guest creation)

---

### 2.2 Client -> Server Messages
All messages sent over the WS connection must be JSON formatted.

**Join Room:**
```json
{
  "type": "join",
  "room_id": 1
}
```

**Leave Room:**
```json
{
  "type": "leave",
  "room_id": 1
}
```

**Send Text Chat:**
```json
{
  "type": "chat",
  "room_id": 1,
  "content": "Hello world!"
}
```

**Send Media Chat:**
*(Note: Media must be uploaded via HTTP first to get a `media_id`)*
```json
{
  "type": "media",
  "room_id": 1,
  "media_id": 45
}
```

---

### 2.3 Server -> Client Messages

**Auth Token (Direct Guest Creation):**
Sent immediately after connection if the client connected using `?username=...` instead of a token.
```json
{
  "type": "auth",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Rate Limit Info:**
Sent upon initial connection to inform the client of their limits.
```json
{
  "type": "rate_limit",
  "remaining": 30,
  "reset_in_seconds": 60
}
```

**Room History:**
Sent immediately after a client sends a `join` message. Contains the last 50 messages.
```json
{
  "type": "history",
  "room_id": 1,
  "messages": [
    {
      "type": "history",
      "room_id": 1,
      "message_id": 10,
      "sender_id": 1,
      "username": "coolguest",
      "content": "Hello!",
      "media_url": null,
      "media_type": null,
      "timestamp": 1698765432
    }
  ]
}
```

**Incoming Chat Message:**
Broadcast to all users in the room when a message is sent.
```json
{
  "type": "chat",
  "room_id": 1,
  "message_id": 11,
  "sender_id": 2,
  "username": "user2",
  "content": "Hi there!",
  "media_url": "/media/images/abc123.jpg",
  "media_type": "image",
  "timestamp": 1698765500,
  "rate_limited": false,
  "filtered": false
}
```
*Note: If `filtered` is `true`, the `content` string has been mutated by the server's profanity filter (e.g., `f***`).*

**Error Message:**
Sent when an action fails (e.g., sending an empty message, media not found, rate limit hit).
```json
{
  "type": "error",
  "error": "rate limit exceeded, slow down",
  "code": "RATE_LIMITED"
}
```

---

## 3. Media Upload API

All media endpoints require authentication via the `Authorization` header.

**Global Auth Header:**
```
Authorization: Bearer <jwt_token>
```

### 3.1 Upload Image
*   **Endpoint:** `POST /media/upload/image`
*   **Content-Type:** `multipart/form-data`
*   **Max File Size:** 10MB
*   **Allowed MIME:** `image/jpeg`, `image/png`, `image/webp`

**Form Data:**
*   `image` (file): The image binary.

**Response:** `201 Created`
```json
{
  "id": 1,
  "url": "/media/images/64bit_hash.jpg",
  "type": "image",
  "file_size": 1048576
}
```
*   `413 Request Entity Too Large` (File too big)
*   `415 Unsupported Media Type` (Invalid MIME type)

---

### 3.2 Upload GIF
*   **Endpoint:** `POST /media/upload/gif`
*   **Content-Type:** `multipart/form-data`
*   **Max File Size:** 15MB
*   **Allowed MIME:** `image/gif`

**Form Data:**
*   `gif` (file): The gif binary.

**Response:** `201 Created` (Same structure as 3.1, but `"type": "gif"`)

---

### 3.3 Upload Voice Note
*   **Endpoint:** `POST /media/upload/voice`
*   **Content-Type:** `multipart/form-data`
*   **Max File Size:** 5MB
*   **Allowed MIME:** `audio/webm`, `audio/ogg`, `audio/mp4`, `audio/mpeg`

**Form Data:**
*   `voice` (file): The audio binary.

**Response:** `201 Created` (Same structure as 3.1, but `"type": "voice"`)

---

## 4. Media Moderation API

### 4.1 Flag Media
Reports a piece of media. If a piece of media receives `N` flags (configured via `MEDIA_FLAG_THRESHOLD`, default 3), it is automatically marked as flagged and can no longer be shared in chat.

*   **Endpoint:** `POST /media/flag`
*   **Auth Required:** Yes
*   **Content-Type:** `application/json`

**Request Body:**
```json
{
  "media_id": 1,
  "reason": "inappropriate" 
}
```
*Valid `reason` enum:* `inappropriate`, `spam`, `nudity`, `violence`, `copyright`, `other`

**Responses:**

*   `204 No Content` (Success - No body returned)
*   `400 Bad Request` (Invalid reason or missing fields)
*   `404 Not Found` (Media does not exist)
*   `409 Conflict` (User has already flagged this specific media)

---

## 5. Static Media Serving

### 5.1 Access Uploaded Media
Serves the actual files uploaded via the Media API.

*   **Endpoint:** `GET /media/files/{path}`
*   **Auth Required:** No (Publicly accessible via URL)
*   **Path Variables:** `path` is the relative path returned in the upload response (e.g., `images/64bit_hash.jpg`).

**Responses:**
*   `200 OK` (Returns binary file with appropriate `Content-Type` header)
*   `400 Bad Request` (If path traversal is detected e.g., `../`)

