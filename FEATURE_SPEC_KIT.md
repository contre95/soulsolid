# SoulSolid Feature Development Spec Kit

This document provides comprehensive guidance for developing new features in the SoulSolid music library management application.

## Architecture Overview

SoulSolid follows a modular, feature-based architecture with clear separation of concerns:

```
src/
├── features/           # Feature modules (business logic)
│   ├── analyze/       # Batch analysis operations (AcoustID, lyrics)
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
├── <some_name>.go    # Feature-specific data structures
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
- Handler struct with only feature-specific service dependencies
- NewHandler constructor pattern with single service parameter
- Dual response support (HTMX + JSON)
- **Important**: Handlers should only receive services from their own feature, not cross-feature dependencies

**HTMX Detection Pattern**: `src/features/downloading/handlers.go:42-50`
- Check `c.Get("HX-Request") == "true"` for HTMX requests
- Return HTML partials for UI, JSON for API
- Consistent error handling for both response types

**Section Rendering Pattern for Sidebar Features** (`Render*Section` methods):
- For sections in sidebar (e.g., analyze_duplicates), always set `data["Section"] = "your_section_name"` and conditionally render full layout:
  ```go
  data := fiber.Map{"Section": "analyze_duplicates"}
  if c.Get("HX-Request") != "true" {
      // Full page for F5/direct navigation (navbar + sidebar via main.html)
      return c.Render("main", data)
  }
  return c.Render("sections/analyze_duplicates", data)
  ```
- Matches `views/partials/main.html` if/else chain. Critical for new analyze sections to appear on refresh.
- Reference: `src/features/duplicates/handlers.go:20-29` (duplicates example).

**Error Response Pattern**: 
- HTMX errors: `src/features/downloading/handlers.go:43-46`
- API errors: `src/features/downloading/handlers.go:47-50`

### 3. Routes (`routes.go`)

**Reference**: `src/features/downloading/routes.go:8-40`
- RegisterRoutes function with app and feature-specific service parameters only
- API routes grouped under feature prefix
- UI routes under `/ui` group for HTMX partials
- Handler instantiation pattern with single service dependency
- **Important**: Route registration should only pass the feature's own service to handlers

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
- Routes registered with feature-specific services only
- **Important**: Handlers should only receive services from their own feature to maintain clean architecture and avoid tight coupling between features

## Configuration Management

### 1. Accessing Configuration

**Reference**: `src/features/downloading/handlers.go:753-754`
Access config via `cfgManager.Get()` and specific fields.

### 2. Configuration Structure

**Reference**: `src/features/config/config.go:15-95`
Add your feature configuration to Config struct following the existing pattern.

### 3. Adding New Configuration Options

When adding new configuration options that are **not configurable from the config_form UI**, follow these guidelines:

#### Configuration Structure
Add your configuration fields to the appropriate struct in `src/features/config/config.go`:

```go
type YourFeature struct {
    NewOption bool `yaml:"new_option"`
    // other fields...
}
```

#### Default Values
**Reference**: `src/features/config/loader.go:97-188`
All new configuration options **must** be added with their default values in the `createDefaultConfig()` function. This ensures that when no `config.yaml` is provided, the application starts with sensible defaults:

```go
YourFeature: YourFeature{
    NewOption: false,  // Default value
    // other fields with defaults...
},
```

#### Examples of Non-UI Configuration
These configuration options are examples of settings that are not configurable from the UI form and should be handled this way:

- `auto_start_watcher` (Import section) - Controls automatic file system watching
- Internal service configurations
- Advanced debugging options
- Performance tuning parameters

#### Important Notes
- **Do not remove or override** existing configuration options that can't be changed at runtime.
- Always provide sensible default values in `createDefaultConfig()`
- Document the purpose and default value in comments
- Consider whether the option truly needs UI access or is better left as an advanced configuration

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

**Reference**: `src/main.go:88-93`
Register job handlers in main.go following the existing pattern:
- Create task instances that implement job execution logic
- Register handlers with jobService.RegisterHandler() for each job type
- Multiple job types can share the same task handler (e.g., download_track, download_album all use DownloadJobTask)
- Job types are strings that match the parameters used in service.StartJob() calls

### 3. Job Triggering

**Reference**: `src/features/downloading/service.go:106-114`
Jobs are triggered through the service layer, not directly from handlers:
- Service methods call `jobService.StartJob()` with job type, title, and parameters
- Handlers call service methods, which return job IDs
- Handlers return appropriate responses (HTMX toast or JSON with job ID)
- **Important**: This maintains clean architecture where handlers don't directly access job services

### 4. Jobs and Queue Together

Jobs and queue work together - see [Queue Integration](#queue-integration) for details:
- Jobs add items to queue when encountering edge cases needing user decision
- Queue items are processed by user actions, not automatically
- Example: Import job finds duplicate → adds to queue → user reviews and decides

## Queue Integration

For items requiring manual user decision, integrate with the queue system:

### 1. Overview: Queue vs Jobs

The queue and job system serve different purposes:

| Aspect | Jobs | Queue |
|--------|------|-------|
| **Purpose** | Automated background processing | Manual user decisions |
| **Trigger** | User initiates, runs automatically | Job encounters edge case |
| **Execution** | Background goroutine | User visits UI and takes action |
| **Example** | "Download album", "Analyze library" | "Duplicate track found", "Lyrics not found" |

**Pattern**: Features use both - jobs do the work, queue captures edge cases needing review.

### 2. QueueItemType Definitions

**Reference**: `src/music/queue.go:13-30`

QueueItemType differentiates what created the queue item and what actions are available:

| QueueItemType | Value | Purpose | Used By |
|---------------|-------|---------|---------|
| `ManualReview` | `"manual_review"` | Track needs manual review before import | importing |
| `Duplicate` | `"duplicate"` | Track is a duplicate of existing track | importing |
| `FailedImport` | `"failed_import"` | Track failed to import | importing |
| `MissingMetadata` | `"missing_metadata"` | Track is missing required metadata | importing |
| `ExistingLyrics` | `"existing_lyrics"` | Track already has lyrics | lyrics |
| `Lyric404` | `"lyric_404"` | Lyrics not found (404) | lyrics |
| `FailedLyrics` | `"failed_lyrics"` | Lyrics fetch failed due to error | lyrics |

### 3. The Action Flow

The queue system follows an event-driven pull pattern:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Job runs       │────▶│  User visits    │────▶│  User submits   │
│  (background)   │     │  /feature/queue  │     │  action         │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                                                │
        ▼                                                ▼
┌─────────────────┐                              ┌─────────────────┐
│  queue.Add()    │                              │  Service.Process│
│  QueueItem      │────▶ (items)                 │  QueueItem()    │
└─────────────────┘                              └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │  queue.Remove() │
                                               │  (item processed)│
                                               └─────────────────┘
```

**Step-by-step**:
1. **Job runs** - Background job processes tracks
2. **Encounters issue** - Finds duplicate, missing metadata, failed, etc.
3. **Adds to queue** - Calls `queue.Add(QueueItem{Type, Track, Metadata})`
4. **User visits queue** - Goes to `/feature/queue` (e.g., `/lyrics/queue`)
5. **User takes action** - POST to `/feature/queue/:id/:action` (e.g., `override`)
6. **Service processes** - `Service.ProcessQueueItem()` executes action
7. **Item removed** - `queue.Remove()` after processing

### 4. Adding Items to Queue

**Reference**: `src/features/importing/directory_job.go:148-165`

From within a job, add items when encountering issues:

```go
func (e *DirectoryImportTask) addTrackToQueue(track *music.Track, queueType music.QueueItemType, jobID string, metadata map[string]string) error {
    if track.ID == "" {
        return fmt.Errorf("track ID cannot be empty")
    }
    item := music.QueueItem{
        ID:        track.ID,
        Type:      queueType,
        Track:     track,
        Timestamp: time.Now(),
        JobID:     jobID,
        Metadata:  metadata,
    }
    return e.service.queue.Add(item)
}
```

**Important**: Only add to queue when user decision is needed - not for normal processing failures.

### 5. Processing Queue Items

**Reference**: `src/features/lyrics/service.go:68-190`

Each feature defines its own processing logic in the service layer:

```go
func (s *Service) ProcessQueueItem(ctx context.Context, itemID string, action string) error {
    item, err := s.queue.GetByID(itemID)
    if err != nil {
        return fmt.Errorf("queue item not found: %w", err)
    }

    switch item.Type {
    case ExistingLyrics:
        switch action {
        case "override":
            // Fetch from another provider
        case "keep_old":
            // Just remove from queue
        }
    case Lyric404:
        switch action {
        case "no_lyrics":
            // Mark track as having no lyrics
        }
    // ... other types
    }

    // After processing, remove from queue
    return s.queue.Remove(itemID)
}
```

The service contains:
1. **Get item** from queue by ID
2. **Switch on QueueItemType** - different types have different valid actions
3. **Switch on action** - execute the appropriate logic
4. **Remove from queue** - item is processed

### 6. Queue Routes

**Reference**: `src/features/lyrics/routes.go:26-31`

Register routes for queue UI and actions:

```go
// Queue routes
queue := app.Group("/lyrics/queue")
queue.Get("/items", handler.RenderLyricsQueueItems)
queue.Get("/items/grouped", handler.RenderGroupedLyricsQueueItems)
queue.Post("/:id/:action", handler.ProcessQueueItem)
queue.Post("/group/:groupType/:groupKey/:action", handler.ProcessQueueGroup)
queue.Post("/clear", handler.ClearQueue)
```

Typical routes:
- `GET /feature/queue/items` - List all items
- `GET /feature/queue/items/grouped` - List items grouped by artist/album
- `POST /feature/queue/:id/:action` - Process single item with action
- `POST /feature/queue/group/:type/:key/:action` - Process group
- `POST /feature/queue/clear` - Clear all items

### 7. Adding Queue to a New Feature

To add queue support to a feature that doesn't have it:

1. **Add queue dependency to service**:
```go
type Service struct {
    // ... other deps
    queue music.Queue
}

func NewService(..., queue music.Queue) *Service {
    return &Service{
        // ... other fields
        queue: queue,
    }
}
```

2. **Add queue to main.go**:
```go
queue := queue.NewInMemoryQueue()
importingService := importing.NewService(..., queue, ...)
```

3. **Add items during job execution** (when encountering edge cases)

4. **Add ProcessQueueItem method** to service with switch on QueueItemType

5. **Add route handlers** for queue display and actions

6. **Create queue UI templates** in `views/feature/`

### 8. Job → Queue Relationship

Jobs and queue work together:

```
Job (background task)
    │
    ├── normal case ──────▶ Job completes successfully
    │
    └── edge case ──────▶ queue.Add(QueueItem) 
                              │
                              ▼
                        User reviews
                              │
                              ▼
                        POST action
                              │
                              ▼
                        Service.ProcessQueueItem()
                              │
                              ▼
                        queue.Remove()
```

**Jobs add items to queue** when encountering:
- Duplicates
- Missing required metadata
- Processing failures that need user decision
- Existing data that might need override

**Jobs do NOT add to queue** for:
- Normal processing (just complete)
- Expected failures (network timeout, etc.) - log and continue
- Items that can be auto-retried

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
- **Important**: Handlers should only depend on services from their own feature to maintain loose coupling and clean architecture

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
- [ ] **Handlers only depend on services from their own feature** (clean architecture)
- [ ] Code is properly documented
- [ ] Tests are written for critical paths
- [ ] Dependencies are updated in package.json
- [ ] CSS is built and committed

## Example: Complete Feature Implementation

See the `downloading` and `analyze` features for complete examples that demonstrate:

### Downloading Feature
- Service layer: `src/features/downloading/service.go:21-35`
- Handler layer: `src/features/downloading/handlers.go:14-26` (with single service dependency)
- HTMX integration: `src/features/downloading/handlers.go:42-50`
- Job integration: `src/features/downloading/download_job.go:12-25`
- Template structure: `views/downloading/album_results.html:6-49`
- Route registration: `src/features/downloading/routes.go:8-40` (with feature-specific service only)
- Clean architecture: handlers only depend on their own feature's service

### Analyze Feature
The `analyze` feature demonstrates batch processing operations on the entire library:
- **Service layer**: `src/features/analyze/service.go:21-30` - Injects multiple services (tagging, lyrics, library, jobs)
- **Handler layer**: `src/features/analyze/handlers.go:14-19` - Single service dependency pattern
- **Job integration**: Multiple job types (`analyze_acoustid`, `analyze_lyrics`) with progress tracking
  - AcoustID analysis: `src/features/analyze/acoustid_job.go:28-134` - Batch fingerprinting and AcoustID lookup
  - Lyrics analysis: `src/features/analyze/lyrics_job.go:29-146` - Batch lyrics fetching with provider selection
- **HTMX integration**: `src/features/analyze/handlers.go:38-45` - Dual response support with toast notifications
- **Template structure**: `views/sections/analyze.html:1-98` - Card-based UI with provider selection
- **Route registration**: `src/features/analyze/routes.go:8-17` - API and UI routes with feature-specific handler
- **Clean architecture**: Maintains separation by only accessing cross-feature services through the service layer

This spec kit should be followed consistently when developing new features to maintain code quality, user experience consistency, and architectural coherence across the SoulSolid application.

