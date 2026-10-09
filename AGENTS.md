# Repository Guidelines

## Project Structure & Module Organization

The Go module is rooted here. `cmd/server/` starts the API; `internal/config/` loads settings; `internal/http/` handles requests; `internal/fit/` and `internal/analysis/` are reserved for parsing and metrics; `internal/store/` manages PostgreSQL. Migrations go in `migrations/`. The React/Vite/TypeScript app is in `frontend/`, with API helpers in `src/api/`, components in `src/components/`, and screens in `src/pages/`. Go tests live beside code as `*_test.go`; React component tests live beside components as `*.test.tsx` and use Vitest. Keep personal FIT files in `samples/` and out of Git.

## Build, Test, and Development Commands

- Copy `.env.example` to `.env`; set `DATABASE_URL` and start PostgreSQL or its SSH tunnel before `go run ./cmd/server`. The server listens on `HTTP_ADDR` or `:3000` by default. `ENV_FILE` can point to a `.env` elsewhere.
- `go build ./...` compiles the API; `go test ./...` runs Go tests.
- `cd frontend && pnpm install && pnpm dev` installs locked dependencies and starts Vite.
- `cd frontend && pnpm test` runs Vitest once; `pnpm test:watch` starts watch mode.
- `cd frontend && pnpm build` type-checks and builds the frontend; `pnpm preview` serves the production build locally.

## Coding Style & Naming Conventions

Run `gofmt` on Go files. Use short, lowercase package names, `PascalCase` for exported identifiers, and `camelCase` for unexported identifiers. Keep HTTP, parsing, analysis, and storage separate. Use two spaces in TypeScript/CSS and follow the existing React function-component style. Write comments and UI text in Korean.

## Testing Guidelines

Vitest, React Testing Library, and jsdom are configured for component tests. Cover file selection/drop, upload success and errors, and accessible UI behavior. Add Go tests for parsing, missing fields, coordinate conversion, upload validation, and HTTP behavior. Use sanitized fixtures. Run `go test ./...`, `cd frontend && pnpm test`, and `pnpm build` when relevant; no coverage threshold is set.

## Commits & Pull Requests

The visible history contains only the initial commit, so no message convention is established. Use concise Korean subjects and keep commits feature-focused (for example, `FIT 업로드 API 구현`). Pull requests should describe changes, list validation results, link issues, and include UI screenshots.

## Configuration, API Scope & Privacy

Keep `.env`, credentials, local databases, and personal FIT files out of version control. Update `.env.example` with placeholders when adding settings. `POST /api/activities` currently accepts a `.fit` file up to 25 MB and returns its name and size; parsing and persistence are not implemented yet. Treat activity records and GPS coordinates as sensitive data.
