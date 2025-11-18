# SoulSolid Feature Development Spec Kit

This document provides comprehensive guidance for developing new features in the SoulSolid music library management application.

## Architecture Overview

SoulSolid follows a modular, feature-based architecture with clear separation of concerns:

```
src/
├── features/           # Feature modules (business logic)
│   ├── config/        # Configuration management
│   ├── downloading/   # Music download functionality
│   ├── importing/     # Library import/processing
│   ├── jobs/          # Background job management
│   ├── library/       # Music library operations
│   ├── hosting/       # HTTP server and middleware
│   └── ui/           # UI page rendering
├── infra/             # Infrastructure services
│   ├── database/     # Database layer
│   ├── metadata/     # External metadata providers
│   ├── queue/        # Job queue implementation
│   └── watcher/      # File system monitoring
├── music/            # Core domain models
└── main.go          # Application entry point
```

## Core Technologies

- **Backend**: Go 1.25+ with Fiber web framework
- **Frontend**: HTMX + Hyperscript for dynamic interactions
- **Styling**: TailwindCSS with dark mode support
- **Database**: SQLite
- **Templates**: Go HTML templates with custom functions
- **Jobs**: In-memory queue system
- **Dependencies**: Managed via Go modules and npm

## Feature Structure Pattern

Every feature follows this consistent structure:

```
features/yourfeature/
├── handlers.go       # HTTP request handlers
├── routes.go         # Route registration
├── service.go        # Business logic layer
├── models.go         # Feature-specific data structures
└── telegram.go       # (Optional) Telegram bot integration
```

### 1. Service Layer (`service.go`)

The service layer contains business logic and follows this pattern:

```go
package yourfeature

import (
    "log/slog"
    "github.com/contre95/soulsolid/src/infra/database"
)

type Service struct {
    db          *database.SqliteLibrary
    configMgr   *config.Manager
    // other dependencies
}

func NewService(db *database.SqliteLibrary, configMgr *config.Manager) *Service {
    return &Service{
        db:        db,
        configMgr: configMgr,
    }
}

func (s *Service) DoSomething(input InputType) (OutputType, error) {
    slog.Debug("Doing something", "input", input)
    
    // Business logic here
    
    return result, nil
}
```

### 2. Handler Layer (`handlers.go`)

Handlers manage HTTP requests and support both API (JSON) and UI (HTML) responses:

```go
package yourfeature

import (
    "log/slog"
    "github.com/gofiber/fiber/v2"
)

type Handler struct {
    service *Service
}

func NewHandler(service *Service) *Handler {
    return &Handler{service: service}
}

func (h *Handler) HandleAction(c *fiber.Ctx) error {
    slog.Debug("HandleAction called")
    
    var req RequestType
    if err := c.BodyParser(&req); err != nil {
        // HTMX error response
        if c.Get("HX-Request") == "true" {
            return c.Render("toast/toastErr", fiber.Map{
                "Msg": "Invalid request body",
            })
        }
        // API error response
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }
    
    // Call service
    result, err := h.service.DoSomething(req)
    if err != nil {
        slog.Error("Service failed", "error", err)
        if c.Get("HX-Request") == "true" {
            return c.Render("toast/toastErr", fiber.Map{
                "Msg": "Operation failed",
            })
        }
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Operation failed",
        })
    }
    
    // HTMX success response
    if c.Get("HX-Request") == "true" {
        return c.Render("yourfeature/result_partial", fiber.Map{
            "Result": result,
        })
    }
    
    // API success response
    return c.JSON(result)
}
```

### 3. Routes (`routes.go`)

Route registration follows a consistent pattern:

```go
package yourfeature

import (
    "github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, service *Service) {
    handler := NewHandler(service)
    
    // API routes
    api := app.Group("/yourfeature")
    api.Post("/action", handler.HandleAction)
    api.Get("/items", handler.GetItems)
    
    // UI routes (for HTMX partials)
    ui := app.Group("/ui")
    ui.Get("/yourfeature/partial", handler.GetPartial)
}
```

## HTMX Integration Patterns

### 1. Detection and Dual Response Handling

All handlers must check for HTMX requests and respond appropriately:

```go
if c.Get("HX-Request") == "true" {
    // Return HTML partial for UI
    return c.Render("partial_template", data)
}
// Return JSON for API
return c.JSON(data)
```

### 2. Common HTMX Attributes Used

- `hx-get` / `hx-post`: Make requests
- `hx-target`: Target element for content swap
- `hx-swap`: How to swap content (`innerHTML`, `outerHTML`, etc.)
- `hx-trigger`: When to trigger request (`load`, `click`, etc.)
- `hx-indicator`: Loading indicator element

### 3. Loading States

Use consistent loading patterns:

```html
<div hx-get="/api/data" hx-indicator="#loading">
    <div id="loading" class="htmx-indicator">
        <div class="spinner"></div>
    </div>
    <!-- Content will be replaced -->
</div>
```

### 4. Toast Notifications

Use the toast system for user feedback:

```go
// Success
return c.Render("toast/toastOk", fiber.Map{
    "Msg": "Operation completed successfully!",
})

// Error
return c.Render("toast/toastErr", fiber.Map{
    "Msg": "Operation failed!",
})

// Info
return c.Render("toast/toastInfo", fiber.Map{
    "Msg": "Processing...",
})
```

## Template Structure

### 1. Main Layout (`views/partials/main.html`)

The main template uses conditional rendering based on the `.Section` variable:

```html
<div id="contenido">
    {{if eq .Section "dashboard"}}
    {{template "sections/dashboard" .}}
    {{else if eq .Section "yourfeature"}}
    {{template "sections/yourfeature" .}}
    {{end}}
</div>
```

### 2. Feature Section Template (`views/sections/yourfeature.html`)

Create a section template for your feature:

```html
<div id="contenido" class="animate__animated animate__fadeIn">
    <h1 class="text-3xl font-bold text-slate-800 dark:text-white mb-8">
        Your Feature
    </h1>
    
    <!-- Feature content here -->
    <div hx-get="/ui/yourfeature/partial" hx-trigger="load">
        Loading...
    </div>
</div>
```

### 3. Partial Templates (`views/yourfeature/`)

Create reusable partials for dynamic content:

```html
<!-- views/yourfeature/item_list.html -->
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    {{range .Items}}
    <div class="bg-white dark:bg-gray-800 rounded-lg p-4">
        <h3 class="font-semibold">{{.Name}}</h3>
        <p class="text-sm text-gray-600 dark:text-gray-400">{{.Description}}</p>
    </div>
    {{else}}
    <p class="text-gray-500 col-span-full">No items found</p>
    {{end}}
</div>
```

## Dependency Injection Pattern

Dependencies are injected through the main.go file:

```go
// In main.go
yourFeatureService := yourfeature.NewService(db, cfgManager, otherDeps)

// Register routes
yourfeature.RegisterRoutes(app, yourFeatureService)
```

## Configuration Management

### 1. Accessing Configuration

```go
config := cfgManager.Get()
setting := config.YourFeature.SomeSetting
```

### 2. Configuration Structure

Add your feature configuration to the Config struct in `features/config/config.go`:

```go
type Config struct {
    // existing fields...
    YourFeature YourFeatureConfig `yaml:"yourfeature"`
}

type YourFeatureConfig struct {
    Enabled bool   `yaml:"enabled"`
    Setting string `yaml:"setting"`
}
```

## Database Operations

### 1. Using the Database Service

```go
func (s *Service) SaveItem(item *Item) error {
    query := `INSERT INTO items (name, description) VALUES (?, ?)`
    _, err := s.db.DB.Exec(query, item.Name, item.Description)
    return err
}

func (s *Service) GetItems() ([]Item, error) {
    query := `SELECT id, name, description FROM items`
    rows, err := s.db.DB.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var items []Item
    for rows.Next() {
        var item Item
        if err := rows.Scan(&item.ID, &item.Name, &item.Description); err != nil {
            return nil, err
        }
        items = append(items, item)
    }
    return items, nil
}
```

## Job Integration

For long-running operations, integrate with the job system:

### 1. Create Job Task

```go
package yourfeature

import (
    "github.com/contre95/soulsolid/src/features/jobs"
)

type YourFeatureTask struct {
    service *Service
}

func NewYourFeatureTask(service *Service) *YourFeatureTask {
    return &YourFeatureTask{service: service}
}

func (t *YourFeatureTask) Execute(ctx jobs.JobContext) error {
    // Long-running operation
    ctx.UpdateProgress(50, "Processing...")
    
    result, err := t.service.ProcessLongOperation(ctx.Data)
    if err != nil {
        return err
    }
    
    ctx.UpdateProgress(100, "Complete")
    return nil
}
```

### 2. Register Job Handler

```go
// In main.go
yourFeatureTask := yourfeature.NewYourFeatureTask(yourFeatureService)
jobService.RegisterHandler("yourfeature_task", jobs.NewBaseTaskHandler(yourFeatureTask))
```

### 3. Trigger Job from Handler

```go
func (h *Handler) StartLongOperation(c *fiber.Ctx) error {
    jobID, err := h.jobService.EnqueueJob("yourfeature_task", jobData)
    if err != nil {
        return c.Render("toast/toastErr", fiber.Map{
            "Msg": "Failed to start operation",
        })
    }
    
    return c.Render("toast/toastOk", fiber.Map{
        "Msg": fmt.Sprintf("Operation started (Job: %s)", jobID),
    })
}
```

## Telegram Bot Integration (Optional)

If your feature needs Telegram bot support:

```go
package yourfeature

import (
    "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func RegisterTelegramHandlers(bot *hosting.TelegramBot, service *Service) {
    bot.HandleCommand("yourcommand", func(update tgbotapi.Update) error {
        // Handle command
        return nil
    })
}
```

## Frontend Dependencies

### 1. Adding New Dependencies

Edit `package.json` and run:

```bash
npm install new-package
npm run copy:deps  # Copy to public directory
```

### 2. CSS Customization

Add custom styles to `public/css/input.css` and rebuild:

```bash
npm run build:css
```

## Testing Your Feature

### 1. Development Setup

```bash
npm run dev  # Builds assets and starts Go server
```

### 2. Testing HTMX Interactions

- Use browser developer tools to monitor HTMX requests
- Check Network tab for XHR requests with `HX-Request: true` header
- Verify toast notifications appear correctly

### 3. Testing API Endpoints

```bash
curl -X POST http://localhost:8080/yourfeature/action \
  -H "Content-Type: application/json" \
  -d '{"param": "value"}'
```

## Error Handling Patterns

### 1. Service Layer Errors

```go
var (
    ErrItemNotFound = errors.New("item not found")
    ErrInvalidInput = errors.New("invalid input")
)

func (s *Service) GetItem(id string) (*Item, error) {
    // Validate input
    if id == "" {
        return nil, ErrInvalidInput
    }
    
    // Query database
    item, err := s.queryItem(id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrItemNotFound
        }
        return nil, err
    }
    
    return item, nil
}
```

### 2. Handler Error Responses

```go
item, err := h.service.GetItem(id)
if err != nil {
    if errors.Is(err, yourfeature.ErrItemNotFound) {
        if c.Get("HX-Request") == "true" {
            return c.Render("toast/toastErr", fiber.Map{
                "Msg": "Item not found",
            })
        }
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Item not found",
        })
    }
    
    // Log unexpected errors
    slog.Error("Unexpected error", "error", err)
    if c.Get("HX-Request") == "true" {
        return c.Render("toast/toastErr", fiber.Map{
            "Msg": "Internal server error",
        })
    }
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": "Internal server error",
    })
}
```

## Logging Best Practices

### 1. Structured Logging

```go
import "log/slog"

slog.Debug("Processing request", "itemID", id, "userID", userID)
slog.Info("Operation completed", "duration", time.Since(start))
slog.Warn("Deprecated API used", "endpoint", "/old-endpoint")
slog.Error("Database operation failed", "error", err, "query", query)
```

### 2. Context Information

Include relevant context in log messages:

- Request IDs
- User identifiers
- Resource IDs
- Operation types
- Durations

## Security Considerations

### 1. Input Validation

```go
func validateInput(input string) error {
    if len(input) > 1000 {
        return errors.New("input too long")
    }
    if strings.Contains(input, "<script>") {
        return errors.New("invalid characters")
    }
    return nil
}
```

### 2. SQL Injection Prevention

Always use parameterized queries:

```go
// Good
query := "SELECT * FROM items WHERE name = ?"
rows, err := db.Query(query, name)

// Bad
query := fmt.Sprintf("SELECT * FROM items WHERE name = '%s'", name)
```

## Performance Guidelines

### 1. Database Operations

- Use transactions for multiple operations
- Implement proper indexing
- Consider pagination for large datasets
- Use connection pooling

### 2. HTTP Responses

- Stream large responses
- Implement proper caching headers
- Compress responses when appropriate
- Use HTMX for partial updates

## Deployment Considerations

### 1. Configuration

- All configuration should be externalized
- Support environment variables
- Provide sensible defaults
- Document all configuration options

### 2. Database Migrations

- Version control schema changes
- Provide upgrade and downgrade paths
- Test migrations thoroughly
- Backup before applying migrations

## Code Style Guidelines

### 1. Go Code

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable names
- Keep functions small and focused
- Add package documentation
- Use interfaces for dependencies

### 2. Template Code

- Use consistent indentation
- Include proper HTML escaping
- Follow semantic HTML structure
- Include accessibility attributes

## Checklist for New Features

Before submitting a new feature, ensure:

- [ ] Feature follows the standard directory structure
- [ ] All handlers support both HTMX and API responses
- [ ] Proper error handling is implemented
- [ ] Logging is added for important operations
- [ ] Configuration is externalized
- [ ] Database operations use parameterized queries
- [ ] Templates are responsive and accessible
- [ ] Toast notifications are used for user feedback
- [ ] Long operations use the job system
- [ ] Code is properly documented
- [ ] Tests are written for critical paths
- [ ] Dependencies are updated in package.json
- [ ] CSS is built and committed

## Example: Complete Feature Implementation

See the `downloading` feature for a complete example that demonstrates:
- Service layer with business logic
- Handler layer with HTMX support
- Job integration for long-running operations
- Template structure
- Configuration management
- Error handling patterns

This spec kit should be followed consistently when developing new features to maintain code quality, user experience consistency, and architectural coherence across the SoulSolid application.
