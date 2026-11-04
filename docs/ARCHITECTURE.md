# Architecture

Hypo is deliberately a thin product. The value — the safety rules, the target
catalog, the measurement logic — lives in the Mole CLI engine, and this
document explains how Hypo stays a dashboard over it instead of growing its
own opinions about deletion.

```
┌────────────── frontend (WebView2, Vue 3) ──────────────┐
│ dashboard: engine chip, health, cpu/mem/disk/net,      │
│ batteries, top processes, recent operations            │
└───────────────────────┬────────────────────────────────┘
                        │ Wails bindings (generated TS)
┌───────────────────────┴────────────────────────────────┐
│ Go service layer (internal/app)                        │
│  engine presence, typed snapshots, friendly failures   │
├────────────────────────────────────────────────────────┤
│ engine client (internal/engine)                        │
│  locate tree · spawn hidden processes · JSON decode ·  │
│  timeouts · cancellation · exit-code/stderr reporting  │
└───────────────────────┬────────────────────────────────┘
                        │ subprocess + JSON
┌───────────────────────┴────────────────────────────────┐
│ Mole CLI engine (GPL-3.0, unmodified)                  │
│  status.exe --json · analyze.exe --json                │
│  powershell mole.ps1 status/history/analyze            │
│  (later: clean --json, uninstall --select — proposed)  │
└────────────────────────────────────────────────────────┘
```

## Layer contracts

**Frontend.** Renders facts, collects intent. It polls `GetStatus` (5 s,
paused when hidden), never parses engine text output, and shows engine errors
verbatim rather than guessing at causes.

**Service layer (`internal/app`).** Owns no cleanup logic. It locates the
engine once, re-locates on demand, and converts engine failures into messages
the UI can display. A missing engine is a renderable state, not a crash.

**Engine client (`internal/engine`).**

- *Locator* (`locator.go`): search order is `HYPO_ENGINE_ROOT` (an invalid
  value is an error, not a silent skip), `<exe dir>/engine` (how installers
  will ship the engine), then a `Mole` checkout beside the Hypo source tree
  for development. A candidate is valid only when both `mole.ps1` and
  `lib/core/base.ps1` exist — a tree without the safety core is not an engine.
- *Client* (`client.go`): every call spawns its own process with captured
  stdout/stderr, a per-command timeout, and hidden windows on Windows
  (`CREATE_NO_WINDOW` — a GUI must not flash consoles). Non-zero exits become
  errors carrying the exit code and the last stderr lines. Native Go tools
  (`bin/status.exe`, `bin/analyze.exe`) are preferred; `mole.ps1` is the
  fallback that also self-heals missing binaries.

## Safety invariants

These are the rules that survive every milestone. A PR that violates any of
them is wrong, whatever else it improves.

1. Hypo never deletes. No file-removal API appears above the engine layer.
2. Recycle-first is the engine's contract; Hypo never passes a flag that asks
   for permanent deletion, and never falls back when recycling fails.
3. Preview and execute must remain one code path *in the engine*; Hypo merely
   renders both and never re-implements either.
4. Re-check at removal time belongs to the engine. The GUI's selection state
   is intent: between preview and execute the engine re-enumerates and
   re-runs every guard (owner checks, junctions, whitelist, protected paths).
5. The operations log is the record. Hypo reads it via `history --json` and
   renders it; it does not keep a parallel ledger.
6. Display sizes are measurements, not authority. A byte count shown in the
   UI never widens what the engine will remove.

## Engine versioning

Hypo ships a pinned engine tree (`engine/` beside the executable) so the GUI
and the safety rules it was tested against move together. A Mole install on
`PATH` is never used silently; the locator only knows the three locations
above. When the engine contract grows (see
[ENGINE-CONTRACT.md](ENGINE-CONTRACT.md)), the pinned tree and this document
change in the same PR.
