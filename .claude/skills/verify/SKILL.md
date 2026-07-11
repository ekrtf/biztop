---
name: verify
description: Build, launch and drive biztop to verify a change end-to-end in the real GUI.
---

# Verifying biztop

The app is a single Go binary serving an embedded frontend (go:embed at build
time, so any html/css/js change requires a rebuild/restart).

## Launch

Run from the repo root (data paths resolve from the working directory):

```
go run ./cmd/biztop
```

Serves http://localhost:5055. The port is hardcoded in `cmd/biztop/main.go`
(`addr = ":5055"`). If 5055 is taken (the user often has their own instance
running, built from old code), temporarily edit `addr` to `:5056`, start the
server, then revert the edit; `go run` compiles at launch so the revert is safe.

Readiness check plus proof the build embeds your frontend change:

```
curl -s http://localhost:5056/static/style.css | grep -c '<something-from-your-diff>'
```

## Drive

Playwright (Python, sync API) with Chromium is installed on this machine.
Views are hash-routed; navigate directly:

- `#/mission`, `#/clients`, `#/objectifs`
- `#/compta/pilotage/<year>`, `#/compta/transactions/<year>`
- `#/fees/<year>`

Gotchas:

- Wait for `networkidle` plus ~600ms for Chart.js (loaded from a CDN, so the
  browser needs network access).
- Do NOT take `full_page=True` screenshots of Transactions: thousands of rows
  make the capture unreadably tall. Use viewport screenshots.
- Useful responsive check: `document.documentElement.scrollWidth -
  document.documentElement.clientWidth` must be 0 (no page-level horizontal
  scroll); wide tables scroll inside their `.table-box`.
- The Objectifs refresh button shells out to the `codex` CLI and can take
  minutes; do not click it during verification.

## Cleanup

`lsof -ti :5056 | xargs kill` (never kill 5055, that is the user's instance).
