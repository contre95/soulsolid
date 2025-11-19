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

**Reference**: `src/features/downloading/service.go:21-35`
- Service struct with dependencies injected via constructor
- Business logic methods with structured logging
- Clean separation from HTTP concerns

**Example Pattern**: See `src/features/library/service.go:15-25` for service initialization

### 2. Handler Layer (`handlers.go`)

**Reference**: `src/features/downloading/handlers.go:14-26`
- Handler struct with service dependency
- NewHandler constructor pattern
- Dual response support (HTMX + JSON)

**HTMX Detection Pattern**: `src/features/downloading/handlers.go:42-50`
- Check `c.Get("HX-Request") == "true"` for HTMX requests
- Return HTML partials for UI, JSON for API
- Consistent error handling for both response types

**Error Response Pattern**: 
- HTMX errors: `src/features/downloading/handlers.go:43-46`
- API errors: `src/features/downloading/handlers.go:47-50`

### 3. Routes (`routes.go`)

**Reference**: `src/features/downloading/routes.go:8-40`
- RegisterRoutes function with app and service parameters
- API routes grouped under feature prefix
- UI routes under `/ui` group for HTMX partials
- Handler instantiation pattern

## HTMX Integration Patterns

### 1. Detection and Dual Response Handling

**Reference**: `src/features/downloading/handlers.go:42-50`
All handlers must check for HTMX requests and respond appropriately.

### 2. Common HTMX Attributes Used

**Reference**: `views/sections/download.html:7-10`
- `hx-get` / `hx-post`: Make requests
- `hx-target`: Target element for content swap
- `hx-swap`: How to swap content (`innerHTML`, `outerHTML`, etc.)
- `hx-trigger`: When to trigger request (`load`, `click`, etc.)
- `hx-indicator`: Loading indicator element

### 3. Loading States

**Reference**: `views/downloading/album_results.html:14-21`
Use consistent loading patterns with hx-indicator and spinner elements.

### 4. Toast Notifications

**Reference**: `src/features/downloading/handlers.go:57-60`
Use the toast system for user feedback:
- Success: `views/toast/toastOk.html`
- Error: `views/toast/toastErr.html`
- Info: `views/toast/toastInfo.html`

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

**Reference**: `views/partials/main.html:21-40`
The main template uses conditional rendering based on the `.Section` variable.

### 2. Feature Section Template (`views/sections/download.html`)

**Reference**: `views/sections/download.html:1-10`
Create a section template for your feature with proper HTMX loading triggers.

### 3. Partial Templates (`views/downloading/`)

**Reference**: `views/downloading/album_results.html:6-49`
Create reusable partials for dynamic content with proper iteration and fallback handling.

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

**Reference**: `src/main.go:28-44`
Dependencies are injected through the main.go file following this pattern:
- Configuration manager loaded first
- Infrastructure services created next
- Feature services created with dependencies
- Routes registered with services

## Configuration Management

### 1. Accessing Configuration

**Reference**: `src/features/downloading/handlers.go:753-754`
Access config via `cfgManager.Get()` and specific fields.

### 2. Configuration Structure

**Reference**: `src/features/config/config.go:15-95`
Add your feature configuration to Config struct following the existing pattern.

## Database Operations

### 1. Using the Database Service

**Reference**: `src/features/library/service.go:35-50`
Follow the pattern for database operations with proper error handling and resource cleanup.

### 2. Query Patterns

**Reference**: `src/features/library/service.go:67-85`
Use parameterized queries and proper row scanning patterns.

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

### 1. Job Task Creation

**Reference**: `src/features/downloading/download_job.go:12-25`
Create task structs that implement job execution logic.

### 2. Job Registration

**Reference**: `src/main.go:56-66`
Register job handlers in main.go following the existing pattern.

### 3. Job Triggering

**Reference**: `src/features/downloading/handlers.go:343-354`
Trigger jobs from handlers and return appropriate responses.

## Telegram Bot Integration (Optional)

If your feature needs Telegram bot support:

**Reference**: `src/features/downloading/telegram.go:1-20`
Follow the pattern for registering Telegram command handlers.

## Frontend Dependencies

### 1. Adding New Dependencies

**Reference**: `package.json:18-28`
Edit package.json and use npm scripts to manage dependencies.

### 2. Asset Building

**Reference**: `package.json:7-11`
Use the provided npm scripts for building CSS and copying dependencies.

## Testing Your Feature

### 1. Development Setup

**Reference**: `package.json:11`
Use `npm run dev` for development with asset building.

### 2. Testing HTMX Interactions

- Use browser developer tools to monitor HTMX requests
- Check Network tab for XHR requests with `HX-Request: true` header
- Verify toast notifications appear correctly

### 3. Testing API Endpoints

Use curl or similar tools to test JSON endpoints directly.

## Error Handling Patterns

### 1. Service Layer Errors

**Reference**: `src/features/downloading/service.go:100-120`
Define custom error types and handle them consistently.

### 2. Handler Error Responses

**Reference**: `src/features/downloading/handlers.go:64-74`
Handle different error types with appropriate HTTP status codes and user messages.

## Logging Best Practices

### 1. Structured Logging

**Reference**: `src/features/downloading/handlers.go:38`
Use slog with consistent field names and levels.

### 2. Context Information

**Reference**: `src/features/downloading/handlers.go:341`
Include relevant context like IDs, operation types, and durations.

## Security Considerations

### 1. Input Validation

**Reference**: `src/features/downloading/handlers.go:41-50`
Validate all inputs and provide clear error messages.

### 2. SQL Injection Prevention

**Reference**: `src/features/library/service.go:70-75`
Always use parameterized queries with proper variable binding.

## Performance Guidelines

### 1. Database Operations

**Reference**: `src/features/library/service.go:35-50`
Use transactions, proper indexing, and connection pooling.

### 2. HTTP Responses

**Reference**: `src/features/hosting/server.go:88-89`
Configure appropriate body limits and response compression.

## Deployment Considerations

### 1. Configuration

**Reference**: `src/features/config/loader.go:15-30`
Externalize all configuration with environment variable support.

### 2. Database Migrations

Follow the existing database initialization patterns in `src/infra/database/sqlite.go`.

## Code Style Guidelines

### 1. Go Code

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable names
- Keep functions small and focused
- Add package documentation
- Use interfaces for dependencies

### 2. Template Code

**Reference**: `views/downloading/album_results.html:10-48`
- Use consistent indentation
- Include proper HTML escaping
- Follow semantic HTML structure
- Include accessibility attributes

## Middleware Patterns

**Reference**: `src/features/hosting/middleware.go:1-25`
Follow existing middleware patterns for logging, HTMX detection, and request handling.

## Template Functions

**Reference**: `src/features/hosting/server.go:32-77`
Use existing template functions and add new ones following the same pattern.

## Static Asset Management

**Reference**: `src/features/hosting/server.go:113-114`
Follow the pattern for serving static assets and node_modules.

## Checklist for New Features

Before submitting a new feature, ensure:

- [ ] Feature follows the standard directory structure
- [ ] All handlers support both HTMX and API responses (`src/features/downloading/handlers.go:42-50`)
- [ ] Proper error handling is implemented (`src/features/downloading/handlers.go:64-74`)
- [ ] Logging is added for important operations (`src/features/downloading/handlers.go:38`)
- [ ] Configuration is externalized (`src/features/config/config.go:15-95`)
- [ ] Database operations use parameterized queries (`src/features/library/service.go:70-75`)
- [ ] Templates are responsive and accessible (`views/downloading/album_results.html:10-48`)
- [ ] Toast notifications are used for user feedback (`src/features/downloading/handlers.go:57-60`)
- [ ] Long operations use the job system (`src/main.go:56-66`)
- [ ] Code is properly documented
- [ ] Tests are written for critical paths
- [ ] Dependencies are updated in package.json
- [ ] CSS is built and committed

## Example: Complete Feature Implementation

See the `downloading` feature for a complete example that demonstrates:
- Service layer: `src/features/downloading/service.go:21-35`
- Handler layer: `src/features/downloading/handlers.go:14-26`
- HTMX integration: `src/features/downloading/handlers.go:42-50`
- Job integration: `src/features/downloading/download_job.go:12-25`
- Template structure: `views/downloading/album_results.html:6-49`
- Route registration: `src/features/downloading/routes.go:8-40`

This spec kit should be followed consistently when developing new features to maintain code quality, user experience consistency, and architectural coherence across the SoulSolid application.

