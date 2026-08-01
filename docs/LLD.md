# Low-Level Design (LLD) — Notification Service

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Project Structure](#2-project-structure)
3. [Application Startup Flow](#3-application-startup-flow)
4. [Request Lifecycle (Request → Response)](#4-request-lifecycle)
5. [Layer-by-Layer Breakdown](#5-layer-by-layer-breakdown)
   - 5.1 Config Layer
   - 5.2 Logger Layer
   - 5.3 Database Layer
   - 5.4 Middleware Layer
   - 5.5 Handler Layer
   - 5.6 Service Layer
   - 5.7 Store (Repository) Layer
   - 5.8 Domain Layer
   - 5.9 Validation Layer
   - 5.10 Common/Response Layer
   - 5.11 Error Handling Layer
6. [API Endpoints & Flow Diagrams](#6-api-endpoints--flow-diagrams)
   - 6.1 POST /api/v1/templates — Create Template
   - 6.2 PUT /api/v1/templates/:id — Update Template
   - 6.3 GET /api/v1/templates/:id — Get Active Template
   - 6.4 GET /api/v1/templates/:id/versions/:version — Get Specific Version
   - 6.5 GET /api/v1/templates — List Templates (Cursor Pagination)
   - 6.6 DELETE /api/v1/templates/:id — Deactivate Template
7. [Error Handling Flow](#7-error-handling-flow)
8. [Database Schema & Migrations](#8-database-schema--migrations)
9. [Graceful Shutdown Flow](#9-graceful-shutdown-flow)

---

## 1. Architecture Overview

The Notification System follows a **clean layered architecture** with strict separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                        HTTP Client                          │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP Request
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                     Echo HTTP Server                         │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Middleware Pipeline                        │  │
│  │  1. RequestIDMiddleware()                               │  │
│  │  2. RequestLoggerMiddleware()                           │  │
│  │  3. HTTPErrorHandler (global, not per-route)            │  │
│  └──────────────────────┬────────────────────────────────┘  │
│                          │                                   │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │              Handler (template.Handler)                │  │
│  │  - Parse request params / body                         │  │
│  │  - Validate via middleware.ValidateAndBind             │  │
│  │  - Call Service layer                                  │  │
│  │  - Map errors → HTTP responses                         │  │
│  │  - Return JSON response                                │  │
│  └──────────────────────┬────────────────────────────────┘  │
│                          │                                   │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │              Service (template.Service)                  │  │
│  │  - Business logic                                       │  │
│  │  - UUID generation, versioning, variable extraction     │  │
│  │  - Call Store interface                                  │  │
│  └──────────────────────┬────────────────────────────────┘  │
│                          │                                   │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │        Store (template.PGStore — implements Store)     │  │
│  │  - GORM-based PostgreSQL queries                      │  │
│  │  - Transactional version updates                      │  │
│  │  - Domain error mapping (gorm.ErrRecordNotFound →     │  │
│  │    domain.ErrTemplateNotFound)                        │  │
│  └──────────────────────┬────────────────────────────────┘  │
│                          │                                   │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │              PostgreSQL Database                        │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Module Dependency Graph

```
services/template-svc/cmd/server/main.go
        │
        ├──► pkg/config          (env loading)
        ├──► pkg/logger          (zap logger)
        ├──► pkg/database        (GORM + PostgreSQL)
        ├──► internal/server     (Echo setup + middleware)
        │        ├──► pkg/middleware/request_id.go
        │        ├──► pkg/middleware/request.go
        │        └──► pkg/middleware/errors.go
        └──► internal/template
                 ├── handler.go  (routes + HTTP handlers)
                 ├── service.go  (business logic)
                 ├── store.go    (Store interface)
                 ├── store_pg.go (PGStore implementation)
                 ├── dto.go      (request/response DTOs)
                 └── types.go    (local Template type)
```

### Workspace (go.work)

```
go 1.26.3

use (
    ./pkg                              → github.com/bikky-kc013/notification-system/pkg
    ./services/template-svc             → github.com/bikky-kc013/notification-system/services/template-svc
)
```

Two Go modules:
- `pkg/` — shared library (config, logger, database, middleware, common, validation, pagination, domain types)
- `services/template-svc/` — the actual service binary

---

## 2. Project Structure

```
notification_service/
├── .env                              # Environment variables
├── go.work                           # Go workspace definition
├── go.work.sum
├── Makefile                          # Build/test/lint automation
├── README.md
│
├── pkg/                              # Shared library module
│   ├── go.mod
│   ├── go.sum
│   ├── common/
│   │   ├── errors.go                 # AppError type, error constructors, sentinel errors
│   │   └── response.go               # Standard API Response struct, helpers
│   ├── config/
│   │   └── config.go                 # Config struct, Load() from env/.env
│   ├── database/
│   │   ├── connection.go             # DBConnection (GORM DB + sql.DB), NewDBConnection()
│   │   └── postgres.go               # Database wrapper, WithTxn, begin/commit/rollback
│   ├── domain/
│   │   ├── deliverystatus.go         # DeliveryStatus struct
│   │   ├── enums.go                  # Channel, Priority, NotificationType, NotificationStatus, DeliveryState, FailureClass
│   │   ├── errors.go                 # Domain sentinel errors (ErrTemplateNotFound, etc.)
│   │   ├── events.go                 # Event structs (NotificationCreated, PreferenceUpdated, WebhookDelivery)
│   │   ├── ids.go                    # Typed IDs (TemplateID, NotificationID, ClientID)
│   │   ├── notification.go           # Notification struct
│   │   ├── provider.go               # Provider enum (SendGrid, SES, Twilio, MSG91, FCM, APNS)
│   │   └── template.go               # Domain Template struct, ValidateVariables(), NextVersion()
│   ├── logger/
│   │   └── logger.go                 # Global zap logger, correlation ID context
│   ├── middleware/
│   │   ├── errors.go                 # HTTPErrorHandler (global Echo error handler)
│   │   ├── request.go                # RequestLoggerMiddleware
│   │   ├── request_id.go            # RequestIDMiddleware (X-Request-ID)
│   │   └── validation.go            # ValidateRequest, ValidateJSON, ValidateAndBind, etc.
│   ├── pagination/
│   │   └── pagination.go             # Offset-based pagination helpers
│   └── validation/
│       ├── errors.go                 # ValidationError struct, human-readable messages
│       └── validator.go              # Global validator.Validate, ValidateStruct()
│
└── services/
    └── template-svc/                 # Template Service module
        ├── .env
        ├── go.mod
        ├── go.sum
        ├── cmd/
        │   └── server/
        │       └── main.go           # Entry point: startup + DI + graceful shutdown
        ├── deploy/
        │   └── Dockerfile            # Multi-stage Docker build
        ├── internal/
        │   ├── server/
        │   │   └── server.go         # Echo server creation, middleware registration, Start
        │   └── template/
        │       ├── dto.go            # Request/Response DTOs
        │       ├── handler.go        # HTTP handlers + route registration
        │       ├── service.go        # Business logic
        │       ├── store.go          # Store interface (repository abstraction)
        │       ├── store_pg.go       # PostgreSQL Store implementation (GORM)
        │       └── types.go          # Local Template type
        ├── logs/
        │   └── app.log
        └── migrations/
            ├── 000001_create_templates_table.down.sql
            └── 000001_create_templates_table.up.sql
```

---

## 3. Application Startup Flow

```
main() [cmd/server/main.go]
  │
  ├── 1. config.Load()
  │     ├── Reads `ENV` env var (default: "local")
  │     ├── If local/dev: attempts godotenv.Load() (.env file)
  │     ├── Reads all env vars into Config struct:
  │     │     • PORT=:8080 (default)
  │     │     • DATABASE_CONN_URL (required)
  │     │     • DB_MAX_IDLE_CONNS=10
  │     │     • DB_MAX_OPEN_CONNS=100
  │     │     • DB_CONN_MAX_LIFETIME=1h
  │     │     • DB_CONN_MAX_IDLE_TIME=30m
  │     │     • LOG_ROOT_PATH=./logs
  │     │     • LOG_LEVEL=info
  │     │     • SHUTDOWN_TIMEOUT=15s
  │     │     • CORS_ALLOWED_ORIGINS=*
  │     ├── Validates: DATABASE_CONN_URL is non-empty
  │     └── Returns *Config or error (→ exit if error)
  │
  ├── 2. logger.Init(cfg.Env)
  │     ├── "production" → zap.NewProductionConfig()
  │     │     └── JSON output, ISO8601 timestamps
  │     └── else → zap.NewDevelopmentConfig()
  │           └── Colorized console output
  │
  ├── 3. logger.Get() → global *zap.Logger
  │
  ├── 4. database.NewDBConnection(cfg)
  │     ├── gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{Logger: logger.Warn})
  │     ├── db.DB() → *sql.DB
  │     ├── SetMaxIdleConns(10)
  │     ├── SetMaxOpenConns(100)
  │     ├── SetConnMaxLifetime(1h)
  │     ├── SetConnMaxIdleTime(30m)
  │     └── Returns *DBConnection{DB: *gorm.DB, SQL: *sql.DB}
  │
  │     ╔══════════════════════════════════════════════╗
  │     ║        DEPENDENCY INJECTION (Wiring)         ║
  │     ╚══════════════════════════════════════════════╝
  │
  ├── 5. store = template.NewPGStore(database.NewDatabase(db.DB))
  │     └── Wraps *gorm.DB in *database.Database → *template.PGStore
  │
  ├── 6. svc = template.NewService(store)
  │     └── Injects Store interface into Service
  │
  ├── 7. h = template.NewHandler(svc, logger)
  │     └── Injects Service + Logger into Handler
  │
  │     ╔══════════════════════════════════════════════╗
  │     ║           SERVER CREATION                    ║
  │     ╚══════════════════════════════════════════════╝
  │
  ├── 8. srv = server.New(cfg, logger, db)
  │     ├── echo.New() → *echo.Echo
  │     ├── registerMiddlewares():
  │     │   ├── s.Echo.Use(middleware.RequestIDMiddleware())
  │     │   ├── s.Echo.Use(middleware.RequestLoggerMiddleware(logger))
  │     │   └── s.Echo.HTTPErrorHandler = middleware.NewErrorHandler(logger, isProduction)
  │     └── Returns *Server
  │
  ├── 9. h.RegisterRoutes(srv.Echo.Group("/api/v1"))
  │     ├── POST   /api/v1/templates                  → h.Create
  │     ├── PUT    /api/v1/templates/:id              → h.Update
  │     ├── GET    /api/v1/templates/:id              → h.GetActive
  │     ├── GET    /api/v1/templates/:id/versions/:version → h.GetVersion
  │     ├── GET    /api/v1/templates                  → h.List
  │     └── DELETE /api/v1/templates/:id              → h.Deactivate
  │
  │     ╔══════════════════════════════════════════════╗
  │     ║        GRACEFUL SHUTDOWN SETUP               ║
  │     ╚══════════════════════════════════════════════╝
  │
  ├── 10. ctx, stop = signal.NotifyContext(context.Background(), SIGINT, SIGTERM)
  │      └── Context cancelled when SIGINT (Ctrl+C) or SIGTERM received
  │
  ├── 11. srv.Start(ctx)
  │      ├── Creates StartConfig{Address: ":8000", HideBanner: true, HidePort: true}
  │      ├── sc.Start(ctx, srv.Echo) → starts HTTP listener
  │      └── Blocks until context cancelled (signal received)
  │
  │     ╔══════════════════════════════════════════════╗
  │     ║        SHUTDOWN (deferred)                    ║
  │     ╚══════════════════════════════════════════════╝
  │
  ├── 12. db.Close() → closes *sql.DB
  │
  └── 13. logger.Sync() → flushes buffered log entries
```

---

## 4. Request Lifecycle (Request → Response)

This is the path **every HTTP request** travels through the system:

```
HTTP REQUEST
     │
     ▼
┌─────────────────────────────────────────────────────────┐
│  1. Echo Router Match                                    │
│     • Match URL + method against registered routes       │
│     • If no match → HTTPErrorHandler (404)              │
│     • If match → execute middleware chain + handler      │
└─────────────────────────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────────────────────────┐
│  2. RequestIDMiddleware [pkg/middleware/request_id.go]  │
│     • Read X-Request-ID header from request              │
│     • If missing → generate 32-char hex random ID        │
│     • Store in echo.Context (c.Set)                      │
│     • Set response header X-Request-ID                   │
│     • Bridge into Go's context.Context:                  │
│       logger.ContextWithCorrelationID(ctx, id)           │
│     • Update request with enriched context              │
│     • Call next middleware/handler                      │
└─────────────────────────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────────────────────────┐
│  3. RequestLoggerMiddleware [pkg/middleware/request.go]  │
│     • Record start time                                  │
│     • Call next middleware/handler (the actual handler)  │
│     • After handler returns:                             │
│       - Collect: method, path, uri, status, latency,    │
│         ip, user_agent, request_id                      │
│       - If error: append zap.Error(err)                 │
│       - If health check path (/healthz, /readyz):       │
│         log at DEBUG level                              │
│       - Else: log at INFO level                         │
│     • Return err (propagated up)                        │
└─────────────────────────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────────────────────────┐
│  4. Handler (e.g., template.Handler.Create)             │
│     • Parse path params, query params, request body     │
│     • Validate request (ValidateAndBind)                │
│     • Call service method                               │
│     • Map service errors → HTTP errors (AppError)      │
│     • Write JSON response (or return error)            │
└─────────────────────────────────────────────────────────┘
     │
     ├── On success: c.JSON(status, response) → direct write
     │
     └── On error: return error
              │
              ▼
     ┌──────────────────────────────────────────────────┐
     │  5. HTTPErrorHandler [pkg/middleware/errors.go]  │
     │     • If response already committed → skip       │
     │     • Check status code from error               │
     │     • 404 → route not found response              │
     │     • *AppError → serialize as JSON              │
     │     • Unknown → 500 internal server error        │
     └──────────────────────────────────────────────────┘
              │
              ▼
     ┌──────────────────────────────────────────────────┐
     │  6. Response written to client                    │
     │     Status code, headers (incl. X-Request-ID),   │
     │     JSON body                                     │
     └──────────────────────────────────────────────────┘
```

### Standard Response Format [pkg/common/response.go]

**Success:**
```json
{
  "success": true,
  "data": { ... },
  "meta": { "limit": 20, "offset": 0, "total": 100, "total_pages": 5 },
  "correlation_id": "abc123..."
}
```

**Error:**
```json
{
  "success": false,
  "error": {
    "code": 400,
    "error_code": "VALIDATION_ERROR",
    "message": "validation failed",
    "fields": [
      { "field": "name", "message": "name is required" }
    ]
  },
  "correlation_id": "abc123..."
}
```

---

## 5. Layer-by-Layer Breakdown

### 5.1 Config Layer (`pkg/config/config.go`)

**Purpose:** Load configuration from environment variables with `.env` file fallback.

**Key struct:**
```go
type Config struct {
    Env             string          // "local", "development", "staging", "production"
    Port            string          // ":8080"
    DatabaseURL     string          // PostgreSQL connection string
    LogRootPath     string          // "./logs"
    LogLevel        string          // "info"
    LogMaxSizeMB    int             // 100
    LogMaxAgeDays   int             // 30
    LogMaxBackups   int             // 5
    ShutdownTimeout time.Duration   // 15s
    DBMaxIdleConns    int
    DBMaxOpenConns    int
    DBConnMaxLifetime time.Duration
    DBConnMaxIdleTime time.Duration
    CORSAllowedOrigins []string
}
```

**Flow:**
1. Read `ENV` var → determines environment
2. If local/development → try `godotenv.Load()` to load `.env`
3. Read each var with defaults via helper functions
4. Validate: `DATABASE_CONN_URL` is required
5. Return `*Config` or error

**Helper functions:**
- `getEnvOrDefault(key, default)` → string
- `getEnvInt(key, default)` → int
- `getEnvDuration(key, default)` → time.Duration
- `getEnvSlice(key, default)` → []string (comma-separated)

### 5.2 Logger Layer (`pkg/logger/logger.go`)

**Purpose:** Global structured logger powered by `go.uber.org/zap` with correlation ID support.

**Key concepts:**
- Global `*zap.Logger` singleton (var log)
- `Init(environment)` → creates production/dev config
- `Get()` → returns global logger (creates dev fallback if not initialized)
- `ContextWithCorrelationID(ctx, id)` → stores correlation ID in Go context
- `CorrelationIDFromContext(ctx)` → retrieves correlation ID
- `WithContext(ctx)` → returns logger enriched with correlation ID field
- Context-aware helpers: `InfoContext`, `ErrorContext`, `DebugContext`, `WarnContext`, `FatalContext`

**Correlation ID flow:**
1. `RequestIDMiddleware` generates/reads request ID
2. Calls `logger.ContextWithCorrelationID(req.Context(), id)`
3. Sets updated context on request via `c.SetRequest(req.WithContext(ctx))`
4. Downstream code can call `logger.CorrelationIDFromContext(ctx)` to get the ID
5. Error responses include `correlation_id` field via `logger.CorrelationIDFromContext(c.Request().Context())`

### 5.3 Database Layer

#### `pkg/database/connection.go`

**Purpose:** Establish and configure the GORM PostgreSQL connection.

```go
type DBConnection struct {
    DB  *gorm.DB    // GORM ORM instance
    SQL *sql.DB     // Underlying sql.DB for pool config
}
```

**Flow:**
1. `gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Warn})` → opens connection
2. `db.DB()` → gets underlying `*sql.DB`
3. Configures pool: MaxIdleConns, MaxOpenConns, ConnMaxLifetime, ConnMaxIdleTime
4. `Close()` → closes `*sql.DB`
5. `Ping()` → health check

#### `pkg/database/postgres.go`

**Purpose:** Transaction wrapper and convenience Database type.

```go
type Database struct {
    *gorm.DB    // Embeds *gorm.DB
}
```

**Key methods:**
- `NewDatabase(db *gorm.DB)` → wraps GORM DB
- `WithContext(ctx)` → calls `d.DB.WithContext(ctx)` (returns `*Database`)
- `WithTxn(ctx, fn)` → full transaction lifecycle:
  1. `beginTxn(ctx)` → `db.Begin()`
  2. Execute `fn(tx)`
  3. On success: `commitTxn()` → `db.Commit()`
  4. On failure: `rollBackTxn()` → `db.Rollback()` (returns both errors if rollback fails)

### 5.4 Middleware Layer

#### `RequestIDMiddleware` (`pkg/middleware/request_id.go`)

**Order:** 1st in chain

**Logic:**
1. Read `X-Request-ID` from request header
2. If empty → generate 16 random bytes → hex encode → 32-char string
3. Store in `echo.Context` via `c.Set("request_id", id)`
4. Set response header `X-Request-ID: <id>`
5. Bridge into Go's standard context: `ctx = logger.ContextWithCorrelationID(req.Context(), id)`
6. Update request: `c.SetRequest(req.WithContext(ctx))`
7. Call next handler

**Why this matters:** Every downstream layer (service, store, logger, error responses) can access the correlation ID from Go's `context.Context` without depending on Echo.

#### `RequestLoggerMiddleware` (`pkg/middleware/request.go`)

**Order:** 2nd in chain

**Logic:**
1. Record `start := time.Now()`
2. Capture `path` and `uri`
3. Call `next(c)` — the actual handler
4. After handler returns, collect fields:
   - `method`, `path`, `uri`, `status`, `latency`, `ip`, `user_agent`, `request_id`
   - If error returned: `zap.Error(err)`
5. If path ends with `/healthz` or `/readyz` → log at `DEBUG` level
6. Otherwise → log at `INFO` level
7. Return error (if any) to propagate to `HTTPErrorHandler`

#### `HTTPErrorHandler` (`pkg/middleware/errors.go`)

**Purpose:** Global error handler assigned to `s.Echo.HTTPErrorHandler`. Catches all unhandled errors from handlers and middleware.

**Logic:**
1. If response already committed → skip (no double-write)
2. Get HTTP status from error via `echo.StatusCode(err)`
3. If status == 404 → return route-not-found response using `common.NewRouteNotFoundError()`
4. If error is `*AppError` → serialize as JSON response (with status code from `AppError.Code`)
5. Otherwise → log unhandled error + return 500 with `common.NewInternalError()`

**Error type hierarchy:**
```
error
 ├── echo.HTTPError (from Echo itself, e.g., 404, 405)
 ├── *common.AppError
 │     ├── Code (HTTP status)
 │     ├── ErrorCode (string code like "VALIDATION_ERROR")
 │     ├── Message
 │     └── Err (wrapped underlying error)
 └── other (anything else → 500)
```

#### `Validation Middleware` (`pkg/middleware/validation.go`)

**Purpose:** Request validation and binding utilities. Not a single middleware — a collection of helper functions used **inside handlers**.

**Key functions:**

| Function | Purpose |
|----------|---------|
| `ValidateAndBind(c, req)` | Binds + validates body; sends error response on failure |
| `ValidateAndBindQuery(c, req)` | Binds + validates query params |
| `ValidateJSON(c, req)` | Binds body + validates, returns error |
| `ValidateQuery(c, req)` | Binds query + validates, returns error |
| `ValidateURI(c, req)` | Binds path params + validates |
| `RespondWithValidationError(c, err)` | Writes 400 with field-level errors |
| `ValidateRequest(requestType)` | **Middleware factory** — can be used as route middleware to auto-validate |

**Validation flow (ValidateAndBind):**
1. `c.Bind(req)` — JSON deserialization
2. `validation.ValidateStruct(req)` — struct tag validation via `go-playground/validator`
3. On error → `RespondWithValidationError`:
   - Converts to `*ValidationError` with field→message map
   - Sorts field names alphabetically
   - Writes 400 JSON response with standard format
4. On success → return nil

### 5.5 Handler Layer (`internal/template/handler.go`)

**Purpose:** HTTP concerns only. Parse, validate, call service, map errors, respond.

**Handler struct:**
```go
type Handler struct {
    svc *Service   // Business logic
    log *zap.Logger // Logging
}
```

**Route registration:**
```go
func (h *Handler) RegisterRoutes(g *echo.Group) {
    g.POST("/templates", h.Create)         // Create template
    g.PUT("/templates/:id", h.Update)      // Update template (new version)
    g.GET("/templates/:id", h.GetActive)   // Get active version
    g.GET("/templates/:id/versions/:version", h.GetVersion) // Get specific version
    g.GET("/templates", h.List)            // List active templates
    g.DELETE("/templates/:id", h.Deactivate) // Deactivate template
}
```

**Common handler pattern:**
```go
func (h *Handler) SomeMethod(c *echo.Context) error {
    // 1. Parse request (params, query, body)
    // 2. Validate (ValidateAndBind or manual checks)
    // 3. Call service layer
    // 4. Handle errors with domain error matching
    // 5. Return JSON response
}
```

**Error mapping in handlers:**
```
domain.ErrTemplateNotFound  → common.NewNotFoundError()      → 404
domain.ErrConcurrentUpdate  → common.NewConflictError()       → 409
Any other error             → common.NewInternalError()       → 500
```

### 5.6 Service Layer (`internal/template/service.go`)

**Purpose:** Business logic. Orchestrates operations, generates IDs, manages versions.

**Service struct:**
```go
type Service struct {
    store Store  // Repository abstraction
}
```

**Business logic per method:**

| Method | Logic |
|--------|-------|
| `Create` | Generate UUID v4, set version=1, extract variables from subject/body (template syntax `{{var}}`), call store.Create |
| `Update` | Get active template, extract variables, increment version (+1), call store.CreateNewVersion (transactional) |
| `GetActive` | Delegate to store.GetActive |
| `GetVersion` | Delegate to store.GetVersion |
| `List` | Clamp limit (1-100, default 20), delegate to store.List, convert to response DTOs |
| `Deactivate` | Delegate to store.Deactivate |

**Variable extraction (`extractVariables`):**
- Scans subject + body for `{{variableName}}` patterns
- Deduplicates by name
- Returns unique sorted list of variable names
- Called automatically if `Variables` field is nil in request

**Response mapping (`toResponse`):**
- Converts internal `Template` → `*TemplateResponse` DTO

### 5.7 Store (Repository) Layer

#### Interface (`internal/template/store.go`)
```go
type Store interface {
    Create(ctx context.Context, t Template) error
    GetActive(ctx context.Context, id string) (Template, error)
    GetVersion(ctx context.Context, id string, version int) (Template, error)
    CreateNewVersion(ctx context.Context, next Template) error
    Deactivate(ctx context.Context, id string) error
    List(ctx context.Context, limit int, cursor string) ([]Template, string, error)
}
```

**Why an interface?** Enables testing with mocks, future replacements (e.g., caching layer).

#### PostgreSQL Implementation (`internal/template/store_pg.go`)

**GORM Model:**
```go
type templateModel struct {
    TemplateID string      `gorm:"column:template_id;primaryKey"`
    Version    int         `gorm:"column:version;primaryKey"`
    Name       string      `gorm:"column:name"`
    Channel    string      `gorm:"column:channel"`
    Subject    string      `gorm:"column:subject"`
    Body       string      `gorm:"column:body"`
    Variables  StringSlice `gorm:"column:variables;type:jsonb;serializer:json"`
    IsActive   bool        `gorm:"column:is_active"`
    CreatedAt  time.Time   `gorm:"column:created_at"`
}
```

**Custom type `StringSlice`:** Implements `driver.Value` (→ JSON marshal) and `Scanner` (← JSON unmarshal) for PostgreSQL JSONB arrays.

**Method implementations:**

| Method | SQL Pattern | Notes |
|--------|------------|-------|
| `Create` | `INSERT INTO templates ...` | Simple GORM Create |
| `GetActive` | `SELECT * FROM templates WHERE template_id=? AND is_active=true LIMIT 1` | Returns `domain.ErrTemplateNotFound` if gorm.ErrRecordNotFound |
| `GetVersion` | `SELECT * FROM templates WHERE template_id=? AND version=? LIMIT 1` | Same error mapping |
| `CreateNewVersion` | **Transaction:** 1) `UPDATE templates SET is_active=false WHERE template_id=? AND is_active=true` 2) `INSERT INTO templates ...` | Atomic; checks RowsAffected==0 → ErrTemplateNotFound; unique violation → ErrConcurrentUpdate |
| `Deactivate` | `UPDATE templates SET is_active=false WHERE template_id=? AND is_active=true` | Checks RowsAffected==0 → ErrTemplateNotFound |
| `List` | `SELECT * FROM templates WHERE is_active=true ORDER BY created_at DESC LIMIT ?` (+ cursor clause) | Cursor-based keyset pagination |

**Cursor-based pagination in `List`:**
- Cursor is a template ID
- If cursor provided: `WHERE created_at < (SELECT created_at FROM templates WHERE template_id = ?)`
- Ordered by `created_at DESC`
- Returns `limit` rows + next cursor (last item's ID if exactly `limit` rows returned)

**Unique violation detection:**
```go
func isUniqueViolation(err error) bool {
    var pgErr interface{ SQLState() string }
    return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
```
Error code `23505` = PostgreSQL unique violation. Used to detect concurrent version creation.

### 5.8 Domain Layer (`pkg/domain/`)

**Purpose:** Pure domain types with zero external dependencies.

**Types:**
| Type | Fields | Purpose |
|------|--------|---------|
| `Template` | ID, Version, Name, Channel, Subject, Body, Variables, IsActive, CreatedAt | Domain template |
| `Notification` | ID, ClientID, ExternalUserID, TemplateID, Channel, Priority, Type, Status, Subject, Body, Variables, ScheduledAt, RetryCount, CreatedAt, UpdatedAt | Notification |
| `DeliveryStatus` | ID, NotificationID, Channel, Provider, ProviderMessageID, State, FailureClass, FailureReason, AttemptNumber, CreatedAt | Per-channel delivery tracking |
| `TemplateID/NotificationID/ClientID` | typed strings | Type safety |

**Enums (`enums.go`):**
```
Channel: EMAIL, SMS, PUSH, INAPP
Priority: CRITICAL, HIGH, NORMAL, LOW
NotificationType: TRANSACTIONAL, PROMOTIONAL, ALERT
NotificationStatus: PENDING, SCHEDULED, SENT, DELIVERED, FAILED, CANCELLED
DeliveryState: SENT, DELIVERED, FAILED, BOUNCED, OPENED, CLICKED
FailureClass: TEMPORARY, PERMANENT
```

**Status transition validation:**
```go
var validTransitions = map[NotificationStatus][]NotificationStatus{
    StatusPending:   {StatusScheduled, StatusSent, StatusFailed, StatusCancelled},
    StatusScheduled: {StatusPending, StatusCancelled},
    StatusSent:      {StatusDelivered, StatusFailed},
    StatusDelivered: {},
    StatusFailed:    {},
    StatusCancelled: {},
}
```

**Sentinel errors (`domain/errors.go`):**
```go
var (
    ErrTemplateNotFound     = errors.New("template not found")
    ErrTemplateInactive     = errors.New("template is not active")
    ErrMissingVariable      = errors.New("missing required variable")
    ErrInvalidChannel       = errors.New("invalid channel")
    ErrNotificationNotFound = errors.New("notification not found")
    ErrPreferencesNotFound  = errors.New("preferences not found")
    ErrAllChannelsBlocked   = errors.New("all requested channels blocked by preferences")
    ErrInvalidTransition    = errors.New("invalid status transition")
    ErrDuplicateDelivery    = errors.New("delivery already recorded, idempotent skip")
    ErrRateLimited          = errors.New("rate limited, retry later")
    ErrConcurrentUpdate     = errors.New("concurrent update detected, retry")
)
```

These are used for `errors.Is()` checks in handlers and services.

**Events (`events.go`):** Event structs for future async processing:
- `NotificationCreatedEvent` — emitted after notification creation
- `PreferenceUpdatedEvent` — emitted when user preferences change
- `WebhookDeliveryEvent` — emitted on delivery webhook receipt

### 5.9 Validation Layer (`pkg/validation/`)

**Global validator:**
```go
var Validate *validator.Validate = validator.New()
```

**ValidateStruct:**
```go
func ValidateStruct(s interface{}) error {
    err := Validate.Struct(s)
    if err != nil {
        if validationErrors, ok := err.(validator.ValidationErrors); ok {
            return NewValidationError(validationErrors)
        }
        return err
    }
    return nil
}
```

**ValidationError struct:**
- `Errors map[string]string` — field → human-readable message
- `NewValidationError(errs validator.ValidationErrors)` — converts validator errors to messages
- Supported tags: `required`, `email`, `min`, `max`, `gte`, `lte`, `gt`, `lt`, `len`, `oneof`, `phone`, `uuid`, `url`, `alphanum`, `alpha`, `numeric`

**DTO validation tags (from `dto.go`):**
```go
type CreateTemplateRequest struct {
    Name      string   `json:"name" validate:"required,min=3,max=255"`
    Channel   string   `json:"channel" validate:"required,oneof=EMAIL SMS PUSH INAPP"`
    Subject   string   `json:"subject,omitempty"`
    Body      string   `json:"body" validate:"required"`
    Variables []string `json:"variables,omitempty"`
}
```

### 5.10 Common/Response Layer

#### `pkg/common/response.go`

**Standard response struct:**
```go
type Response struct {
    Success       bool        `json:"success"`
    Data          interface{} `json:"data,omitempty"`
    Error         *ErrorInfo  `json:"error,omitempty"`
    Meta          *Meta       `json:"meta,omitempty"`
    CorrelationID string      `json:"correlation_id,omitempty"`
}
```

**Helper functions:**
| Function | HTTP Status | Usage |
|----------|-------------|-------|
| `SuccessResponse(c, data)` | 200 | Generic success |
| `CreatedResponse(c, data)` | 201 | Resource created |
| `SuccessResponseWithMeta(c, data, meta)` | 200 | Paginated responses |
| `ErrorResponse(c, status, message)` | varies | Simple error |
| `ValidationErrorResponse(c, fields)` | 400 | Field-level errors |
| `AppErrorResponse(c, err)` | varies | From AppError |
| `NoRouteHandler()` | 404 | Route not found |
| `NoMethodHandler()` | 405 | Method not allowed |

#### `pkg/common/errors.go`

**AppError type:**
```go
type AppError struct {
    Code      int    `json:"code"`
    ErrorCode string `json:"error_code,omitempty"`
    Message   string `json:"message"`
    Err       error  `json:"-"` // hidden from JSON
}
```

**Error code constants:**
```
AUTH_UNAUTHORIZED, AUTH_FORBIDDEN, AUTH_INVALID_TOKEN, AUTH_EXPIRED_TOKEN
AUTH_INVALID_CREDENTIALS, VALIDATION_ERROR, BAD_REQUEST, RESOURCE_NOT_FOUND
RESOURCE_CONFLICT, INTERNAL_ERROR, SERVICE_UNAVAILABLE, RATE_LIMITED, ROUTE_NOT_FOUND
```

**Sentinel errors (for `errors.Is` matching):**
```go
ErrUnauthorized, ErrForbidden, ErrValidation, ErrNotFound, ErrBadRequest,
ErrConflict, ErrInvalidCredentials, ErrInternalServer, ErrExpiredToken,
ErrInvalidToken, ErrServiceUnavailable, ErrRateLimited
```

**Constructor functions:**
```go
NewNotFoundError(message, err)       → 404, ErrCodeNotFound
NewBadRequestError(message, err)     → 400, ErrCodeBadRequest
NewInternalError(message, err)       → 500, ErrCodeInternal
NewConflictError(message)            → 409, ErrCodeConflict
NewValidationError(message)          → 400, ErrCodeValidation
NewUnauthorizedError(message)        → 401, ErrCodeUnauthorized
NewForbiddenError(message)           → 403, ErrCodeForbidden
NewServiceUnavailableError(message)  → 503, ErrCodeServiceUnavailable
NewTooManyRequestsError(message)     → 429, ErrCodeRateLimited
NewRouteNotFoundError(message, err)  → 404, ErrRouteNotFound
NewInternalServerError(message)      → 500, ErrCodeInternal
```

Each constructor wraps a sentinel error (e.g., `ErrNotFound`) so `errors.Is(appErr, common.ErrNotFound)` works.

### 5.11 Pagination Layer (`pkg/pagination/pagination.go`)

**Purpose:** Offset-based pagination (not currently used by template-svc, but available for future services).

```go
type Params struct {
    Limit  int `form:"limit" json:"limit"`
    Offset int `form:"offset" json:"offset"`
}
```

**Note:** The template service uses **cursor-based pagination** directly in `store_pg.go`, not the offset-based `pagination.Params`.

---

## 6. API Endpoints & Flow Diagrams

### 6.1 POST /api/v1/templates — Create Template

```
Client                                     Echo Server
  │                                            │
  │  POST /api/v1/templates                    │
  │  Content-Type: application/json            │
  │  Body: {"name":"Welcome","channel":"EMAIL",│
  │         "subject":"Hi {{name}}","body":"Hello {{name}}"}
  │                                            │
  │───────────────────────────────────────────►│
  │                                            │
  │           1. RequestIDMiddleware            │
  │              ├─ Read/generate X-Request-ID  │
  │              ├─ Set response header         │
  │              └─ Bridge to Go context        │
  │                                            │
  │           2. RequestLoggerMiddleware        │
  │              ├─ Record start time           │
  │              └─ Call next                   │
  │                                            │
  │           3. Handler.Create [handler.go:34] │
  │              ├─ var req CreateTemplateRequest│
  │              ├─ ValidateAndBind(&req)       │
  │              │   ├─ c.Bind(&req)            │
  │              │   └─ validation.ValidateStruct│
  │              │       ├─ name: required,min=3│
  │              │       ├─ channel: required,  │
  │              │       │   oneof=EMAIL,SMS,...│
  │              │       └─ body: required      │
  │              │                                 if fail ─► 400 validation error
  │              ├─ svc.Create(ctx, req)        │
  │              │   ├─ extractVariables()      │
  │              │   │   └─ Scans for {{var}}   │
  │              │   │      returns ["name"]    │
  │              │   ├─ Template{               │
  │              │   │   ID: uuid.New(),        │
  │              │   │   Version: 1,            │
  │              │   │   IsActive: true,        │
  │              │   │   CreatedAt: now()       │
  │              │   │ }                        │
  │              │   └─ store.Create(ctx, t)    │
  │              │       └─ db.Create(&model)   │
  │              │           └─ INSERT INTO     │
  │              │              templates ...   │
  │              │                                 if fail ─► 500 internal error
  │              └─ c.JSON(201, toResponse(t))  │
  │                                            │
  │  ◄─────────────────────────────────────────│
  │  HTTP 201 Created                          │
  │  X-Request-ID: abc123                      │
  │  Body: {"id":"uuid","version":1,           │
  │         "name":"Welcome",...}              │
```

### 6.2 PUT /api/v1/templates/:id — Update Template (Versioned)

```
Client                                    Echo Server
  │                                            │
  │  PUT /api/v1/templates/abc-123             │
  │  Body: {"subject":"New","body":"New body"} │
  │───────────────────────────────────────────►│
  │                                            │
  │  Middleware chain (same as above)          │
  │                                            │
  │  Handler.Update [handler.go:48]            │
  │    ├─ id = c.Param("id") → "abc-123"       │
  │    ├─ Validate id not empty                 │
  │    ├─ Bind & validate request body          │
  │    ├─ svc.Update(ctx, id, req)             │
  │    │   ├─ store.GetActive(ctx, id)         │
  │    │   │   └─ SELECT ... WHERE template_id │
  │    │   │      = ? AND is_active = true     │
  │    │   │                                    │
  │    │   │   if not found ──► ErrTemplateNotFound ──► 404
  │    │   │
  │    │   ├─ extractVariables()               │
  │    │   ├─ Template{                         │
  │    │   │   Version: current.Version + 1,    │
  │    │   │   IsActive: true                   │
  │    │   │ }                                  │
  │    │   └─ store.CreateNewVersion(ctx, next) │
  │    │       └─ WithTxn:                      │
  │    │           ├─ UPDATE templates          │
  │    │           │   SET is_active=false      │
  │    │           │   WHERE template_id=?      │
  │    │           │   AND is_active=true       │
  │    │           │                              │
  │    │           │   if RowsAffected==0 ──► ErrTemplateNotFound
  │    │           │                              │
  │    │           └─ INSERT new version        │
  │    │               if unique violation ──► ErrConcurrentUpdate ──► 409
  │    │
  │    └─ c.JSON(200, toResponse(next))        │
  │                                            │
  │  ◄── 200 OK                                │
```

### 6.3 GET /api/v1/templates/:id — Get Active Template

```
  Handler.GetActive [handler.go:73]
    ├─ id = c.Param("id")
    ├─ Validate id not empty
    ├─ svc.GetActive(ctx, id)
    │   └─ store.GetActive(ctx, id)
    │       └─ SELECT ... WHERE template_id=? AND is_active=true
    │           if not found ──► ErrTemplateNotFound ──► 404
    └─ c.JSON(200, toResponse(t))
```

### 6.4 GET /api/v1/templates/:id/versions/:version — Get Specific Version

```
  Handler.GetVersion [handler.go:91]
    ├─ id = c.Param("id")
    ├─ version = strconv.Atoi(c.Param("version"))
    │   if invalid ──► 400
    ├─ svc.GetVersion(ctx, id, version)
    │   └─ store.GetVersion(ctx, id, version)
    │       └─ SELECT ... WHERE template_id=? AND version=?
    │           if not found ──► ErrTemplateNotFound ──► 404
    └─ c.JSON(200, toResponse(t))
```

### 6.5 GET /api/v1/templates — List Templates (Cursor Pagination)

```
  Handler.List [handler.go:114]
    ├─ limit = parseInt(c.QueryParam("limit"), 20)
    │   if limit <= 0 || limit > 100 → clamp to 20
    ├─ cursor = c.QueryParam("cursor") → template ID
    ├─ svc.List(ctx, limit, cursor)
    │   └─ store.List(ctx, limit, cursor)
    │       ├─ Base: SELECT ... WHERE is_active=true
    │       │       ORDER BY created_at DESC LIMIT ?
    │       ├─ If cursor: WHERE created_at < (
    │       │   SELECT created_at FROM templates
    │       │   WHERE template_id = ?)
    │       └─ If len(results) == limit → set next_cursor = last item's ID
    │           else → next_cursor = ""
    └─ c.JSON(200, ListTemplatesResponse{
           Templates: [...],
           NextCursor: "next-id-or-empty"
       })
```

### 6.6 DELETE /api/v1/templates/:id — Deactivate Template

```
  Handler.Deactivate [handler.go:127]
    ├─ id = c.Param("id")
    ├─ Validate id not empty
    ├─ svc.Deactivate(ctx, id)
    │   └─ store.Deactivate(ctx, id)
    │       └─ UPDATE templates SET is_active=false
    │           WHERE template_id=? AND is_active=true
    │           if RowsAffected==0 ──► ErrTemplateNotFound ──► 404
    └─ c.NoContent(204)
```

---

## 7. Error Handling Flow

The error handling architecture has 5 layers:

```
Layer 1: Domain Sentinel Errors (pkg/domain/errors.go)
  • errors.New("template not found")  ← package-level vars
  • Used with errors.Is() for domain logic branching

Layer 2: Store Error Mapping (internal/template/store_pg.go)
  • gorm.ErrRecordNotFound → domain.ErrTemplateNotFound
  • pg unique violation (23505) → domain.ErrConcurrentUpdate
  • Other DB errors → wrapped with fmt.Errorf

Layer 3: Handler Error Mapping (internal/template/handler.go)
  • errors.Is(err, domain.ErrTemplateNotFound) → common.NewNotFoundError()
  • errors.Is(err, domain.ErrConcurrentUpdate) → common.NewConflictError()
  • anything else → common.NewInternalError()

Layer 4: AppError (pkg/common/errors.go)
  • Structured error with HTTP status, error code, message, wrapped cause
  • Supports errors.Is() and errors.As()
  • Multiple constructors for each error category

Layer 5: HTTPErrorHandler (pkg/middleware/errors.go)
  • Catch-all for any error reaching Echo's error handler
  • 404 → route not found
  • *AppError → serialize as JSON with status code
  • Unknown → 500 internal server error
```

### Error propagation diagram:

```
Store (PG)
  │
  │ return domain.ErrTemplateNotFound
  ▼
Handler.GetActive
  │
  │ if errors.Is(err, domain.ErrTemplateNotFound)
  │     return common.NewNotFoundError("template not found", err)
  ▼
HTTPErrorHandler
  │
  │ errors.As(err, &appErr) → true
  │ c.JSON(status, appErr)
  ▼
Client receives:
  HTTP 404
  { "success": false, "error": { "code": 404, "error_code": "RESOURCE_NOT_FOUND", "message": "template not found" }, "correlation_id": "..." }
```

### Error flow for unhandled panics / unknown errors:

```
Handler returns unexpected error
  │
  ▼
RequestLoggerMiddleware catches it:
  │ log fields include zap.Error(err)
  │ returns err
  ▼
HTTPErrorHandler:
  │ status = 500 (default)
  │ log.Error("unhandled error", ...)
  │ c.JSON(500, common.NewInternalError("internal server error", err))
  ▼
Client receives:
  HTTP 500
  { "success": false, "error": { "code": 500, "error_code": "INTERNAL_ERROR", "message": "internal server error" } }
```

### Validation error flow:

```
Handler.Create
  │
  │ ValidateAndBind(c, &req)
  │   ├── c.Bind(&req) → JSON deserialization
  │   └── validation.ValidateStruct(req)
  │         └── validator.Validate.Struct(req)
  │               └── returns validator.ValidationErrors
  │                     └── NewValidationError → ValidationError{Errors: map}
  ▼
RespondWithValidationError(c, err):
  │ toFieldErrors(err) → []FieldError
  │ c.JSON(400, Response{
  │     Success: false,
  │     Error: { Code: 400, ErrorCode: "VALIDATION_ERROR",
  │              Message: "validation failed", Fields: [...] },
  │     CorrelationID: ...
  │ })
  ▼
Handler returns the error from ValidateAndBind (which already wrote the response)
HTTPErrorHandler sees committed response → skips
```

---

## 8. Database Schema & Migrations

### Table: `templates`

```sql
CREATE TABLE IF NOT EXISTS templates (
    template_id UUID NOT NULL,
    version     INTEGER NOT NULL DEFAULT 1,
    name        VARCHAR(255) NOT NULL,
    channel     VARCHAR(20) NOT NULL,
    subject     VARCHAR(500) NOT NULL DEFAULT '',
    body        TEXT NOT NULL,
    variables   JSONB NOT NULL DEFAULT '[]',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (template_id, version)
);

CREATE INDEX idx_templates_active ON templates (template_id, is_active) WHERE is_active = true;
CREATE INDEX idx_templates_name ON templates (name);
CREATE INDEX idx_templates_channel ON templates (channel);
```

### Key Design Decisions:

1. **Composite Primary Key** `(template_id, version)`:
   - Allows multiple versions of the same template
   - Version tracking for audit trail
   - `is_active` flag determines which version is current
   - Partial index `idx_templates_active` for efficient active template lookups

2. **UUID as template_id**:
   - Generated by application (`github.com/google/uuid`)
   - Not using PostgreSQL `gen_random_uuid()` for portability
   - UUID v4 as string (no dashes in stored form)

3. **JSONB for Variables**:
   - Array of strings: `["name", "email", "product"]`
   - Uses custom `StringSlice` type for GORM scanning
   - Flexible schema evolution without migrations

4. **No foreign key constraints** (yet):
   - Template-svc is independent; no references to other tables
   - Future services (notification-svc) will reference template_id

---

## 9. Graceful Shutdown Flow

```
main() [cmd/server/main.go:43-54]

ctx, stop := signal.NotifyContext(context.Background(), SIGINT, SIGTERM)
defer stop()
                 │
                 ▼
srv.Start(ctx)
  │
  ├── sc.Start(ctx, srv.Echo)
  │     ├── HTTP server starts listening on :8000
  │     └── Blocks (select on ctx.Done())
  │
  │   ┌──────────────────────────────────────┐
  │   │  OS sends SIGINT or SIGTERM          │
  │   │  ─────────────────────────────►      │
  │   │  ctx is cancelled                     │
  │   └──────────────────────────────────────┘
  │                 │
  │                 ▼
  │   sc.Start returns
  │   ├── HTTP server stops accepting new connections
  │   └── Existing requests are drained (with SHUTDOWN_TIMEOUT=15s)
  │
  ├── srv.Start returns error (or nil)
  │
  ▼
db.Close() → closes *sql.DB (waits for active queries)
  │
  ▼
logger.Sync() → flushes buffered log entries to disk
  │
  ▼
Program exits
```

**Shutdown sequence:**
1. OS sends SIGINT/SIGTERM
2. `signal.NotifyContext` cancels the context
3. `echo.StartConfig.Start()` receives cancellation
4. HTTP server performs graceful shutdown:
   - Stops accepting new connections
   - Waits up to `SHUTDOWN_TIMEOUT` (15s) for in-flight requests to complete
   - If timeout expires, forcefully closes connections
5. `Start()` returns
6. `db.Close()` — closes PostgreSQL connection pool
7. `logger.Sync()` — ensures all log entries are written to disk

---

## Appendix: Key Code Paths Summary

| Action | Files traversed (in order) |
|--------|---------------------------|
| Startup | `main.go` → `config.go` → `logger.go` → `connection.go` → `store_pg.go` → `service.go` → `handler.go` → `server.go` → `main.go` |
| Create Template | `request_id.go` → `request.go` → `handler.go:Create` → `service.go:Create` → `store_pg.go:Create` |
| Update Template | `request_id.go` → `request.go` → `handler.go:Update` → `service.go:Update` → `store_pg.go:GetActive` → `store_pg.go:CreateNewVersion` (txn) |
| Get Template | `request_id.go` → `request.go` → `handler.go:GetActive/GetVersion` → `service.go:GetActive/GetVersion` → `store_pg.go:GetActive/GetVersion` |
| List Templates | `request_id.go` → `request.go` → `handler.go:List` → `service.go:List` → `store_pg.go:List` |
| Deactivate Template | `request_id.go` → `request.go` → `handler.go:Deactivate` → `service.go:Deactivate` → `store_pg.go:Deactivate` |
| Error (domain) | `store_pg.go` → `handler.go` (error mapping) → `common/errors.go` (AppError) → `middleware/errors.go` (HTTPErrorHandler) → client |
| Error (validation) | `handler.go` → `middleware/validation.go:ValidateAndBind` → `validation/errors.go` → `common/response.go` → client |
| Error (404 route) | Echo router → `middleware/errors.go:NewErrorHandler` → `common/errors.go:NewRouteNotFoundError` → client |
| Graceful Shutdown | OS signal → `signal.NotifyContext` → `server.go:Start` → `main.go:db.Close` → `main.go:logger.Sync` |
