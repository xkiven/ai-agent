# AGENTS.md - AI Agent Development Guide

This document provides guidelines for agentic coding tools operating in this repository.

## Project Overview

- **Language**: Go 1.24.7
- **Framework**: Gin web framework
- **Architecture**: Layered architecture (model -> dao -> service -> route/api)
- **Dependencies**: Redis for session storage, external AI client

## Build & Development Commands

```bash
# Build the application
go build -o ai-agent

# Run the application
go run main.go

# Run with hot reload (if air is installed)
air

# Format code
go fmt ./...

# Vet code
go vet ./...

# Run all tests
go test ./...

# Run a single test file
go test -v ./service/

# Run a specific test
go test -v -run TestFunctionName ./service/

# Get dependencies
go mod tidy

# View dependency graph
go mod graph
```

## Code Style Guidelines

### Imports

Group imports in this order:
1. Standard library (context, errors, fmt, log, time, etc.)
2. Project internal packages (ai-agent/model, ai-agent/dao, etc.)
3. External packages (github.com/gin-gonic/gin, github.com/google/uuid, etc.)

```go
import (
    "ai-agent/dao"
    "ai-agent/internal/aiclient"
    "ai-agent/model"
    "context"
    "errors"
    "fmt"
    "log"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)
```

### Naming Conventions

- **Packages**: Short, lowercase, no underscores (e.g., `dao`, `service`, `model`)
- **Exported functions/types**: PascalCase (e.g., `ChatService`, `NewRedisStore`)
- **Private functions/variables**: camelCase (e.g., `sessionID`, `handleFAQ`)
- **Constants**: PascalCase for exported, camelCase for private (e.g., `IntentFAQ`, `ticketOpen`)
- **Interfaces**: PascalCase, typically with "er" suffix for single-method interfaces (e.g., `Reader`, `Writer`)
- **File names**: lowercase with underscores (e.g., `redis_store.go`, `chat_service.go`)

### Type Definitions

- Use struct types with explicit field names
- Use JSON tags for all API-facing structs
- Define types close to their usage
- Use custom type aliases for constants (e.g., `type IntentType string`)

```go
type IntentType string

const (
    IntentFAQ     IntentType = "faq"
    IntentFlow    IntentType = "flow"
    IntentUnknown IntentType = "unknown"
)

type ChatRequest struct {
    SessionID string     `json:"session_id"`
    Message   string     `json:"message"`
    UserID    string     `json:"user_id"`
    History   []Message  `json:"history,omitempty"`
    Intent    IntentType `json:"intent,omitempty"`
    FlowID    string     `json:"flow_id,omitempty"`
}
```

### Error Handling

- Use `fmt.Errorf` with `%w` for wrapped errors
- Define custom error variables in each package for common errors
- Return errors early (fail fast pattern)
- Log errors at call site with context

```go
var (
    ErrSessionConflict = errors.New("session conflict: current session is newer")
    ErrMaxRetries      = errors.New("max retries exceeded")
    ErrInvalidSession  = errors.New("invalid session")
    ErrInvalidParam    = errors.New("invalid parameter")
)

func (s *RedisStore) Get(ctx context.Context, sessionID string) (*model.Session, error) {
    if sessionID == "" {
        return nil, fmt.Errorf("%w: sessionID is empty", ErrInvalidParam)
    }
    // ...
}
```

### Comments

- Comment exported functions and types with Chinese comments (matches existing codebase)
- Use doc comments for packages
- Keep comments concise but descriptive

```go
// ChatService 聊天服务结构体
type ChatService struct {
    ai            *aiclient.Client
    store         *dao.RedisStore
    decisionLayer *DecisionLayer
}

// NewChatService 创建ChatService实例
func NewChatService(ai *aiclient.Client, store *dao.RedisStore) *ChatService {
    // ...
}
```

### Function Design

- Keep functions focused on single responsibility
- Use dependency injection for external services (e.g., passing clients as parameters)
- Return early for error cases
- Use named return values sparingly

### Layer Responsibilities

- **model/**: Data structures, constants, type definitions
- **dao/**: Data access layer, database/Redis operations
- **service/**: Business logic, orchestration
- **route/**: HTTP route registration
- **api/**: HTTP handlers, request/response parsing
- **internal/**: Private packages, external integrations

### Testing

- Test files should be named `*_test.go`
- Use table-driven tests when testing multiple cases
- Test exported functions primarily
- Use test utilities from testing package

### Configuration

- Hardcoded for now (see main.go)
- AI client URL: `http://127.0.0.1:8000`
- Redis: `localhost:6379`
- Server port: `:8080`

### Logging

- Use standard `log` package
- Include session/user context in log messages
- Log important state transitions

```go
log.Printf("[Session %s] 状态: %s, FlowID: %s", session.ID, session.State, session.FlowID)
```
