# Hypo

An open-source Windows maintenance GUI that drives the [Mole CLI](https://github.com/tw93/Mole)
engine — a community project under GPL-3.0, **not affiliated with the Mole
project**.

Hypo is a dashboard over the engine, not a reimplementation of it. Every
delete, recycle and uninstall runs through the engine's own guarded pipeline
(`Remove-SafeItem` and friends, protected paths, owner guards, the operations
log). Hypo itself contains no deletion logic and never will.

![status: M0](https://img.shields.io/badge/milestone-M0%20scaffold-blue)

## Current state — M0

- Engine locator: finds a Mole engine via `HYPO_ENGINE_ROOT`, an `engine/`
  folder beside the executable, or a Mole checkout beside the Hypo source tree
- `EngineClient`: runs `status` / `history` / `analyze` as hidden subprocesses
  with JSON contracts, timeouts and cancellation; prefers the native Go tools
  (no PowerShell startup cost) and falls back to `mole.ps1`
- Dashboard: health score, CPU / memory / disks / network / battery cards,
  top processes, recent operations from the operations log
- Unit tests for the orchestration layer (run on any OS with fake engine
  executables)

Next milestones live in [docs/ROADMAP.md](docs/ROADMAP.md). The engine-side
changes Hypo needs (for example `clean --json`) are specified in
[docs/ENGINE-CONTRACT.md](docs/ENGINE-CONTRACT.md).

## Safety model

The rules that make this stack trustworthy live in the engine, and Hypo is
built around not bypassing them:

1. **Hypo never deletes.** All mutations go through engine subprocesses.
2. **Recycle-first, always.** The engine recycles and never falls back to
   permanent deletion; neither does Hypo ask it to.
3. **Preview and execute share one code path.** What a dry run reports is what
   the real run does.
4. **The engine re-checks everything at removal time.** A checkbox in the GUI
   is intent, never a list of files to delete.
5. **Everything is logged.** The operations log records each removal; the
   dashboard renders it instead of keeping its own.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full layering.

## Prerequisites

- Go 1.26+
- Node.js 20+ with npm
- A Mole engine (a clone of the [`windows` branch](https://github.com/tw93/Mole/tree/windows)
  works): placed beside this repo, or pointed at by `HYPO_ENGINE_ROOT`
- Wails CLI v3: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- To *run* the app: Windows 10/11 (WebView2 is preinstalled). Building and
  unit-testing works on Linux/macOS.

## Development

```bash
# from the repo root
cd frontend && npm install && npm run build && cd ..

# regenerate TypeScript bindings after changing service methods
GOOS=windows wails3 generate bindings -clean=true -ts -i

# checks (Linux/macOS friendly; the Windows target avoids host webview deps)
gofmt -l . && go vet ./internal/... && go test ./internal/... && GOOS=windows go vet ./...
GOOS=windows go build -o bin/hypo.exe .

# full dev experience on Windows
wails3 dev -config ./build/config.yml
```

With a Mole checkout beside this repo, the locator finds it automatically; the
dashboard then shows live metrics from your machine.

## Trademark

"Mole" and the Mole logo are trademarks of the Mole project. Hypo is an
independent GPL fork ecosystem project: it uses its own name and icon, and
does not imply endorsement by or affiliation with Mole. See the engine's
`TRADEMARK.md`.

## License

GPL-3.0, like the engine. Hypo's UI layer drives the engine as a separate
process; when the two are distributed together, the whole combination is
GPL-3.0.
