# 1. Technology Stack

- **Status:** Accepted
- **Date:** 2026-08-24
- **Deciders:** contre95 (maintainer)
- **Supersedes:** —
- **Superseded by:** —

## Context

Soulsolid is a self-hosted music library manager: it imports, tags, organizes,
downloads and streams a personal music collection. Its users are hoarders
running the app on a NAS, a home server, a Raspberry Pi or a small VPS —
usually one person per instance, usually via `podman run` or `docker run`.

That deployment target shapes everything. The app has to:

- ship as few moving parts as possible (ideally one process, one image),
- start with zero configuration and generate sane defaults on first run,
- do heavy background work (fingerprinting, downloads, lyric sync, reorganize
  passes) without blocking the UI,
- expose a mobile-friendly web UI plus a Telegram control surface,
- stay cheap on RAM and CPU.

It is also a solo project with a large domain surface (`src/features/` currently
holds config, downloading, importing, jobs, library, logging, lyrics,
merge, metadata, metrics, playlists, reorganize, streaming, ui). Maintenance
budget is the scarcest resource. Every technology choice below is measured
against "can one person keep this alive for years?" more than against
benchmarks or novelty.

This ADR records the foundational stack. Code organization is a separate
concern and is covered in ADR 0002.

## Decision Drivers

- **Single-artifact deployment.** Users should not have to run a database
  container, a Node process and an app process to listen to their music.
- **Zero-config first run.** No credentials, no migrations to run by hand, no
  `.env` mandatory.
- **Low operational surface for the maintainer.** Fewer languages, fewer build
  steps, fewer runtimes to keep patched.
- **Maintainer familiarity over popularity.** A stack the maintainer is fast in
  beats a stack that scores better on paper.
- **Boring, stable, non-trendy.** Preference for tools that follow their own
  path and change slowly, over tools that churn with the ecosystem.
- **Concurrency as a first-class need.** The job system is not an afterthought;
  it's the core of importing and downloading.
- **Backups a user can understand.** Copying a file should be a valid backup
  strategy.

## Decision

### Language: Go (1.25)

Go is the implementation language for everything server-side.

Rationale:

- **Distribution.** `go build` produces one executable. Combined with a
  container image, the install story is a single `podman run`. (See Open
  Questions — this goal is currently only partially met.)
- **Concurrency.** Goroutines and channels map directly onto the job model in
  `src/features/jobs`. Downloads, directory imports and lyric fetches are
  naturally concurrent, and Go makes that ordinary rather than exotic.
- **Familiarity.** The maintainer is productive in Go. On a solo project this
  outweighs most other considerations.
- **Simplicity and consistency.** The syntax is familiar to anyone coming from
  any C-family language, there is largely one way to do a thing, and `gofmt`
  ends style debates. Go does not chase ecosystem trends; it follows its own
  path and changes slowly. For a project intended to be maintained for years by
  one person, that conservatism is the feature.
- **Footprint.** Runs comfortably in the resource envelope of a NAS or Pi.

### Web framework: Fiber v2

HTTP routing, middleware and template rendering go through
`github.com/gofiber/fiber/v2`, with `gofiber/template/html/v2` wrapping Go's
`html/template` (`src/hosting/server.go`).

Rationale:

- **Express-like ergonomics.** The routing and middleware API mirrors the
  Express/PHP-era model the maintainer already thinks in.
- **Batteries included.** Middleware, static file serving and the template
  engine integration come wired. Custom template functions register cleanly
  against the `html` engine.
- **Proven in prior projects.** The maintainer has shipped other applications on
  Fiber; the failure modes are known.
- **Adequate performance.** Built on `fasthttp`; throughput has never been the
  bottleneck.

Explicit caveat: **if Soulsolid had a smaller domain, `net/http` from the
standard library would have been the right call.** The stdlib router is now
capable enough for simple services. Fiber earns its place here because the
number of routes, middlewares and rendering paths is large enough that the
common problems it solves are actually being solved.

### Database: SQLite (`mattn/go-sqlite3`)

The library, job history and metadata live in a single SQLite file
(`src/library.db` in development, `/data` in the container).

Rationale:

- **Zero-config for self-hosters.** No second container, no connection string,
  no user/password provisioning. The app creates the database if it isn't there
  and the user never thinks about it again.
- **Single-file backup.** The entire library state is one file. "Copy this file
  somewhere safe" is a complete, correct backup procedure that a non-technical
  user can follow.

The workload — one person managing one library, with reads vastly outnumbering
writes — sits well inside SQLite's comfort zone, so the usual objections about
write concurrency do not bite.

### Frontend: server-rendered HTML + HTMX

Views are Go `html/template` files under `views/`, enhanced with HTMX 2,
hyperscript, and styled with Tailwind CSS v4. Supporting libraries: jQuery,
SlimSelect, SweetAlert2, ApexCharts, FontAwesome/Iconify, animate.css. All are
vendored into `public/` by an npm script at build time.

Rationale:

- **No JavaScript build pipeline in production.** Assets are copied, not
  bundled. There is no webpack/vite config, no `node_modules` shipped, and no
  JSON API that exists solely to feed a client-side app. Node is a build-time
  dependency for Tailwind only.
- **The server owns state.** There is exactly one source of truth, and it's in
  Go. No client-side store, no cache invalidation layer, no reconciliation
  between a server model and a duplicate client model.
- **Honest assessment of the maintainer's skills.** The maintainer is not a
  frontend developer. The prior background is PHP with server-side rendering and
  jQuery where necessary. Building on that model produces working software
  quickly; building on React/Vue/Svelte would produce a second codebase the
  maintainer is slow in and would eventually neglect.
- **Rejection of the client-state trade-off on principle.** Moving application
  state into the browser was, in the maintainer's view, a bad bargain the
  industry made by default rather than by decision. Soulsolid declines to pay
  that cost. HTMX gets the interactivity benefits of AJAX without importing the
  state-management problem that came bundled with SPA frameworks.

### Supporting choices

| Concern | Choice |
| --- | --- |
| Config | YAML (`gopkg.in/yaml.v3`) with a custom `!env_var` tag for secrets |
| Validation | `go-playground/validator/v10` |
| Logging | `charmbracelet/log` |
| Audio tags | `bogem/id3v2`, `dhowden/tag`, `go-flac/*` |
| File watching | `fsnotify` |
| Images | `nfnt/resize`, `golang.org/x/image` |
| Chat control | `go-telegram-bot-api/v5` |
| Dev environment | devenv / Nix (`devenv.nix`) |
| Packaging | OCI image via `Containerfile` (podman/docker) |

### External binaries as dependencies

Soulsolid shells out to `chromaprint` (`fpcalc`) for acoustic fingerprinting,
and to `flac` / `id3v2` for certain tagging operations.

This is a **deliberate, accepted trade-off**. Reimplementing acoustic
fingerprinting or a complete FLAC toolchain in Go is a multi-year effort in its
own right, and maintaining that code would consume the entire maintenance budget
of the project. No mature pure-Go equivalents exist today, and there is no
indication that will change soon. Until then, invoking battle-tested C tools is
the correct engineering decision, even though it means the runtime image must
carry them and the app must handle subprocess failure paths.

## Alternatives Considered

| Alternative | Why not |
| --- | --- |
| **Rust / C++** | Better raw performance and true static linking, but far slower iteration for this maintainer, and the audio/tagging ecosystem advantage doesn't offset the velocity loss on a solo project. |
| **Python / Node backend** | Rich audio and scraping ecosystems, but requires shipping a runtime and dependency tree to self-hosters — directly opposed to the single-artifact driver. Also worse for the long-running concurrent job model. |
| **`net/http` (stdlib) only** | Genuinely viable and would reduce dependencies. Rejected because the domain is large enough that Fiber's included middleware and template integration save real, repeated work. Would be the preferred choice for a smaller app. |
| **Gin / Echo / chi** | Comparable quality. Fiber won on prior maintainer experience, not on technical superiority. This is acknowledged as a familiarity-driven choice. |
| **PostgreSQL / MySQL** | Better concurrency and tooling, but forces every self-hoster to run and back up a second service. Fails the zero-config and single-file-backup drivers. |
| **React / Vue / Svelte SPA** | Rejected on both capability and principle: it would require a JS build pipeline, a parallel API layer, and duplicated state — in a skill area the maintainer does not have. |
| **Bare `<script>` / vanilla JS** | Would avoid HTMX, but reintroduces hand-rolled fetch-and-patch DOM code, which is exactly what HTMX exists to eliminate. |
| **Pure-Go audio fingerprinting** | Does not exist at usable maturity. Writing it is out of scope. |

## Consequences

### Positive

- Deployment is one container and a handful of volume mounts. No database
  service, no reverse-proxy-mandatory setup, no runtime install.
- Backup and migration are file operations a user already understands.
- Adding a feature touches Go and HTML templates only — one mental model, one
  language, one place where state lives.
- Background work is expressible directly with goroutines rather than an
  external queue or worker service.
- Low memory and CPU floor; the app is realistic on constrained hardware.
- The stack changes slowly, so maintenance is dominated by Soulsolid's own
  domain rather than by dependency churn.

### Negative / accepted costs

- **CGO is required.** `mattn/go-sqlite3` needs CGO, and the container sets
  `CGO_ENABLED=2`. Cross-compilation is therefore not a simple `GOOS`/`GOARCH`
  matrix.
- **The runtime image is fatter than it should be.** Because Go's `plugin`
  package is used for downloader plugins (`go build -buildmode=plugin`), the
  runtime stage still carries the Go toolchain, `gcc` and `libc-dev` so plugins
  can be compiled against the running binary. The image is not the minimal
  scratch/distroless artifact a Go app would normally produce.
- **Runtime binary dependencies.** `chromaprint`, and where used `flac` and
  `id3v2`, must be present in the image. Fingerprinting silently degrades if
  they are missing, so error handling around subprocess invocation matters.
- **Node is required at build time.** Tailwind v4 and the vendored JS assets are
  produced by npm scripts (`npm run build:assets`). Production does not need
  Node, but contributors and CI do.
- **jQuery coexists with HTMX.** Both are loaded. This is historical overlap, not
  design, and is a candidate for cleanup — though not urgent.
- **SQLite write concurrency is a ceiling.** Acceptable for the single-user
  target, but a multi-tenant or heavily-parallel-write future would require
  revisiting the storage decision.
- **Frontend ambition is bounded.** Genuinely rich client interactions (complex
  drag-and-drop editors, offline mode, real-time waveform editing) would be
  awkward here. That constraint is accepted knowingly.

## Open Questions

1. **CGO and the static binary story.** The stated driver of "one static binary"
   is currently not met: `mattn/go-sqlite3` requires CGO, and the `plugin`
   package requires dynamic linking regardless. Two threads worth pulling:
   - Migrating to `modernc.org/sqlite` (pure Go) would remove the SQLite half of
     the CGO requirement and simplify cross-compilation, at the cost of some
     performance and a different maturity profile.
   - The `plugin`-based extension mechanism is the harder blocker. As long as it
     stays, CGO and a toolchain-bearing runtime image stay with it. An
     alternative plugin transport (subprocess + IPC, WASM, or an HTTP contract)
     would decouple this.

   These are deliberately **not** resolved here and should get their own ADR.

2. **jQuery removal.** Whether the remaining jQuery usage is worth extracting in
   favour of HTMX + hyperscript alone.

3. **Runtime image slimming.** Whether the plugin build path can be moved
   entirely to a separate builder image, allowing the runtime stage to drop the
   Go toolchain and compiler.

## Related

- ADR 0002 — Vertical slice architecture
- ADR 0003 — Background job engine
- `docs/features/hosting.md` — HTTP server and middleware
- `docs/features/jobs.md` — background job system
- `docs/plugins.md` — plugin mechanism
