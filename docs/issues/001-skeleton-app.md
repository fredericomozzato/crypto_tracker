# Slice 1 — Skeleton App

## Context

This is the first slice of a greenfield Go TUI project. The repo currently has no Go code — only documentation (PRD, ARCHITECTURE, CLAUDE.md, roadmap). The goal is to establish the project structure, initialize the Go module, and deliver a minimal working Bubble Tea program that renders a placeholder and quits cleanly.

## Scope (from roadmap)

- Project structure: `cmd/crypto-tracker/main.go`, `internal/ui/app.go`
- Bubble Tea program with alt screen, renders a placeholder message
- `q` / `Ctrl+C` quits cleanly with root context cancellation
- **TDD:** model handles `tea.KeyMsg("q")` → returns `tea.Quit`

## Files to create

### 1. `go.mod` + `go.sum`

- `go mod init github.com/fredericomozzato/crypto_tracker`
- Go 1.23
- Dependencies: `charmbracelet/bubbletea` v1, `charmbracelet/lipgloss` v1

### 2. `Makefile`

As specified in CLAUDE.md — `fmt`, `lint`, `test`, `vuln`, `build`, `check` targets.

### 3. `.golangci.yml`

As specified in CLAUDE.md — errcheck, staticcheck, gosimple, gocritic, noctx.

### 4. `cmd/crypto-tracker/main.go`

- Parse `--debug` flag with stdlib `flag` package
- Set up `slog` logger (file or `io.Discard` based on `--debug`)
- Create root `context.Context` with `signal.NotifyContext` for `SIGINT`/`SIGTERM`
- Defer `cancel()`
- Create `AppModel`, wire into `tea.NewProgram` with `tea.WithAltScreen()` and `tea.WithContext(ctx)`
- Run program, `os.Exit(1)` on error

### 5. `internal/ui/app.go`

- `AppModel` struct with `width`, `height int` fields (for future `WindowSizeMsg` handling)
- `NewAppModel() AppModel` constructor
- `Init() tea.Cmd` — returns `nil`
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)`:
  - `tea.KeyMsg`: `q` → return `model, tea.Quit`; `ctrl+c` → return `model, tea.Quit`
  - `tea.WindowSizeMsg`: store width/height
- `View() string` — render `"crypto-tracker — press q to quit"` centered placeholder

## Test plan

### `internal/ui/app_test.go`

Write tests **before** implementation (red-green TDD):

1. **TestNewAppModel** — constructor returns a valid model
2. **TestQuitOnQ** — send `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}` → assert returned cmd produces `tea.QuitMsg`
3. **TestQuitOnCtrlC** — send `tea.KeyMsg{Type: tea.KeyCtrlC}` → assert returned cmd produces `tea.QuitMsg`
4. **TestWindowSizeMsg** — send `tea.WindowSizeMsg{Width: 120, Height: 40}` → assert model stores dimensions
5. **TestViewRendersPlaceholder** — call `View()` → assert output contains `"crypto-tracker"`
6. **TestIgnoresOtherKeys** — send `tea.KeyMsg` for `a`, `b`, etc. → assert no `tea.Quit` returned

### How to assert `tea.Quit`

`tea.Quit` is a `tea.Cmd` (a function). Call it and type-assert the result to `tea.QuitMsg{}`:

```go
cmd := ... // returned from Update
msg := cmd()
_, ok := msg.(tea.QuitMsg)
assert ok
```

## Implementation order

1. `go mod init` + install dependencies
2. Create `Makefile` and `.golangci.yml`
3. Write `internal/ui/app_test.go` (all tests, all red)
4. Write `internal/ui/app.go` (make tests green)
5. Write `cmd/crypto-tracker/main.go`
6. Run `make check` — all must pass

## Verification

```bash
make check          # fmt + lint + test + vuln — must all pass
make build          # produces ./crypto-tracker binary
./crypto-tracker    # shows placeholder, q quits cleanly
./crypto-tracker --debug  # same, but creates log file
```
