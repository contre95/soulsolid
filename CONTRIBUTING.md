# Contributing

## Development setup

See the [Development](README.md#development) section of the README (devenv or manual setup).

## Project layout

| Path | What lives there |
|---|---|
| `src/features/<feature>/` | One folder per feature (importing, downloading, lyrics, …): service, handlers, routes |
| `src/music/` | Domain types and interfaces |
| `src/infra/` | Adapters: database, tag reading/writing, metadata providers, fingerprinting, … |
| `src/hosting/` | HTTP server and middleware |
| `views/` | htmx templates |
| `docs/` | User and feature documentation |

When a change alters how a feature behaves, update its page in `docs/features/<feature>.md`.

Writing a downloader plugin? See [docs/plugins.md](docs/plugins.md).

## Before opening a PR

```bash
go build ./...
go vet ./...
go test ./...
gofmt -w <files you changed>
```

Tests are sparse, so new ones are welcome.

## Trying a change

| Image | How it's built |
|---|---|
| `contre95/soulsolid:feat-<branch>` | Run the **Feature Build** workflow manually on your branch |
| `contre95/soulsolid:dev` | Merging a PR into `dev` |
| `contre95/soulsolid:nightly` | Every day at 02:00 UTC from `main` |

## Commit messages and releases

Releases are managed by [release-please](https://github.com/googleapis/release-please). PRs are squash-merged, so the **PR title** becomes the commit message on `main` and must follow [Conventional Commits](https://www.conventionalcommits.org/), e.g. `fix: library path runtime change`.

Branch names (`fix/…`, `feat/…`) don't affect the version, only the PR title does.

release-please keeps a Release PR open with the next version and changelog. Merging it tags the release and publishes the `vX.Y.Z`, `beta` and `latest` images.

The version bump is the largest one among the commits since the last release:

| Commit | Bump | Example from 0.27.2 |
|---|---|---|
| `fix:` or `perf:` | patch | 0.27.3 |
| `feat:` | minor | 0.28.0 |
| `feat!:` / `fix!:`, or a `BREAKING CHANGE:` footer | major | 1.0.0 (see below) |
| `chore:`, `docs:`, `refactor:`, `ci:`, `test:`, … | none | no Release PR |
| Commit body line `Release-As: 1.0.0` | that exact version | 1.0.0 |

While the version is below 1.0.0, a breaking change bumps the minor version (0.27.2 → 0.28.0) instead of going to 1.0.0 (`bump-minor-pre-major` in `release-please-config.json`). To release 1.0.0, use `Release-As: 1.0.0`.
