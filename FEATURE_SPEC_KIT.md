# SoulSolid Feature Development Spec Kit

This document provides comprehensive guidance for developing new features in the SoulSolid music library management application.

## Architecture Overview

SoulSolid follows a modular, feature-based architecture with clear separation of concerns:

```
src/
├── features/           # Feature modules (business logic)
│   ├── config/        # Configuration management
│   ├── downloading/   # Music download functionality
│   ├── duplicates/    # Duplicate detection (analyze-section feature)
│   ├── hosting/       # HTTP server and middleware
│   ├── importing/     # Library import/processing
│   ├── jobs/          # Background job management
│   ├── library/       # Music library operations
│   ├── logging/       # Logger setup
│   ├── lyrics/        # Lyrics fetching and analysis (analyze-section feature)
│   ├── metadata/      # Tag editing and AcoustID analysis (analyze-section feature)
│   ├── metrics/       # Library metrics
│   ├── playlists/     # Playlist management
│   ├── reorganize/    # File reorganization analysis (analyze-section feature)
│   └── ui/            # UI page rendering
├── infra/             # Infrastructure services
│   ├── database/     # Database layer
│   ├── files/        # File system utilities
│   ├── fingerprint/  # Audio fingerprinting
│   ├── providers/    # External metadata/lyrics providers
│   ├── queue/        # Job queue implementation
│   ├── tag/          # Audio tag reading/writing
│   └── watcher/      # File system monitoring
├── music/            # Core domain models
└── main.go          # Application entry point
```

## Core Technologies

- **Backend**: Go 1.25+ with Fiber web framework
- **Frontend**: HTMX + Hyperscript for dynamic interactions
- **Styling**: TailwindCSS with dark mode support
- **Database**: SQLite (no migration system; tables use `CREATE TABLE IF NOT EXISTS`)
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

**Example**: `downloading.Service` struct and `downloading.NewService()` in `src/features/downloading/service.go`
- Service struct with dependencies injected via constructor
- Business logic methods with structured logging
- Clean separation from HTTP concerns

### 2. Handler Layer (`handlers.go`)

**Example**: `downloading.Handler` struct and `downloading.NewHandler()` in `src/features/downloading/handlers.go`
- Handler struct with only feature-specific service dependencies
- `NewHandler` constructor pattern
- Dual response support (HTMX + JSON)
- **Important**: Handlers should only receive services from their own feature, not cross-feature dependencies

**Cross-feature access via interface**: When a handler truly needs data from another feature, define an interface in the handler's own package and inject it. Example: `lyrics.MetadataService` interface in `src/features/lyrics/handlers.go` exposes only the data the lyrics handler needs from the metadata feature, keeping the coupling explicit and testable.

**Config manager exception**: The config manager (`*config.Manager`) may be injected directly into handlers when runtime config reads are needed — see `reorganize.Handler` in `src/features/reorganize/handlers.go`.

**HTMX Detection Pattern**: See any handler in `src/features/downloading/handlers.go`
- Check `c.Get("HX-Request") == "true"` for HTMX requests
- Return HTML partials for UI, JSON for API
- Consistent error handling for both response types

### 3. Routes (`routes.go`)

**Example**: `downloading.RegisterRoutes()` in `src/features/downloading/routes.go`
- `RegisterRoutes` function receiving app and feature-specific dependencies only
- API routes grouped under feature prefix
- UI routes under `/ui` group for HTMX partials
- **Important**: Route registration should only pass the feature's own service to handlers

## Analyze-Section Features

A group of features share a common pattern: they appear as sub-sections inside the **Analyze** area of the UI, launch background jobs that iterate over the existing library, and optionally surface a queue for items that require a user decision.

Current analyze-section features:

| Feature | Job type | Has queue? | Section key |
|---------|----------|-----------|-------------|
| `lyrics` | `analyze_lyrics` | Yes | `analyze_lyrics` |
| `metadata` | `analyze_acoustid` | No | `analyze_metadata` |
| `reorganize` | `analyze_reorganize` | No | `analyze_files` |
| `duplicates` | `analyze_duplicates` | Yes | `analyze_duplicates` |

All of these follow the same approach:
1. User triggers a job from the UI section
2. Job iterates the library doing its work (fetch lyrics, fingerprint, reorganize files, find duplicates…)
3. Normal outcomes complete silently; edge cases (duplicate found, lyrics already exist, etc.) are added to a queue
4. User visits the queue sub-section to review and take action on each item

The landing `sections/analyze.html` shows a list of all `analyze_`-prefixed jobs via `/ui/jobs/list?prefix=analyze_`. Adding a new analyze-section feature means registering its job type with `analyze_` prefix so it appears there automatically.

## Section Rendering and the HTMX URL-Push Problem

### The Problem

HTMX can push URLs while swapping content into a target `div`. For example, navigating to the lyrics queue swaps the queue content into `#contenido` and pushes `/lyrics/queue` to the browser history. This works fine for in-app navigation.

However, when the user **refreshes** or navigates directly to `/lyrics/queue`, the server returns only the partial HTML that was originally meant for the `div`. There is no `<html>`, no `<head>` (CSS), no sidebar, no navbar — the page looks broken.

### The Fix

Every handler that owns a navigable URL must detect whether the request is a full page load or an HTMX swap, and respond accordingly:

```go
func (h *Handler) RenderAnalyzeLyricsSection(c *fiber.Ctx) error {
    data := fiber.Map{"Section": "analyze_lyrics"}
    if c.Get("HX-Request") != "true" {
        // Direct navigation or F5: render the full shell
        // main.html includes <head>, sidebar, navbar, and uses .Section to pick the right inner template
        return c.Render("main", data)
    }
    // HTMX swap: return only the section partial
    return c.Render("sections/analyze_lyrics", data)
}
```

The `views/partials/main.html` template contains an if/else chain on `.Section` that selects the correct inner template (`views/sections/<name>.html`). **Add a new entry there for every new navigable section.**

Current chain:
```
"dashboard"          → sections/dashboard
"library"            → sections/library
"import"             → sections/import
"jobs"               → sections/jobs
"settings"           → sections/settings
"playlists"          → sections/playlists
"download"           → sections/download
"analyze"            → sections/analyze
"analyze_lyrics"     → sections/analyze_lyrics
"analyze_files"      → sections/analyze_files
"analyze_metadata"   → sections/analyze_metadata
"analyze_duplicates" → sections/analyze_duplicates
```

**Checklist when adding a navigable section**:
1. Set `data["Section"] = "your_section_key"` in the handler
2. Add the `else if eq .Section "your_section_key"` branch in `views/partials/main.html`
3. Create `views/sections/your_section_key.html`
4. In the handler, render `"main"` when `HX-Request` is absent, the section partial otherwise

## HTMX Integration Patterns

### Detection and Dual Response Handling

All handlers must check for HTMX requests and respond appropriately (see any handler in `src/features/downloading/handlers.go`).

### Common HTMX Attributes

- `hx-get` / `hx-post`: Make requests
- `hx-target`: Target element for content swap
- `hx-swap`: How to swap content (`innerHTML`, `outerHTML`, etc.)
- `hx-trigger`: When to trigger request (`load`, `click`, etc.)
- `hx-indicator`: Loading indicator element
- `hx-push-url`: Push a URL to browser history after swap

### Loading States

```html
<div hx-get="/api/data" hx-indicator="#loading">
    <div id="loading" class="htmx-indicator">
        <div class="spinner"></div>
    </div>
</div>
```

### Toast Notifications

Toasts are HTMX responses meant to be swapped into the `#toast-container` div. Only use them in **user action handlers** (create, update, delete, start job, process queue item) — not in data-loading handlers that fill content containers, where the toast would land in the wrong place.

```go
// Success
return c.Render("toast/toastOk", fiber.Map{"Msg": "Operation completed successfully!"})

// Error
return c.Render("toast/toastErr", fiber.Map{"Msg": "Operation failed!"})

// Info
return c.Render("toast/toastInfo", fiber.Map{"Msg": "Processing..."})
```

## Template Structure

### 1. Main Layout (`views/partials/main.html`)

Contains the full HTML shell (head, sidebar, navbar) and the `.Section` if/else chain that picks the inner template. Every navigable section must be registered here.

### 2. Feature Section Template (`views/sections/yourfeature.html`)

```html
<div id="contenido" class="animate__animated animate__fadeIn">
    <h1 class="text-3xl font-bold text-slate-800 dark:text-white mb-8">
        Your Feature
    </h1>
    <div hx-get="/ui/yourfeature/partial" hx-trigger="load">
        Loading...
    </div>
</div>
```

### 3. Partial Templates (`views/yourfeature/`)

```html
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

**Example**: `main()` in `src/main.go`

Dependencies are wired in this order:
1. Configuration manager
2. Infrastructure services (db, queues, tag readers, fingerprint, providers…)
3. Feature services with their dependencies
4. Job handler registration (`jobService.RegisterHandler(jobType, handler)`)
5. Route registration

**Important**: Handlers should only receive services from their own feature to maintain clean architecture.

## Configuration Management

### Accessing Configuration

```go
config := cfgManager.Get()
setting := config.YourFeature.SomeSetting
```

### Configuration Structure

**Example**: `Config` struct in `src/features/config/config.go`

Add feature configuration to the `Config` struct:

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

### Default Values

**Example**: `createDefaultConfig()` in `src/features/config/loader.go`

All new options **must** have defaults in `createDefaultConfig()`:

```go
YourFeature: YourFeatureConfig{
    Enabled: false,
    Setting: "default",
},
```

Options not exposed in the config UI (e.g. `auto_start_watcher`, internal tuning params) still follow this same pattern.

## Database Operations

No migration system exists. Tables are created via `CREATE TABLE IF NOT EXISTS` in `src/infra/database/sqlite.go`. Add new tables there following the existing pattern.

### Query Patterns

**Example**: `library.Service` query methods in `src/features/library/service.go`

```go
func (s *Service) GetItems() ([]Item, error) {
    rows, err := s.db.DB.Query(`SELECT id, name FROM items`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var items []Item
    for rows.Next() {
        var item Item
        if err := rows.Scan(&item.ID, &item.Name); err != nil {
            return nil, err
        }
        items = append(items, item)
    }
    return items, nil
}
```

Always use parameterized queries (`?` placeholders) — never interpolate user input into SQL strings.

## Job Integration

For long-running operations, integrate with the job system:

### Job Task Creation

**Example**: `downloading.DownloadJobTask` in `src/features/downloading/download_job.go`

Create a struct that implements the job execution interface.

### Job Registration

**Example**: job handler registration block in `src/main.go`

```go
myTask := myfeature.NewMyJobTask(myService)
jobService.RegisterHandler("analyze_myfeature", jobs.NewBaseTaskHandler(myTask))
```

Multiple job types can share a handler (e.g. all download variants use `DownloadJobTask`).

### Job Triggering

**Example**: `downloading.Service` job-starting methods in `src/features/downloading/service.go`

Jobs are triggered through the service layer — handlers never call `jobService` directly:

```go
// In service:
jobID, err := s.jobService.StartJob("analyze_myfeature", "My Feature Analysis", params)

// In handler: call service method, return toast or JSON with jobID
```

## Queue Integration

For items requiring a user decision, integrate with the queue system:

### Overview: Queue vs Jobs

| Aspect | Jobs | Queue |
|--------|------|-------|
| **Purpose** | Automated background processing | Manual user decisions |
| **Trigger** | User initiates, runs automatically | Job encounters edge case |
| **Execution** | Background goroutine | User visits UI and takes action |
| **Example** | "Analyze lyrics", "Find duplicates" | "Lyrics already exist", "Duplicate track found" |

### QueueItemType Definitions

**Example**: `music.QueueItemType` constants in `src/music/queue.go`

| QueueItemType | Value | Purpose | Used By |
|---------------|-------|---------|---------|
| `ManualReview` | `"manual_review"` | Track needs manual review before import | importing |
| `Duplicate` | `"duplicate"` | Track is a duplicate of existing track | importing |
| `FailedImport` | `"failed_import"` | Track failed to import | importing |
| `MissingMetadata` | `"missing_metadata"` | Track is missing required metadata | importing |
| `ExistingLyrics` | `"existing_lyrics"` | Track already has lyrics | lyrics |
| `Lyric404` | `"lyric_404"` | Lyrics not found (404) | lyrics |
| `FailedLyrics` | `"failed_lyrics"` | Lyrics fetch failed due to error | lyrics |

### The Action Flow

```
Job runs (background)
    │
    ├── normal case ──────▶ completes silently
    │
    └── edge case ─────▶ queue.Add(QueueItem)
                               │
                         User visits /feature/queue
                               │
                         User POSTs action
                               │
                         Service.ProcessQueueItem()
                               │
                         queue.Remove()
```

### Adding Items to Queue

**Example**: `importing.DirectoryImportTask.addTrackToQueue()` in `src/features/importing/directory_job.go`

```go
item := music.QueueItem{
    ID:        track.ID,
    Type:      music.Duplicate,
    Track:     track,
    Timestamp: time.Now(),
    JobID:     jobID,
    Metadata:  metadata,
}
return s.queue.Add(item)
```

Only add when user decision is required — not for retriable failures or normal processing.

### Processing Queue Items

**Example**: `lyrics.Service.ProcessQueueItem()` in `src/features/lyrics/service.go`

```go
func (s *Service) ProcessQueueItem(ctx context.Context, itemID string, action string) error {
    item, err := s.queue.GetByID(itemID)
    if err != nil {
        return fmt.Errorf("queue item not found: %w", err)
    }
    switch item.Type {
    case music.ExistingLyrics:
        switch action {
        case "override":
            // fetch from another provider
        case "keep_old":
            // no-op, just remove
        }
    }
    return s.queue.Remove(itemID)
}
```

### Queue Routes

**Example**: `lyrics.RegisterRoutes()` queue section in `src/features/lyrics/routes.go`

```go
queue := app.Group("/feature/queue")
queue.Get("/items", handler.RenderQueueItems)
queue.Get("/items/grouped", handler.RenderGroupedQueueItems)
queue.Post("/:id/:action", handler.ProcessQueueItem)
queue.Post("/group/:groupType/:groupKey/:action", handler.ProcessQueueGroup)
queue.Post("/clear", handler.ClearQueue)
queue.Get("/count", handler.QueueCount)
```

### Queue Count Badge

Features with queues expose a `/feature/queue/count` endpoint and wire it to a badge in the sidebar and navbar (see `views/partials/sidebar.html` and `views/partials/navbar.html` for how the importing and lyrics badges are implemented).

### Adding Queue to a New Feature

1. Add `queue music.Queue` to service struct and constructor
2. Create queue in `main.go`: `myQueue := queue.NewInMemoryQueue()`
3. Pass it to `NewService(..., myQueue, ...)`
4. Add `ProcessQueueItem(ctx, id, action)` to service
5. Register queue routes including `/count` endpoint
6. Add sidebar/navbar badge polling `/feature/queue/count`
7. Create queue UI templates in `views/yourfeature/`

## Telegram Bot Integration (Optional)

**Example**: `src/features/downloading/telegram.go`

Follow the pattern for registering Telegram command handlers.

## Frontend Dependencies

**Example**: `package.json` `dependencies` and `scripts` sections

Edit `package.json` and use npm scripts to manage dependencies. Use `npm run dev` for development with live CSS rebuilding.

## Error Handling Patterns

### Service Layer

Define custom error types where needed (see `src/features/downloading/service.go`).

### Handler Error Responses

User action handlers (create, update, delete, start job, process queue item) use toasts:

```go
if err != nil {
    return c.Render("toast/toastErr", fiber.Map{"Msg": err.Error()})
}
```

Data-loading handlers that fill a specific container use `SendString` or JSON so the error lands in the right place rather than the toast container.

## Logging Best Practices

Use `slog` with structured fields throughout (see any handler in `src/features/downloading/handlers.go`):

```go
slog.Info("starting analysis", "trackID", id, "provider", provider)
slog.Error("failed to fetch lyrics", "error", err, "trackID", id)
```

## Security Considerations

- Always use parameterized SQL queries (`?` placeholders, never string interpolation)
- Validate all user inputs before processing

## Middleware and Template Functions

- Middleware patterns: `src/features/hosting/middleware.go`
- Template functions: `hosting.NewServer()` in `src/features/hosting/server.go`
- Static assets: `hosting.NewServer()` static file setup in `src/features/hosting/server.go`

## Checklist for New Features

- [ ] Feature follows the standard directory structure
- [ ] All handlers support both HTMX and API responses
- [ ] If the feature has a navigable URL: section key set, `main.html` updated, dual render pattern implemented
- [ ] Proper error handling: user action errors use `toast/toastErr`, data endpoint errors use `SendString`/JSON
- [ ] Structured logging for important operations
- [ ] Configuration externalized with defaults in `createDefaultConfig()`
- [ ] Database operations use parameterized queries
- [ ] Tables created via `CREATE TABLE IF NOT EXISTS` in `src/infra/database/sqlite.go`
- [ ] Templates are responsive and accessible
- [ ] Toast notifications used for user action feedback
- [ ] Long operations use the job system (triggered via service layer, not handler)
- [ ] If job type starts with `analyze_`: appears automatically in `sections/analyze.html` job list
- [ ] If feature has a queue: `/count` endpoint exists, sidebar/navbar badge wired up
- [ ] Handlers only depend on their own feature's service (cross-feature via local interface)
- [ ] CSS built and committed
