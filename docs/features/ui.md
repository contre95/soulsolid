---
weight: 130
title: "UI / Dashboard"
description: "The dashboard landing page and shared UI entry points."
icon: "dashboard"
draft: false
toc: true
---

The **ui** feature provides the application's landing surface — the dashboard — and a
couple of shared section entry points that don't belong to any single domain feature.
It is intentionally tiny: it renders top-level pages and a dashboard card, leaning on
the [hosting](./hosting.md) `respond` helpers and the Go templates for everything
else. There is no business logic and no persistence here.

## What it does

- Renders the **Dashboard** (the app's home page, served at both `/` and `/dashboard`).
- Renders the **"All Analyze Jobs"** section — the hub that links to the individual
  analyze flows (lyrics, reorganize, metadata, duplicates).
- Serves the **Quick Actions** dashboard card partial.

## How it works

```
routes.go    → registers /, /dashboard, /analyze, /dashboard/quick-actions
handlers.go  → renders sections + the quick-actions card
```

- **`Handler`** (`handlers.go`) is constructed with the `config.Manager` (so future
  cards can surface config-derived data) but currently just renders templates via the
  `respond` package.
- **`RenderDashboard`** renders the `dashboard` section; because it uses
  `respond.Section`, a normal browser navigation gets the full `main` layout while an
  HTMX navigation gets just the section fragment.
- **`RenderAnalyzeSection`** renders the `analyze` section — the umbrella page for the
  analyze-style jobs. The actual analyze sub-sections (e.g. `/analyze/lyrics`,
  `/analyze/files`) are registered by their own features.
- **`GetQuickActionsCard`** returns the `cards/quick_actions` partial used on the
  dashboard.

## Endpoints

Registered in `src/features/ui/routes.go`.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/` | Dashboard (home) | Section |
| GET | `/dashboard` | Dashboard (alias) | Section |
| GET | `/analyze` | "All Analyze Jobs" hub | Section |
| GET | `/dashboard/quick-actions` | Quick-actions card | Partial |

> The analyze hub links out to feature-owned routes such as `/analyze/lyrics`
> ([lyrics](./lyrics.md)), `/analyze/files` ([reorganize](./reorganize.md)), and the
> metadata/duplicates analyze flows — those routes live in their respective features.

## Configuration

The UI feature has no config block of its own. It receives the `config.Manager` for
consistency and potential future use but reads no settings directly.

## Related

- [Hosting](./hosting.md) — the server, templates, and `respond` helpers this feature renders through.
- [Lyrics](./lyrics.md), [Reorganize](./reorganize.md), [Metadata](./metadata.md) — the analyze flows the hub links to.
- [Metrics](./metrics.md) — dashboard cards surface library stats.
