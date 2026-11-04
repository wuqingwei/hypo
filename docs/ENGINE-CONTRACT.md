# Engine contract

How Hypo talks to the Mole CLI engine. Everything here is a read-only or
explicitly requested operation; the engine's own guards remain the only
deletion authority.

Conventions:

- JSON goes to **stdout**, diagnostics and progress to **stderr** (the engine
  already routes update-notice chatter away from redirected output).
- Non-ASCII is `\uXXXX`-escaped where the engine already does so
  (`ConvertTo-UninstallListJson`), so console code pages cannot corrupt
  app names on the way out.
- Exit codes are contractual: the engine already exits non-zero when an
  uninstall fails or is skipped; Hypo surfaces, never retries on its own.
- A timeout or kill from Hypo is a UI fact only. It never implies the engine
  finished, cleaned up, or is safe to re-run blindly.

## Implemented today

### `status --json`

`bin/status.exe --json` (native, preferred) or
`powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File mole.ps1 status --json`.

One snapshot; fields mirrored in `internal/engine/types.go` (`StatusSnapshot`),
defined in the engine's `cmd/status/metrics.go`. Notable fields:
`health_score`, `cpu.per_core`, `memory.{used,total,used_percent,swap_*}`,
`disks[]` with `mount/used/total/used_percent/smart_status`, `network[]`,
`batteries[]`, `top_processes[]`.

### `analyze --json [path]`

`bin/analyze.exe --json [path]` or `mole.ps1 analyze --json [path]`.

One JSON object (contract shared with the macOS CLI, see the engine's
`cmd/analyze/json.go`): `scan_status`, `path`, `entries[]`
(`name/path/size/is_dir/insight/cleanable/last_access`), `large_files[]`,
`total_size`, `total_files`. JSON mode is read-only: no TUI, no deletions, no
operations-log writes.

### `history --json [--limit N]`

`mole.ps1 history --json --limit N`.

`{logs:{operations}, limit, sessions[], operations[]}` —
`internal/engine/types.go` (`HistoryReport`). `operations[]` rows are
`{timestamp, command, run_id, action, path, detail}`; `sessions[]` aggregate
per run with action counts.

### `uninstall --list` (JSON when piped)

`mole.ps1 uninstall --list` with stdout captured.

Array of `{name, id, source, version, publisher, path, size, size_bytes}`;
`id` is the menu's selection key. Uses a 24-hour cache; `--rescan` forces a
refresh.

## Proposed for M1 (not yet in the engine)

### `clean --json`

`mole.ps1 clean --dry-run --json` (preview) and `mole.ps1 clean --json`
(execute). Scope flags (`--system`, `--game-media`, `--gpu-shaders`) behave
exactly as today.

Preview:

```json
{
  "mode": "preview",
  "scope": { "system": false, "game_media": false, "gpu_shaders": false },
  "targets": [
    {
      "id": "browser-chromium-cache",
      "label": "Chromium cache",
      "category": "browser",
      "path": "C:\\Users\\u\\AppData\\Local\\...\\Cache",
      "size_bytes": 123456,
      "entries": 42,
      "whitelisted": false,
      "reason": ""
    }
  ],
  "totals": { "targets": 1, "bytes": 123456 }
}
```

Execute:

```json
{
  "mode": "execute",
  "scope": { "system": false, "game_media": false, "gpu_shaders": false },
  "results": [
    { "id": "browser-chromium-cache", "outcome": "removed", "bytes": 123000, "reason": "" },
    { "id": "old-dumps", "outcome": "skipped", "bytes": 0, "reason": "file held by owner" }
  ],
  "totals": { "removed": 1, "skipped": 1, "failed": 0, "bytes": 123000 }
}
```

Invariants the engine already enforces and the JSON must preserve:

- dry-run counts and the real-run summary come from the same measurement
  helpers — preview numbers must equal what execution reports for unchanged
  state;
- `whitelisted: true` rows appear in the preview (so the UI can explain them)
  but are never candidates for execution;
- dry runs write no operations-log lines and mutate nothing;
- outcomes use the log's vocabulary (`removed`, `recycled`, `skipped`,
  `failed`).

### `uninstall --select <id> [--yes]`

`mole.ps1 uninstall --select <key> --yes` runs the existing interactive
pipeline headlessly: same reread-before-launch, same leftover ownership
checks, same charge-limit notices (emitted as a `notices[]` field in a JSON
envelope or on stderr — to be settled with the engine's maintainers), same
exit-code contract (non-zero when anything fails or is skipped as ineligible).

### Progress events (optional)

Long operations (clean, uninstall, analyze of a big tree) emit
newline-delimited JSON on **stderr**:

```json
{"event": "progress", "op": "clean", "target": "browser-chromium-cache", "done": 3, "total": 12}
```

Hypo ignores anything on stderr that does not parse as an event line; the
plain-text diagnostics remain for humans.
