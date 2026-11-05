package engine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Types mirroring the engine's uninstall contracts: `mo uninstall --list`
// (JSON when piped, bin/uninstall.ps1 ConvertTo-UninstallListRecord) and the
// `--select --json` result envelope (acea1a3f). The engine's uninstaller
// rereads, Store eligibility checks and charge-limit notices remain the only
// authority; these are display facts.

// UninstallApp is one row of the installed-applications inventory.
type UninstallApp struct {
	Name      string  `json:"name"`
	ID        string  `json:"id"`
	Source    string  `json:"source"`
	Version   *string `json:"version"`
	Publisher *string `json:"publisher"`
	Path      *string `json:"path"`
	Size      string  `json:"size"`
	SizeBytes *int64  `json:"size_bytes"`
}

// UninstallOutcome is one app's result in a --select run.
type UninstallOutcome struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Outcome string `json:"outcome"` // uninstalled | skipped | failed
	Detail  string `json:"detail"`
}

// UninstallResult is one `mo uninstall --select --json` envelope.
type UninstallResult struct {
	Results    []UninstallOutcome `json:"results"`
	Incomplete bool               `json:"incomplete"`
	NotFound   []string           `json:"not_found,omitempty"`
}

// UninstallRun is the outcome Hypo reports after an execute run: the
// envelope plus the charge-limit notices the engine printed on stderr.
type UninstallRun struct {
	Result  UninstallResult `json:"result"`
	Notices []string        `json:"notices,omitempty"`
}

const (
	uninstallListTimeout = 2 * time.Minute
	// The engine's own uninstaller ceiling is a fixed 30 minutes per batch;
	// Hypo's budget is slightly above it so the engine, not the GUI, decides.
	uninstallRunTimeout = 32 * time.Minute
)

// UninstallList returns the installed applications. It prefers --list with
// piped stdout, which the engine answers as JSON.
func (c *Client) UninstallList(ctx context.Context) ([]UninstallApp, error) {
	ctx, cancel := context.WithTimeout(ctx, uninstallListTimeout)
	defer cancel()
	res, err := c.powershell(ctx, "uninstall", "--list")
	if err != nil {
		return nil, err
	}
	if res.exitCode != 0 {
		return nil, res.failure("mo uninstall --list", ctx)
	}
	var apps []UninstallApp
	if err := decodeJSON(res.stdout, &apps); err != nil {
		return nil, fmt.Errorf("decode mo uninstall --list output: %w", err)
	}
	return apps, nil
}

// UninstallSelect runs `uninstall --select <keys> --yes --json` and collects
// the result plus the engine's stderr NOTICE lines (charge limits). This is
// the only call besides CleanRun that asks the engine to remove software;
// the caller confirms with the user first.
//
// Exit code 1 is a designed outcome, not a transport failure: the engine
// prints the full JSON envelope (with incomplete=true and per-row outcomes)
// when any uninstaller failed, kept its entry or was skipped as ineligible,
// and only then exits 1. The envelope is decoded and returned in that case;
// a hard error is anything else — a different exit code, or stdout that does
// not decode.
func (c *Client) UninstallSelect(ctx context.Context, ids []string) (*UninstallRun, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no selection keys given")
	}
	ctx, cancel := context.WithTimeout(ctx, uninstallRunTimeout)
	defer cancel()
	res, err := c.powershell(ctx, "uninstall", "--select", strings.Join(ids, ","), "--yes", "--json")
	if err != nil {
		return nil, err
	}
	var result UninstallResult
	if decodeErr := decodeJSON(res.stdout, &result); decodeErr != nil {
		// The envelope failed to parse. With a non-zero exit the engine's own
		// diagnostic wins; with exit 0 the stdout head is the evidence of what
		// polluted the pipe (a leaked interactive row, a BOM, a truncated
		// write), so surface it instead of a bare decode error.
		if res.exitCode != 0 {
			return nil, res.failure("mo uninstall --select", ctx)
		}
		head := strings.TrimSpace(res.stdout)
		if len(head) > 200 {
			head = head[:200] + "…"
		}
		if head == "" {
			head = "(stdout empty)"
		}
		return nil, fmt.Errorf("decode mo uninstall --select output: %w\nstdout head: %s", decodeErr, head)
	}
	if res.exitCode != 0 && res.exitCode != 1 {
		return nil, res.failure("mo uninstall --select", ctx)
	}
	run := &UninstallRun{Result: result}
	for _, line := range strings.Split(res.stderr, "\n") {
		if notice, ok := strings.CutPrefix(strings.TrimSpace(line), "NOTICE: "); ok && notice != "" {
			run.Notices = append(run.Notices, notice)
		}
	}
	return run, nil
}
