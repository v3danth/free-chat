# Free Chat Backend Project Archive

## Project Overview

Building a chat website backend in Go.

Requirements:

* Single backend server
* WebSockets for realtime messaging
* MySQL database
* Guest users and registered users
* JWT authentication
* Clean architecture
* Production-style code organization
* Scalability-minded design
* Google-style engineering standards
* Database reset workflow for schema changes during development

---

# User Requirements

## User Types

### Guest User

Required fields:

* username
* gender
* age
* about

No:

* email
* password

### Registered User

Required fields:

* username
* gender
* age
* about
* email
* password

---

# User Model

Location:

internal/user/model.go

```go
type Type string

const (
    TypeGuest      Type = "guest"
    TypeRegistered Type = "registered"
)

type Gender string

const (
    GenderMale      Gender = "male"
    GenderFemale    Gender = "female"
    GenderNonBinary Gender = "non-binary"
    GenderFemboy    Gender = "femboy"
    GenderOther     Gender = "other"
    GenderCouple    Gender = "couple"
)

type User struct {
    ID uint64

    UserType Type

    Username string

    Gender Gender

    Age uint8

    About string

    Email *string

    PasswordHash *string

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

# Project Structure

```text
cmd/
└── server/
    └── main.go

internal/
├── auth/
│   ├── service.go
│   ├── handler.go
│   ├── jwt.go
│   └── service_test.go
│
├── config/
│   └── config.go
│
├── database/
│   └── mysql.go
│
├── user/
│   ├── model.go
│   ├── repository.go
│   └── repository_test.go
│
└── websocket/
    ├── client.go
    ├── handler.go
    ├── hub.go
    ├── hub_room.go
    ├── room.go
    └── message.go

migrations/
└── 001_init.sql

scripts/
└── reset_db.sql

Makefile
.env
.env.example
```

---

# Repository Pattern

## Purpose

Repository handles database access.

Service handles business logic.

Handler handles HTTP/WebSocket requests.

---

## User Repository Interface

Location:

internal/user/repository.go

```go
type Repository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id uint64) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    GetByUsername(ctx context.Context, username string) (*User, error)
}
```

---

## MySQL Repository

```go
type MySQLRepository struct {
    db *sql.DB
}
```

Constructor:

```go
func NewRepository(db *sql.DB) *MySQLRepository
```

Implemented:

* Create
* GetByID
* GetByEmail
* GetByUsername

---

# Authentication

## JWT

Added:

internal/auth/jwt.go

### JWTManager

Responsible for:

* Generate(userID)
* Validate(token)

JWT contains:

```go
user_id
exp
iat
```

---

## Auth Service

Handles:

* RegisterGuest
* RegisterUser
* Login

Uses:

* bcrypt
* user repository
* JWTManager

---

# Configuration

## Production Config

```go
func Load() *Config
```

Reads:

* APP_ENV
* SERVER_PORT
* MYSQL_HOST
* MYSQL_PORT
* MYSQL_USER
* MYSQL_PASSWORD
* MYSQL_DATABASE
* JWT_SECRET

---

## Testing Config

Added:

```go
func LoadTestConfig() *Config
```

Used by tests.

Avoids requiring .env.

---

# Database

## MySQL

Database:

chat_db

Test database:

chat_db_test

---

# Migration

File:

migrations/001_init.sql

Final schema:

```sql
CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

    user_type ENUM('guest', 'registered') NOT NULL,

    username VARCHAR(32) NOT NULL UNIQUE,

    gender ENUM(
        'male',
        'female',
        'non-binary',
        'femboy',
        'other',
        'couple'
    ) NOT NULL,

    age TINYINT UNSIGNED NOT NULL,

    about VARCHAR(50) NOT NULL DEFAULT '',

    email VARCHAR(255) NULL UNIQUE,

    password_hash VARCHAR(255) NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_user_type (user_type),
    INDEX idx_email (email),
    INDEX idx_username (username)
);
```

Important fix:

Removed trailing comma from ENUM.

---

# Reset Database Script

Original issue:

MySQL SOURCE command cannot be used inside redirected SQL files.

Old:

```sql
DROP TABLE IF EXISTS users;

SOURCE migrations/001_init.sql;
```

Must instead execute migrations separately from Makefile.

---

# Makefile

Final idea:

```make
run:
	go run ./cmd/server

migrate:
	mysql -u $(MYSQL_USER) -p$(MYSQL_PASSWORD) $(MYSQL_DATABASE) < migrations/001_init.sql

db-reset:
	mysql -u $(MYSQL_USER) -p$(MYSQL_PASSWORD) $(MYSQL_DATABASE) -e "DROP TABLE IF EXISTS users;"
	mysql -u $(MYSQL_USER) -p$(MYSQL_PASSWORD) $(MYSQL_DATABASE) < migrations/001_init.sql

test:
	go test ./...

lint:
	golangci-lint run
```

Recommendation:

Do not commit passwords.

Use environment variables.

---

# Testing

Added:

## Auth Tests

service_test.go

Tests:

* guest creation

---

## User Repository Tests

repository_test.go

Tests:

* create user
* get by id

Required:

chat_db_test database

Migration must run before tests.

---

# WebSocket Architecture

Initial design:

Global broadcast.

Refactored to rooms.

---

# Final WebSocket Design

## Client

Location:

internal/websocket/client.go

```go
type Client struct {
    hub *Hub

    conn *websocket.Conn

    send chan []byte

    userID uint64

    rooms map[uint64]bool
}
```

---

## Room

Location:

internal/websocket/room.go

```go
type Room struct {
    ID uint64

    Clients map[*Client]bool
}
```

---

## Hub

Location:

internal/websocket/hub.go

```go
type Hub struct {
    users map[uint64]*Client

    rooms map[uint64]*Room

    register chan *Client

    unregister chan *Client
}
```

---

## Hub Lifecycle

Register:

```go
h.users[client.userID] = client
```

Unregister:

```go
delete(h.users, client.userID)

for roomID := range client.rooms {
    h.LeaveRoom(roomID, client)
}

close(client.send)
```

---

# Room Management

Location:

internal/websocket/hub_room.go

## JoinRoom

Creates room if missing.

Adds client to room.

Adds room to client membership.

---

## LeaveRoom

Removes client from room.

Removes room membership from client.

---

## BroadcastToRoom

```go
func (h *Hub) BroadcastToRoom(
    roomID uint64,
    payload []byte,
)
```

Current recommendation:

```go
default:
    // skip slow client
```

Do NOT disconnect clients yet.

Need proper Disconnect() lifecycle first.

---

# WebSocket Message Model

Location:

internal/websocket/message.go

```go
type MessageType string

const (
    MessageTypeChat  MessageType = "chat"
    MessageTypeJoin  MessageType = "join"
    MessageTypeLeave MessageType = "leave"
)
```

Message:

```go
type Message struct {
    Type MessageType `json:"type"`

    RoomID uint64 `json:"room_id,omitempty"`

    Content string `json:"content,omitempty"`
}
```

---

# WebSocket Authentication

Endpoint:

```text
/ws?token=JWT
```

Flow:

1. Extract token
2. Validate JWT
3. Upgrade websocket
4. Create Client
5. Register client
6. Start Read/Write goroutines

---

# Client Read Loop

Current architecture:

```go
switch msg.Type {

case MessageTypeJoin:
    c.hub.JoinRoom(msg.RoomID, c)

case MessageTypeLeave:
    c.hub.LeaveRoom(msg.RoomID, c)

case MessageTypeChat:
    c.hub.BroadcastToRoom(
        msg.RoomID,
        raw,
    )
}
```

---

# Verified Working

Confirmed:

* Login works
* JWT works
* WebSocket auth works
* Room joins work
* Room broadcasts work
* Two users can exchange messages

Logs observed:

```text
user 2 connected
user 4 connected

user=4 joined room=1 room_size=1
user=2 joined room=1 room_size=2

broadcast room=1 clients=2
```

Behavior is correct.

---

# Recommendations Given

## Keep Sender Receiving Messages

Current behavior:

Sender receives their own chat message.

Recommendation:

Keep it.

Reasons:

* Simpler architecture
* Common in chat systems
* Easier frontend implementation

---

## Add Sender Metadata

Next improvement:

```go
type ChatMessage struct {
    Type MessageType `json:"type"`

    RoomID uint64 `json:"room_id"`

    SenderID uint64 `json:"sender_id"`

    Content string `json:"content"`

    Timestamp time.Time `json:"timestamp"`
}
```

So clients know:

* who sent message
* when it was sent

---

# Unresolved Work

## Immediate Next Step

Persist chat in MySQL.

Add tables:

```sql
rooms
room_members
messages
```

---

## Future Work

### Message Persistence

Store messages.

Load room history.

Survive server restarts.

---

### Room Persistence

Store room membership.

---

### Direct Messages

Support:

user -> user

---

### Multi-Connection Users

Current limitation:

```go
users map[uint64]*Client
```

Only one connection per user.

Future:

```go
users map[uint64]map[*Client]bool
```

or similar.

---

### Disconnect Lifecycle

Create:

```go
func (h *Hub) Disconnect(client *Client)
```

Responsible for:

* removing user
* leaving rooms
* closing channel
* closing websocket

Avoid duplicate cleanup logic.

---

### WebSocket Reliability

Future:

* ping/pong
* heartbeat
* idle timeout
* reconnect handling

---

# Context to Give the Next ChatGPT

We are building a Go chat backend.

Current status:

* MySQL users table implemented
* JWT authentication implemented
* Guest and registered users implemented
* Repository pattern implemented
* Unit tests implemented
* WebSocket authentication implemented
* Room-based chat implemented
* Two clients can join a room and exchange messages successfully

Current WebSocket architecture:

* Hub
* Client
* Room
* Message

Room messages are currently in-memory only.

The next major task is designing and implementing persistent chat storage:

* rooms table
* room_members table
* messages table

Then loading room history when users join rooms.

Maintain clean architecture and production-grade Go code standards.
