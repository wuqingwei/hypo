package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Whitelist access. The file is the engine's own protected-path config
// (~/.config/mole/whitelist.txt, one pattern per line, '#' comments). The
// engine re-reads it on every run, so line edits take effect immediately;
// its fail-closed semantics are untouched (an unreadable or invalid line
// just never matches, and unreadable metadata retains everything).

// Whitelist is the merged view the UI renders.
type Whitelist struct {
	// Path is the file the user patterns live in.
	Path string `json:"path"`
	// Defaults are the engine's built-in protected patterns; read-only for
	// Hypo. Empty when the engine could not report them (the UI shows a
	// note instead of inventing rows).
	Defaults []string `json:"defaults"`
	// UserPatterns are the file's own lines, verbatim minus blank lines and
	// '#' comments.
	UserPatterns []string `json:"user_patterns"`
	// Error carries a read failure; an unreadable whitelist must not render
	// as an empty editable list.
	Error string `json:"error,omitempty"`
}

// whitelistTimeout covers one PowerShell call that dot-sources base.ps1.
const whitelistTimeout = 60 * time.Second

// GetWhitelist returns the engine's built-in patterns and the user file's
// current lines. Read-only; a missing user file is an empty list, not an
// error, matching the engine's own semantics.
func (c *Client) GetWhitelist(ctx context.Context) (*Whitelist, error) {
	ctx, cancel := context.WithTimeout(ctx, whitelistTimeout)
	defer cancel()

	userPath := filepath.Join(engineConfigDir(), "whitelist.txt")

	// One hidden PowerShell call asks the engine for its built-in list. The
	// user patterns are read from disk directly — the same bytes the engine
	// reads, without a shell round-trip.
	script := `. $env:MOLE_LIB\core\base.ps1; ` +
		`$p = @($script:DefaultWhitelistPatterns | ForEach-Object { [Environment]::ExpandEnvironmentVariables($_) }); ` +
		`$u = @(); if (Test-Path -LiteralPath $env:MOLE_WHITELIST_FILE) { $u = @(Get-Content -LiteralPath $env:MOLE_WHITELIST_FILE | ForEach-Object { $_.Trim() } | Where-Object { $_ -and -not $_.StartsWith('#') }) }; ` +
		`[ordered]@{ defaults = $p; user = $u } | ConvertTo-Json -Compress`
	res, err := run(ctx, c.psExe, []string{
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-Command", "$env:MOLE_LIB='" + c.install.Root + "\\lib'; " +
			"$env:MOLE_WHITELIST_FILE='" + userPath + "'; " + script,
	}...)
	if err != nil {
		return nil, err
	}
	if res.exitCode != 0 {
		return nil, res.failure("whitelist export", ctx)
	}

	var payload struct {
		Defaults []string `json:"defaults"`
		User     []string `json:"user"`
	}
	if err := decodeJSON(res.stdout, &payload); err != nil {
		return nil, fmt.Errorf("decode whitelist export: %w", err)
	}
	wl := &Whitelist{
		Path:         userPath,
		Defaults:     payload.Defaults,
		UserPatterns: payload.User,
	}
	if strings.TrimSpace(res.stderr) != "" {
		// base.ps1 diagnostics (e.g. StrictMode noise) mean the defaults may
		// be incomplete; show them as read but flag the doubt.
		wl.Error = "engine export wrote diagnostics; the default list may be incomplete"
	}
	return wl, nil
}

// AddWhitelistEntry appends one pattern line. It refuses empties, comment
// lines and duplicates, and creates the file only when missing (the engine
// treats absence as an empty list).
func (c *Client) AddWhitelistEntry(ctx context.Context, pattern string) (*Whitelist, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || strings.HasPrefix(pattern, "#") {
		return nil, fmt.Errorf("a whitelist pattern must be a non-comment path or wildcard")
	}
	ctx, cancel := context.WithTimeout(ctx, whitelistTimeout)
	defer cancel()

	path := filepath.Join(engineConfigDir(), "whitelist.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	existing, _ := readUserPatterns(path)
	for _, line := range existing {
		if strings.EqualFold(line, pattern) {
			return nil, fmt.Errorf("pattern already whitelisted: %s", pattern)
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if _, err := f.WriteString(pattern + "\r\n"); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return c.GetWhitelist(ctx)
}

// RemoveWhitelistEntry deletes the first matching line. Matching is exact
// (case-insensitive, trimmed) so a wildcard pattern cannot swallow its
// neighbours. A missing file is a no-op returning the current view.
func (c *Client) RemoveWhitelistEntry(ctx context.Context, pattern string) (*Whitelist, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("no pattern given")
	}
	ctx, cancel := context.WithTimeout(ctx, whitelistTimeout)
	defer cancel()

	path := filepath.Join(engineConfigDir(), "whitelist.txt")
	lines, err := readRawLines(path)
	if err != nil || len(lines) == 0 {
		// Absent or unreadable: the engine retains everything anyway; report
		// rather than mutate what cannot be parsed.
		wl, getErr := c.GetWhitelist(ctx)
		if getErr != nil {
			return nil, err
		}
		if err == nil && len(lines) == 0 {
			return wl, fmt.Errorf("pattern not found: %s", pattern)
		}
		return wl, nil
	}

	kept := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Only pattern lines are removable: comments and blanks are file
		// furniture, and removing one must not count as removing a pattern.
		if !removed && trimmed != "" && !strings.HasPrefix(trimmed, "#") && strings.EqualFold(trimmed, pattern) {
			removed = true
			continue
		}
		kept = append(kept, line)
	}
	if !removed {
		wl, getErr := c.GetWhitelist(ctx)
		if getErr != nil {
			return nil, getErr
		}
		return wl, fmt.Errorf("pattern not found: %s", pattern)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(kept, "\r\n")+"\r\n"), 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return nil, err
	}
	return c.GetWhitelist(ctx)
}

// readUserPatterns parses the file the way the engine does: trimmed,
// non-empty, non-comment.
func readUserPatterns(path string) ([]string, error) {
	lines, err := readRawLines(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			out = append(out, trimmed)
		}
	}
	return out, nil
}

func readRawLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	// A trailing newline is a line terminator, not an empty line; drop the
	// phantom last element so counts stay predictable.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

// engineConfigDir mirrors the engine's $env:USERPROFILE\.config\mole.
func engineConfigDir() string {
	profile := os.Getenv("USERPROFILE")
	if profile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", ".config", "mole")
		}
		profile = home
	}
	return filepath.Join(profile, ".config", "mole")
}
