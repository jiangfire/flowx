# AGENTS.md

## Quick reference

```bash
go test ./... -count=1                        # all tests (640+ cases, SQLite in-memory)
go test ./internal/application/bpmn/ -v -run TestName  # single test
go test ./tests/e2e/... -v                    # E2E integration tests
gofmt -w . && go vet ./...                    # format + static check (run before commit)
go build ./cmd/server                         # verify compilation
go run ./cmd/server                           # simple start (SQLite in-memory, no PG/Redis needed)
swag init -g cmd/server/main.go --parseDependency --parseInternal  # regenerate swagger docs

# PostgreSQL integration tests (local)
docker compose -f docker-compose.test.yml up -d
TEST_DB_DRIVER=postgres TEST_PG_DSN="host=localhost port=5432 user=flowx password=flowx123 dbname=flowx sslmode=disable" go test -race -v ./internal/infrastructure/persistence/... ./tests/e2e/...
docker compose -f docker-compose.test.yml down
```

Simple start: set `database.driver: "sqlite"` in `config.yaml` — no PostgreSQL or Redis required. SQLite uses `github.com/glebarez/sqlite` (pure Go, no CGO). If Redis is unavailable the server warns and continues.

CI runs (SQLite): `go vet` → `golangci-lint` → `gosec` → `go test -race` → `go build`.
CI runs (PostgreSQL): parallel `integration-postgres` job with `postgres:16-alpine` service, runs persistence + E2E tests. Match this locally before pushing.

## Architecture

Clean architecture, single entrypoint `cmd/server/main.go` → `internal/app/container.go` (DI container wiring all services).

```
cmd/server/main.go          # entrypoint, config, graceful shutdown
internal/app/container.go   # DI container, all handler/service initialization
internal/
  application/              # service layer (business logic)
  domain/                   # models + interfaces (ports)
  infrastructure/persistence/  # GORM repositories
  interfaces/http/handler/  # Gin HTTP handlers
  interfaces/mcp/           # MCP protocol server
pkg/                        # shared libs (transaction, errors, response, pagination)
tests/e2e/                  # integration tests with real services + SQLite
```

Key wiring: `container.go` creates repos from `*gorm.DB`, injects into services, services into handlers. To add a new module: add model in `domain/`, repo in `persistence/`, service in `application/`, handler in `handler/`, register in `container.go` + `router.go`.

Additional packages under `application/`: `ai/` (LLM service with retry + health check), `datagov/expression/` (expression evaluator used by BPMN engine).

## Domain models

All models embed `base.BaseModel` which provides `ID` (string UUID), `TenantID`, `CreatedAt`, `UpdatedAt`, `DeletedAt` (soft delete). Use `base.GenerateUUID()` for new IDs. `base.JSON` is a `map[string]any` type with GORM `jsonb` support.

## Transactions

Import/bulk operations use all-or-nothing transactions via `pkg/transaction`:

```go
err := transaction.WithTransaction(ctx, s.db, func(txCtx context.Context) error {
    if err := s.repo.Create(txCtx, item); err != nil {
        return err  // triggers rollback
    }
    return nil  // triggers commit
})
```

Inside repos, always use `DBFromContext(ctx, r.db).WithContext(ctx)` instead of `r.db.WithContext(ctx)` — this picks up the transaction from context. The persistence package re-exports `DBFromContext` from `pkg/transaction` to ensure cross-package key consistency.

## Tenant isolation

Repository queries must always include `WHERE tenant_id = ?`. `GetByID` takes explicit `tenantID` parameter and filters by both `id` and `tenant_id`. Never query by ID alone.

## Testing

Tests use SQLite in-memory by default. Shared helper `internal/testutil.SetupTestDB(t, tables...)` handles driver selection and auto-migration. For transaction tests use `SetupTestDBShared` (shared-cache SQLite).

Multi-driver: set `TEST_DB_DRIVER=postgres` + `TEST_PG_DSN` to run converted tests against PostgreSQL. Each call creates an isolated schema dropped in `t.Cleanup`.

Pattern: `setupXxxService(t)` returns `(service, *gorm.DB)` — use the `*gorm.DB` to verify DB state after operations.

Import rollback tests: pre-insert conflicting records via `db.Create()`, attempt import, verify count matches expected (pre-inserted survives, imported rolled back).

E2E tests in `tests/e2e/` use real services (no mocks) with `processDefinitionTable` local struct for the `process_definitions` table.

## BPMN engine

Runtime state is in-memory (`Engine`). Gateway state (`joinReceived`, `inclusiveTokens`) is persisted to DB via `persistInstanceState` and restored on startup via `RestoreRunningInstances`. `CompleteTask` auto-restores instances not in engine memory.

BPMN conditions use `tool.config.xxx` path syntax (e.g., `tool.config.amount > 5000`), not bare variable names.

## Swagger docs

Handlers use swag annotations (`// @Summary`, `// @Router`, etc.). After changing handler signatures or adding endpoints, run `swag init -g cmd/server/main.go --parseDependency --parseInternal`. Generated files in `docs/` (`docs.go`, `swagger.json`, `swagger.yaml`) are committed.

## golangci-lint quirks

- `S1009`: Don't nil-check before `len()` — `len(nilMap)` is 0 in Go
- `ST1023`: Omit redundant type in `var x Type = value` when type is inferrable

## Config

`config.example.yaml` loaded via Viper. Supports env vars with `FLOWX_` prefix (e.g., `FLOWX_DATABASE_HOST`). Health endpoints: `/api/v1/health` (liveness), `/api/v1/ready` (readiness with DB ping + LLM check).

Key config options: `database.driver` (postgres/sqlite), `database.path` (SQLite file path), `llm.model` (wired into OpenAI API call), `webhook.url` + `webhook.timeout_sec` (notification webhook).

## Go version

1.26.3 — CI and release workflows use `go-version: "1.26"`. Update `go.mod`, `.github/workflows/ci.yml`, and `.github/workflows/release.yml` together.
