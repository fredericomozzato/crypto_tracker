---
status: done
branch: feat/010-docs-tooling-cleanup
---

# Slice 10 — Docs & tooling cleanup

## Context

Slices 1–9 are complete. The app is fully functional with markets, portfolio management, and all CRUD operations. Slice 10 is a housekeeping pass with no runtime changes: it removes two stale requirements from `CLAUDE.md` and hardens the Makefile lint target.

---

## Scope

Three targeted changes, no code modifications:

1. Remove the 100×30 terminal size guard requirement from `CLAUDE.md`
2. Remove the `gosimple` linter mention from `CLAUDE.md`
3. Fix the Makefile `lint` target to fail hard on golangci-lint config errors

---

## Files to modify

### 1. `CLAUDE.md` — Remove terminal size guard

**Current text in the "Program setup" section:**

```
Every model must handle `tea.WindowSizeMsg`. If the terminal is below 100 columns × 30 rows, all content is hidden and a single centered message is rendered: `"Terminal too small — resize to at least 100×30"`.
```

**Change:** Remove the second sentence. The `tea.WindowSizeMsg` handler requirement stays (it's still needed for layout). Only the 100×30 guard rule goes away.

The resulting paragraph becomes:

```
Every model must handle `tea.WindowSizeMsg`.
```

---

### 2. `CLAUDE.md` — Remove `gosimple` linter

The CLAUDE.md has two places that mention `gosimple`:

**Active linters list:**
```
- `gosimple` — suggests simpler constructs
```
→ Remove this line entirely.

**`golangci-lint config` example block:**
```yaml
linters:
  enable:
    - errcheck
    - staticcheck
    - gosimple       ← remove
    - gocritic
    - noctx
```
→ Remove the `- gosimple` line from the example.

**Why:** `gosimple` was merged into `staticcheck` in golangci-lint v2. The actual `.golangci.yml` already does not include it. The CLAUDE.md docs are stale and mislead agents into believing it's an active linter.

---

### 3. `Makefile` — Fail hard on golangci-lint config errors

**Current `lint` target:**
```makefile
lint:
	gofumpt -l . | grep . && exit 1 || true
	golangci-lint run ./...
	govulncheck ./...
```

**Issue:** `golangci-lint run ./...` can silently succeed even when the config file has errors (it may fall back to defaults). This masks misconfiguration.

**Fix:** Prepend a `golangci-lint config verify` call, which explicitly validates the config and exits non-zero on any error:

```makefile
lint:
	gofumpt -l . | grep . && exit 1 || true
	golangci-lint config verify
	golangci-lint run ./...
	govulncheck ./...
```

---

## Implementation order

1. Edit `CLAUDE.md` — remove terminal size guard sentence (Program setup section)
2. Edit `CLAUDE.md` — remove `gosimple` from Active linters list
3. Edit `CLAUDE.md` — remove `gosimple` from golangci-lint config example
4. Edit `Makefile` — add `golangci-lint config verify` line

---

## Verification

```bash
# Confirm gosimple is gone from CLAUDE.md
grep -n gosimple CLAUDE.md   # → no results

# Confirm terminal size guard is gone from CLAUDE.md
grep -n "100×30" CLAUDE.md   # → no results

# Confirm Makefile has config verify
grep -n "config verify" Makefile   # → 1 result

# Confirm lint target still works
make lint
```

No new tests required. Existing suite (`make test`) must stay green.
