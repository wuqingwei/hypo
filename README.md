# Hypo

An open-source Windows maintenance GUI that drives the [Mole CLI](https://github.com/tw93/Mole)
engine — a community project under GPL-3.0, **not affiliated with the Mole
project**.

Hypo is a dashboard over the engine, not a reimplementation of it. Every
delete, recycle and uninstall runs through the engine's own guarded pipeline
(`Remove-SafeItem` and friends, protected paths, owner guards, the operations
log). Hypo itself contains no deletion logic and never will.

![status: M1](https://img.shields.io/badge/milestone-M1%20engine%20surface-green)

## What works today

Three pages, all verified on real Windows 10/11 hardware:

- **Dashboard** — live health score, CPU (per-core), memory, disks, network,
  battery and top processes from the engine's `status --json`, polling every
  5 s while the window is visible; recent operations rendered from the
  engine's operations log.
- **Cleanup** — a full preview → confirm → execute → reconcile loop over the
  engine's `clean` command. The preview is the engine's own dry run
  (`clean --dry-run --json`); execution re-runs every engine guard and the
  result view compares what was freed against what was previewed, with the
  engine's skip reasons per row.
- **Uninstall** — the engine's application inventory (searchable, with sizes
  and sources), multi-select, an explicit confirmation, and a per-row results
  view (`uninstalled` / `skipped` / `failed` with reasons, charge-limit
  notices, incomplete badge). Uninstallers run exactly as registered, once,
  with the engine's own reread-before-launch and 30-minute ceiling.

## Safety model

The rules that make this stack trustworthy live in the engine, and Hypo is
built around not bypassing them:

1. **Hypo never deletes.** All mutations go through engine subprocesses.
2. **Recycle-first, always.** The engine recycles and never falls back to
   permanent deletion; neither does Hypo ask it to.
3. **Preview and execute share one code path.** What a dry run reports is
   what the real run does — the engine's JSON records are captured at the
   same points as its summary counters, so parity is by construction.
4. **The engine re-checks everything at removal time.** A checkbox in the GUI
   is intent, never a list of files to delete; between preview and execute
   the engine re-enumerates and re-runs every guard.
5. **Everything is logged.** The operations log records each removal; the
   dashboard renders it instead of keeping its own.
6. **Structured output fails loud.** Any unexpected byte on a JSON pipe — a
   leaked interactive row, vendor uninstaller output — surfaces in the error
   message with the stdout head, rather than being parsed anyway.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the layering and
[docs/ENGINE-CONTRACT.md](docs/ENGINE-CONTRACT.md) for every JSON contract,
including the engine-side features this project added (`clean --json`,
`uninstall --select`).

## The engine

Hypo drives an unmodified [Mole CLI](https://github.com/tw93/Mole) tree
(Windows branch) located via, in order:

1. `HYPO_ENGINE_ROOT` — an explicit path (an invalid value is an error, not a
   silent skip)
2. an `engine\` folder beside the Hypo executable — how the deploy package
   ships it
3. a Mole checkout beside the Hypo source tree — development layout

A candidate counts as an engine only when both `mole.ps1` and
`lib/core/base.ps1` exist; a tree without the safety core is refused. Native
Go tools (`bin/status.exe`, `bin/analyze.exe`) are used directly when present
— no PowerShell startup cost — with `mole.ps1` as the self-healing fallback.

Two CLI features were added to the engine for this project (GPL-3.0, on the
`feat/clean-json` branch, upstreamable as pure scriptability improvements):

- `mo clean [--dry-run] --json` — one JSON envelope for the whole run
- `mo uninstall --select <id[,id...]> [--yes] [--json]` — headless uninstall
  keyed by the ids `mo uninstall --list` already publishes

## Building and running

Prerequisites: Go 1.26+, Node 20+ with npm, the Wails v3 CLI
(`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`), and a Mole
engine tree (see above). Running the app needs Windows 10/11 (WebView2 ships
with the OS); building and unit-testing work on Linux/macOS.

```bash
# one-command Windows deploy package (frontend + exe + engine runtime + zip)
pwsh -NoProfile -File scripts/package.ps1

# regenerate TypeScript bindings after changing service methods
GOOS=windows wails3 generate bindings -clean=true -ts -i

# checks
gofmt -l . && go vet ./internal/... && go test ./internal/... && GOOS=windows go vet ./...
cd frontend && npm install && npm run build

# run in dev on Windows
wails3 dev -config ./build/config.yml
```

The engine's prebuilt native tools cross-compile from the Mole checkout:

```bash
cd ../Mole
GOOS=windows go build -o bin/status.exe ./cmd/status
GOOS=windows go build -o bin/analyze.exe ./cmd/analyze
```

The orchestrator's unit tests run on any OS with fake engine executables; CI
(`.github/workflows/check.yml`) runs fmt, vet, tests, the frontend build and
a Windows cross-build on every push.

## Roadmap

Done and next steps live in [docs/ROADMAP.md](docs/ROADMAP.md). In short:
M1 (engine headless surface + the three pages above) is complete; next
candidates are the analyze treemap page, history and whitelist UIs, a full
Windows PowerShell 5.1 test run of the engine additions, and proposing
`feat/clean-json` upstream.

## Trademark

"Mole" and the Mole logo are trademarks of the Mole project. Hypo is an
independent community project: it uses its own name and icon, and does not
imply endorsement by or affiliation with Mole. See the engine's
`TRADEMARK.md`.

## License

GPL-3.0, like the engine. Hypo's UI layer drives the engine as a separate
process; when the two are distributed together, the whole combination is
GPL-3.0.
