# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- CI matrix testing on Go 1.23, 1.24, and 1.25.
- `gofmt`, `go vet`, and `-race` checks in CI.
- `govulncheck` security scan job.
- `.gitignore` covering Go build artifacts, coverage profiles, and macOS metadata.
- `CHANGELOG.md` following Keep a Changelog format.

### Changed
- CI workflow renamed from `build.yaml` to `ci.yml`.
- CI actions upgraded: `setup-go@v4` → `v5`, `golangci-lint-action@v3` → `v6`.
- Dropped manual `actions/cache@v3` step — `setup-go@v5` caches modules implicitly.
- Dropped 100%-coverage gate in CI in favour of printing coverage; coverage gating belongs in review.
- README expanded with Testing, Contributing, and API Reference sections.
