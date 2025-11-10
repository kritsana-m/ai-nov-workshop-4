## Repository instructions for GitHub Copilot

These instructions are specific to the `workshop-4/backend` Go service (Gin + Viper + GORM).
Keep suggestions concise, idiomatic, and safe for existing clients.

1) Preferred code style
- Use idiomatic Go: gofmt/goimports formatting and short, clear identifiers.
- Follow package naming rules: lowercase, no underscores (example: `internal/store`).
- JSON wire-format uses snake_case keys (e.g. `remaining_points`, `idempotency_key`). Keep JSON tags stable to avoid breaking clients.
- Exported symbols must have GoDoc comments. Keep comments short: one-line summary + longer paragraph only if necessary.
- Use `context.Context` in new public APIs; handlers may use Gin context per existing pattern.

2) Things NOT to do
- Do not change JSON field names for existing models without preserving compatibility.
- Avoid package-level goroutines in libraries; callers should control goroutines.
- Do not swallow errors; always return or log with context (use `%w` for wrapping where propagating).
- Don't add direct network calls or credentials in code. Use `config` and environment variables via Viper.
- Avoid large structural reorganizations in a single change. Prefer small, reviewable commits.

3) Project architecture & patterns (what to follow)
- Layout:
  - `cmd/server` — application entrypoint
  - `internal/config` — configuration (Viper)
  - `internal/store`  — DB init (GORM) and package-level getter/setter
  - `internal/models` — GORM models (User, Transfer, PointLedger)
  - `internal/handlers` — HTTP handlers (Gin) and route registration
- Handlers should be thin: parse and validate input, call DB/logic, map results to JSON responses.
- Database changes should use GORM transactions where atomicity is required (see `createTransfer` in `handlers.go`).
- Tests use Gin's test router and an in-memory SQLite instance created via `store.InitDB("file:<name>?mode=memory&cache=shared")` — follow the same pattern in new tests.

4) Error and HTTP conventions
- Use appropriate status codes: 400 for bad JSON, 404 for not found, 409 for conflict (e.g. insufficient points), 422 for unprocessable entity (validate domain rules).
- Return JSON objects with an `error` string on failure, e.g. `{ "error": "not found" }`.

5) Examples & patterns to mirror
- Handler skeleton (follow existing style):
  - Bind JSON to request struct (with `binding` tags), validate early.
  - Use `store.GetDB()` for DB access inside handlers.
  - Use `db.Transaction(func(tx *gorm.DB) error { ... })` for atomic operations.

- Models: keep GORM tags and JSON tags together. Example field:
  - `RemainingPoints int `json:"remaining_points" gorm:"not null"``

6) Testing guidance
- Unit tests should create a fresh in-memory DB per test to avoid cross-test pollution.
- Use `httptest` and Gin's engine to exercise routes; assert JSON responses and DB state.

7) Files to reference when generating code
- `internal/models/models.go` — model definitions and JSON tag conventions
- `internal/handlers/handlers.go` — canonical handler patterns and transfer transaction
- `internal/store/store.go` — DB init & AutoMigrate usage
- `cmd/server/main.go` — startup flow
- `README.md`, `database.md`, `test_api.sh` — docs & manual test expectations

8) When proposing changes
- Prefer focused PRs that add tests and update `database.md` when models change.
- If changing wire-format (JSON fields), include migration notes and backwards-compatible support in the same PR.

If any of these instructions are unclear or you want a different tone (more strict or more permissive), tell me which section to adjust.
