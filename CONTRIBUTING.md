# Contributing

## Setup

1. Copy `.env.example` to `.env` and fill `AUTH_TOKEN` and `CLIENT_ID`.
2. Run locally:

```bash
go run ./cmd/api
```

3. Run tests:

```bash
go test ./...
```

## Pull Requests

- Keep PRs focused and small.
- Add or update tests for behavior changes.
- Update `README.md` when API or config changes.
- Ensure `go test ./...` passes before opening a PR.

## Commit Style

Use clear, imperative commit messages, for example:
- `fix: correct docker healthcheck endpoint`
- `test: add config env loading tests`
