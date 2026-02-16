# AGENTS.md — For AI Agents Working on Pinchtab

## Quick Start

```bash
git config core.hooksPath .githooks   # one-time setup
go build -o pinchtab .                # build
go test ./... -count=1                # test
golangci-lint run ./...               # lint (v2.9.0+)

# Pre-push checks (matches CI exactly)
./scripts/check.sh                    # run all CI checks locally
```

## Project Structure

Single Go package (`main`), no subdirectories. ~1100 lines across 13 files:

| File | Purpose |
|---|---|
| `main.go` | Entry point, Chrome launch, signal handling, route setup |
| `config.go` | Env vars, constants, defaults |
| `bridge.go` | `Bridge` struct, tab registry, stale tab cleanup |
| `handlers.go` | All HTTP handlers (snapshot, navigate, action, tab, etc.) |
| `snapshot.go` | A11y tree types, parsing, `buildSnapshot()`, text formatting |
| `cdp.go` | Low-level CDP helpers (`withElement`, `clickByNodeID`, `navigatePage`) |
| `state.go` | Session save/restore, `markCleanExit` |
| `middleware.go` | Auth + CORS middleware |
| `interfaces.go` | `Browser` and `TabManager` interfaces |
| `*_test.go` | Tests next to the code they cover |

## Conventions

### Go Style
- **`gofmt`** — always. Pre-commit hook enforces it. Run `gofmt -w .` if unsure.
- **Error returns** — always handle. Use `_ =` for intentional ignores (e.g., best-effort `chromedp.Run` for URL/title fetch).
- **Error strings** — lowercase, no punctuation: `fmt.Errorf("page.navigate: %w", err)` not `"Page.Navigate: %v"`.
- **Error wrapping** — use `%w` not `%v` so callers can `errors.Is/As`.
- **No `any` in signatures** — use typed structs or interfaces.

### Testing
- Test files live next to source: `snapshot.go` → `snapshot_test.go`
- Run with `-count=1` to skip cache: `go test ./... -count=1`
- Tests must not require Chrome — mock CDP calls or test pure functions
- Coverage: `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out`

### Git
- Branch: **`main`** only
- Always **rebase before push**: `git pull --rebase && git push`
- **ALWAYS run pre-push checks**: `./scripts/check.sh` before committing
- Pre-commit hook runs: `gofmt` check → `go vet` → `go test`
- If push is rejected, rebase — never force push unless you know why
- Commit messages: imperative, lowercase, concise. `fix lint: unchecked error return`

### Pre-Push Workflow
```bash
# 1. Make your changes
vim some_file.go

# 2. Run comprehensive checks (matches CI exactly)
./scripts/check.sh

# 3. If checks pass, commit and push
git add -A
git commit -m "your changes"
git push
```

**The check script runs:**
- Format check (`gofmt -l .`)
- Go vet (`go vet ./...`)
- Build test (`go build`)  
- All tests with coverage
- Lint (if golangci-lint installed)

This prevents CI failures and ensures code quality.

### CI
CI runs on every push to `main`:
- **Build & Vet** on Go 1.24 + 1.25
- **Format check** (`gofmt -l .`)
- **Tests with coverage**
- **golangci-lint v2.9.0**
- **Release binaries** on version tags (`v*`)

If CI fails, fix it before moving on. Don't stack commits on a broken build.

## API Design

- All endpoints return JSON (`Content-Type: application/json`) except `/screenshot?raw=true` and `/snapshot?format=text`
- Errors: `{"error": "message"}` with appropriate HTTP status
- Use `jsonResp()` and `jsonErr()` helpers — don't write raw responses
- New actions go in the `actionHandlers` map in `handlers.go`
- Request body limit: 1MB (`http.MaxBytesReader`)

## Adding a New Action

1. Add handler func in `handlers.go` matching `actionHandler` signature
2. Register in `actionHandlers` map
3. Add test in `handlers_test.go` or relevant `*_test.go`
4. Update `skill/pinchtab/SKILL.md` with usage example

## What Not To Do

- Don't add dependencies without a strong reason — stdlib first
- Don't create subdirectories/packages — single package keeps it simple
- Don't add MCP, gRPC, or alternative protocols — HTTP is the interface
- Don't skip the pre-commit hook
