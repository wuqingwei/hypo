package engine

import (
	"context"
	"fmt"
	"time"
)

// Types mirroring `mo clean --json` (contract: docs/ENGINE-CONTRACT.md in
// this repo, implemented by the engine's feat/clean-json branch). The
// engine's own guards remain the only deletion authority; these records are
// display facts from a dry run.

// CleanAction is one outcome in the report's records.
type CleanAction string

const (
	CleanWouldRemove  CleanAction = "would_remove"
	CleanWouldRecycle CleanAction = "would_recycle"
	CleanRemoved      CleanAction = "removed"
	CleanRecycled     CleanAction = "recycled"
	CleanSkipped      CleanAction = "skipped"
	CleanFailed       CleanAction = "failed"
)

// CleanRecord is one path the engine's Remove-SafeItem family touched during
// the run, captured at the same point the summary counters are updated.
type CleanRecord struct {
	Description string      `json:"description"`
	Path        string      `json:"path"`
	SizeBytes   int64       `json:"size_bytes"`
	Action      CleanAction `json:"action"`
	Reason      string      `json:"reason,omitempty"`
}

// CleanTotals is the engine's own accounting figure. The engine sums group
// sizes in KB with a per-group ceiling, so SizeBytes may exceed the sum of
// per-record sizes by at most records × 1024; it is authoritative for freed
// space and must never be re-derived from the records.
type CleanTotals struct {
	SizeBytes     int64 `json:"size_bytes"`
	Items         int   `json:"items"`
	Groups        int   `json:"groups"`
	OwnerCleanups int   `json:"owner_cleanups"`
}

// CleanScope reports the flags that actually ran (--system downgrades to
// false without admin, and the report says what ran, not what was asked).
type CleanScope struct {
	System     bool `json:"system"`
	GameMedia  bool `json:"game_media"`
	GpuShaders bool `json:"gpu_shaders"`
}

// CleanReport is one `mo clean --json` envelope.
type CleanReport struct {
	Mode    string        `json:"mode"` // "preview" or "execute"
	Scope   CleanScope    `json:"scope"`
	Records []CleanRecord `json:"records"`
	Totals  CleanTotals   `json:"totals"`
	Error   string        `json:"error,omitempty"`
}

// cleanJSONTimeout: a dry run walks every cleanup target and can take
// minutes on a full disk; the dashboard's other calls keep their short
// budgets. A real run re-walks the same targets while deleting, so it gets
// the same generous budget.
const cleanJSONTimeout = 10 * time.Minute

// CleanPreview runs `clean --dry-run --json` and decodes the envelope. It
// mutates nothing: dry run is the engine's own guarantee, and this call adds
// no flag that could weaken it.
func (c *Client) CleanPreview(ctx context.Context) (*CleanReport, error) {
	ctx, cancel := context.WithTimeout(ctx, cleanJSONTimeout)
	defer cancel()
	res, err := c.powershell(ctx, "clean", "--dry-run", "--json")
	if err != nil {
		return nil, err
	}
	if res.exitCode != 0 {
		return nil, res.failure("mo clean --dry-run --json", ctx)
	}
	var report CleanReport
	if err := decodeJSON(res.stdout, &report); err != nil {
		return nil, fmt.Errorf("decode mo clean output: %w", err)
	}
	if report.Mode != "preview" {
		return nil, fmt.Errorf("mo clean reported mode %q, want preview", report.Mode)
	}
	return &report, nil
}

// CleanRun runs `clean --json` (execute mode). This is the only call in the
// client that asks the engine to delete. The GUI calls it only after an
// explicit user confirmation; selection state from the preview is intent the
// engine re-derives: the engine re-enumerates and re-runs every guard, so
// what it removes is what the guards allow, not what a checkbox promised.
func (c *Client) CleanRun(ctx context.Context) (*CleanReport, error) {
	ctx, cancel := context.WithTimeout(ctx, cleanJSONTimeout)
	defer cancel()
	res, err := c.powershell(ctx, "clean", "--json")
	if err != nil {
		return nil, err
	}
	if res.exitCode != 0 {
		return nil, res.failure("mo clean --json", ctx)
	}
	var report CleanReport
	if err := decodeJSON(res.stdout, &report); err != nil {
		return nil, fmt.Errorf("decode mo clean output: %w", err)
	}
	if report.Mode != "execute" {
		return nil, fmt.Errorf("mo clean reported mode %q, want execute", report.Mode)
	}
	return &report, nil
}
