# Roadmap

Progress is tracked here so the README stays a front page. Each milestone
lists its exit criteria; a milestone is done when they hold on a real Windows
machine, not only in CI.

## M0 — scaffold (current)

- [x] Wails v3 + Vue 3 + TS project, dark dashboard shell
- [x] Engine locator (env / packaged / sibling) with validation
- [x] EngineClient: status / history / analyze, timeouts, cancellation,
      hidden subprocesses, native-tool preference with script fallback
- [x] Dashboard: health, cpu, memory, disks, network, batteries, top
      processes, recent operations
- [x] Orchestration unit tests runnable on Linux (fake engine executables)
- [x] CI: gofmt/vet/test + frontend build + Windows cross-build
- [x] Smoke run on a real Windows machine against a real engine
      (2026-11-05, Windows 10 22H2: packaged engine route, native status
      tool, live metrics, real operations-log history rendered)

## M1 — engine headless surface

Changes land in the engine tree (upstreamable to the Mole `windows` branch as
pure CLI scriptability features); contracts are specified in
[ENGINE-CONTRACT.md](ENGINE-CONTRACT.md).

- [ ] `clean --json`: structured preview and result output
- [ ] `uninstall --select <id> --yes`: non-interactive uninstall with the
      existing exit-code contract
- [ ] Optional: JSONL progress events for long operations
- [ ] Pester coverage for each flag, dry-run parity included

## M2 — the four flows

- [ ] Cleanup page: grouped targets, dry-run preview, execute, result
      reconciliation (expected vs actual, skipped reasons)
- [ ] Analyze page: treemap (ECharts), drill-down, `cleanable` highlights
- [ ] Uninstall page: search/sort/select, charge-limit notices preserved
- [ ] Whitelist editor UI
- [ ] History page: filtering and per-session grouping

## M3 — productization

- [ ] Installer (engine vendored beside the executable), signing, icons
- [ ] Update checks that never auto-install (mirrors the engine's rule)
- [ ] Elevation guidance for `--system` scope without a resident admin helper
- [ ] i18n: English base, zh-CN first

## M4 — native engine (optional, needs a proven reason)

Porting the engine's safety core to Go so the GUI links it directly. Starts
only if the subprocess boundary measurably hurts (startup, progress
granularity), because re-implementing the guards is the riskiest work in this
ecosystem. Pester cases map to Go tests one-for-one before any behavior moves.
