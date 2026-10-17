---
weight: 90
title: "Metrics"
description: "Library analytics — genre, year, format, lyrics, and metadata-completeness charts."
icon: "monitoring"
draft: false
toc: true
---

The **metrics** feature turns the library into dashboards. It computes aggregate
statistics about the collection — genre distribution, release-year spread, audio
format breakdown, lyrics coverage, and metadata completeness — and renders them as
[ApexCharts](https://apexcharts.com/) on the metrics page. Because computing these
aggregates can be expensive on a large library, the heavy work runs as a background
job that **caches** results in the database; the page then reads the cached values.

## What it does

- Runs a **`metrics`** background job that calculates and **caches** all aggregates.
- Serves a **metrics overview** partial with headline counts (tracks, artists, albums).
- Serves individual **chart fragments** (genre treemap, year bars, format pie,
  metadata horizontal bars) as ApexCharts-ready data.
- Computes **metadata completeness** on the fly (ISRC/BPM/Year/Genre/Lyrics coverage
  as percentages of total tracks).

## How it works

The feature splits cleanly into a cached read path (handlers + service) and a
write/compute path (the job task):

```
routes.go        → registers /metrics chart + overview routes
handlers.go      → HTTP layer: read cached metrics, shape ApexChart fragments
service.go       → reads cached metrics from the store into MetricsData
metrics.go       → LibraryMetrics interface + stat structs
metrics_job.go   → the `metrics` job that computes + caches everything
apex_charts.go   → converts MetricsData into ApexChartData (labels/series/colors)
```

- **`LibraryMetrics`** (`metrics.go`) is the persistence/analytics interface
  implemented by the SQLite store. It provides both the *computation* methods
  (`GetGenreDistribution`, `GetYearDistribution`, `GetLyricsStats`, etc.) and the
  *cache* methods (`StoreMetric`, `GetStoredMetrics`, `ClearStoredMetrics`).
- **`Service`** (`service.go`) reads the cached values into a `MetricsData` struct.
  It is resilient: a failure to read any single metric is logged as a warning and
  leaves that slice empty rather than failing the whole page.
- **`apex_charts.go`** converts `MetricsData` into `ApexChartData` (parallel
  `Labels`, `Series`, and `Colors` slices) for each chart, including a fixed color
  palette and, for the year chart, filtering to years ≥ 1900 and sorting ascending.

### The `metrics` job

`metrics_job.go` (`MetricsCalculationTask`) is what makes the dashboards current. It:

1. **Clears** all stored metrics.
2. Calculates and stores each aggregate in turn, reporting progress at fixed
   checkpoints:
   - genre distribution (20%)
   - lyrics stats (40%)
   - metadata completeness (60%)
   - format distribution (80%)
   - year distribution (95%)
3. Stores each data point via `StoreMetric(type, key, value)` keyed by a metric type
   (`genre_counts`, `lyrics_stats`, `metadata_completeness`, `format_distribution`,
   `year_distribution`).

The page then reads these cached rows, so re-rendering charts is cheap. Run the job
again whenever the library changes enough to want fresh numbers.

### Metadata-completeness chart

`GetMetadataChartHTML` is computed **live** rather than from the cache. It fetches the
total track count plus the counts of tracks with ISRC, valid BPM, valid year, valid
genre, and lyrics, then expresses each as a **percentage of total tracks** for a
horizontal-bar chart. If the library is empty it renders an empty chart.

## Endpoints

Registered in `src/features/metrics/routes.go` under the `/metrics` group.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/metrics/overview` | Headline counts + metrics overview | Partial |
| GET | `/metrics/charts/genre` | Genre distribution (treemap) | Partial (ApexChart) |
| GET | `/metrics/charts/year` | Release-year distribution (vertical bars) | Partial (ApexChart) |
| GET | `/metrics/charts/format` | Audio-format distribution (pie) | Partial (ApexChart) |
| GET | `/metrics/charts/metadata` | Metadata completeness % (horizontal bars) | Partial (ApexChart) |

> The `metrics` job itself is started through the [jobs](./jobs.md) system
> (`POST /jobs/start/metrics`), not through a route in this feature.

## Configuration

Metrics has no dedicated config block. It depends on the SQLite store (which holds
both the source data and the cached metric rows) and the `config.Manager`.

## Related

- [Jobs](./jobs.md) — runs the `metrics` calculation job.
- [Library](./library.md) — the source data the metrics summarize.
- [Lyrics](./lyrics.md) and [Metadata](./metadata.md) — drive the lyrics-coverage and completeness charts.
