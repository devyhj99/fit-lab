# Repository Guidelines

## Project Structure & Module Organization

The Go module lives at the repository root. `cmd/server/` contains the API entry point; `internal/http/`, `fit/`, `analysis/`, and `store/` hold HTTP handling, FIT parsing, calculations, and persistence. Keep PostgreSQL migration files in `migrations/`. The React/Vite/TypeScript app is under `frontend/`, with API helpers in `src/api/`, reusable UI in `src/components/`, and screens in `src/pages/`. There are no test files yet. Put Go tests beside the code they cover (`*_test.go`) and frontend tests beside the relevant components when a test framework is introduced. Use `samples/` only for local FIT data; never commit personal activity files.

## Build, Test, and Development Commands

- `go run ./cmd/server` starts the API on `:3000` by default. Copy `.env.example` to `.env` for local settings; `HTTP_ADDR` is supported, while `DATABASE_URL` awaits database integration.
- `go build ./...` compiles all Go packages; `go test ./...` runs Go tests.
- `cd frontend && npm install && npm run dev` installs dependencies and starts the Vite server.
- `cd frontend && npm run build` type-checks and builds the production frontend; `npm run preview` serves that build locally.

## Coding Style & Naming Conventions

Run `gofmt` on Go files. Use short, lowercase Go package names, `PascalCase` for exported identifiers, and `camelCase` for unexported identifiers. Keep HTTP, parsing, analysis, and persistence concerns in their respective packages. Use two spaces in TypeScript/CSS and follow the existing React function-component style. Keep user-facing UI text and code comments in Korean, consistent with the current project.

## Testing Guidelines

Add focused tests for parsing, missing FIT fields, coordinate/unit conversion, analysis calculations, and HTTP behavior. Use synthetic or sanitized fixtures. Run `go test ./...` and `npm run build` before submitting changes; no coverage threshold or frontend test runner is configured yet.

## Commits & Pull Requests

Git history is unavailable in this checkout, so repository-specific commit conventions cannot be verified. Use concise imperative subjects, such as `Add FIT coordinate conversion`. A pull request should describe the change, note validation commands and results, link related issues, and include screenshots for UI changes.

## Configuration & Privacy

Keep `.env`, credentials, local databases, and personal FIT files out of version control. Update `.env.example` when adding configuration, using placeholder values only. Treat GPS coordinates and activity records as sensitive data.
