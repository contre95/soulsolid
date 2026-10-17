# Contributing

## Commit messages and releases

Releases are managed by [release-please](https://github.com/googleapis/release-please). PRs are squash-merged, so the **PR title** becomes the commit message on `main` and must follow [Conventional Commits](https://www.conventionalcommits.org/), e.g. `fix: library path runtime change`.

release-please keeps a Release PR open with the next version and changelog. Merging it tags the release and publishes the container image.

The version bump is the largest one among the commits since the last release:

| Commit | Bump | Example from 0.27.2 |
|---|---|---|
| `fix:` or `perf:` | patch | 0.27.3 |
| `feat:` | minor | 0.28.0 |
| `feat!:` / `fix!:`, or a `BREAKING CHANGE:` footer | major | 1.0.0 (see below) |
| `chore:`, `docs:`, `refactor:`, `ci:`, `test:`, … | none | no Release PR |
| Commit body line `Release-As: 1.0.0` | that exact version | 1.0.0 |

While the version is below 1.0.0, a breaking change bumps the minor version (0.27.2 → 0.28.0) instead of going to 1.0.0 (`bump-minor-pre-major` in `release-please-config.json`). To release 1.0.0, use `Release-As: 1.0.0`.
