package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Per-command timeouts. The engine owns its own internal waits (for example
// the uninstaller descendant ceiling); these only bound how long Hypo sits on
// a pipe.
const (
	statusTimeout = 30 * time.Second
	// statusScriptTimeout: the mole.ps1 route also downloads or builds the
	// native tool when it is missing, which can legitimately take minutes on
	// a slow network. Killing it at the dashboard's 30 s only turns a
	// recoverable first run into an opaque timeout.
	statusScriptTimeout = 2 * time.Minute
	historyTimeout      = 30 * time.Second
	analyzeTimeout      = 10 * time.Minute
)

// Client runs engine commands and decodes their JSON output. It is safe for
// concurrent use; every call spawns its own process.
type Client struct {
	install *Installation
	psExe   string
}

// NewClient drives the installation with Windows PowerShell 5.1, the host the
// engine itself targets first.
func NewClient(install *Installation) *Client {
	return &Client{install: install, psExe: "powershell.exe"}
}

// NewClientWithPowerShell overrides the PowerShell host. Tests use it to
// substitute a fake executable on non-Windows systems.
func NewClientWithPowerShell(install *Installation, psExe string) *Client {
	return &Client{install: install, psExe: psExe}
}

// Installation returns the engine tree this client drives.
func (c *Client) Installation() *Installation { return c.install }

// Status returns one metrics snapshot. It prefers the native status tool (no
// PowerShell startup cost) and falls back to mole.ps1, which also downloads
// or builds the tool when it is missing or broken.
func (c *Client) Status(ctx context.Context) (*StatusSnapshot, error) {
	if c.install.StatusBin != "" {
		if snap, err := c.statusViaTool(ctx); err == nil {
			return snap, nil
		}
		// Fall through to the script route, which self-heals the binary.
	}
	return c.statusViaScript(ctx)
}

func (c *Client) statusViaTool(ctx context.Context) (*StatusSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, statusTimeout)
	defer cancel()
	res, err := run(ctx, c.install.StatusBin, "--json")
	if err != nil {
		return nil, err
	}
	if res.exitCode != 0 {
		return nil, res.failure("status tool", ctx)
	}
	var snap StatusSnapshot
	if err := decodeJSON(res.stdout, &snap); err != nil {
		return nil, fmt.Errorf("decode status tool output: %w", err)
	}
	return &snap, nil
}

func (c *Client) statusViaScript(ctx context.Context) (*StatusSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, statusScriptTimeout)
	defer cancel()
	res, err := c.powershell(ctx, "status", "--json")
	if err != nil {
		return nil, err
	}
	if res.exitCode != 0 {
		return nil, res.failure("mo status", ctx)
	}
	var snap StatusSnapshot
	if err := decodeJSON(res.stdout, &snap); err != nil {
		return nil, fmt.Errorf("decode mo status output: %w", err)
	}
	return &snap, nil
}

// History returns the newest operations-log entries. limit follows the
// engine's rules: 1-200, 0 keeps the engine default (20).
func (c *Client) History(ctx context.Context, limit int) (*HistoryReport, error) {
	ctx, cancel := context.WithTimeout(ctx, historyTimeout)
	defer cancel()
	args := []string{"history", "--json"}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	res, err := c.powershell(ctx, args...)
	if err != nil {
		return nil, err
	}
	if res.exitCode != 0 {
		return nil, res.failure("mo history", ctx)
	}
	var report HistoryReport
	if err := decodeJSON(res.stdout, &report); err != nil {
		return nil, fmt.Errorf("decode mo history output: %w", err)
	}
	return &report, nil
}

// Analyze returns the engine's directory scan as raw JSON (contract:
// cmd/analyze/json.go, identical to the macOS CLI). An empty path keeps the
// engine's own default. The JSON mode is read-only: it never deletes and
// never writes the operations log.
func (c *Client) Analyze(ctx context.Context, path string) (json.RawMessage, error) {
	if c.install.AnalyzeBin != "" {
		if raw, err := c.analyzeViaTool(ctx, path); err == nil {
			return raw, nil
		}
	}
	return c.analyzeViaScript(ctx, path)
}

func (c *Client) analyzeViaTool(ctx context.Context, path string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, analyzeTimeout)
	defer cancel()
	args := []string{"--json"}
	if path != "" {
		args = append(args, path)
	}
	res, err := run(ctx, c.install.AnalyzeBin, args...)
	if err != nil {
		return nil, err
	}
	if res.exitCode != 0 {
		return nil, res.failure("analyze tool", ctx)
	}
	return decodeRawJSON(res.stdout)
}

func (c *Client) analyzeViaScript(ctx context.Context, path string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, analyzeTimeout)
	defer cancel()
	args := []string{"analyze", "--json"}
	if path != "" {
		args = append(args, path)
	}
	res, err := c.powershell(ctx, args...)
	if err != nil {
		return nil, err
	}
	if res.exitCode != 0 {
		return nil, res.failure("mo analyze", ctx)
	}
	return decodeRawJSON(res.stdout)
}

// powershell runs `mole.ps1 <args...>`. The flags keep the host machine-safe
// for a GUI: no profile (speed, no user script side effects), non-interactive,
// and bypassed execution policy so a downloaded engine tree still runs — the
// engine's own installers use the same policy.
func (c *Client) powershell(ctx context.Context, args ...string) (*runResult, error) {
	full := append([]string{
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-File", c.install.MoleScript,
	}, args...)
	return run(ctx, c.psExe, full...)
}

// runResult is one finished engine process.
type runResult struct {
	stdout   string
	stderr   string
	exitCode int
}

// failure turns a non-zero exit into an actionable error, including the
// context deadline when the run was killed by one and the last stderr lines
// (the engine routes progress and diagnostics there when output is captured).
func (r *runResult) failure(what string, ctx context.Context) error {
	msg := fmt.Sprintf("%s failed with exit code %d", what, r.exitCode)
	if ctxErr := ctx.Err(); ctxErr != nil {
		msg = fmt.Sprintf("%s (%v)", msg, ctxErr)
	}
	if tail := lastLines(r.stderr, 6); tail != "" {
		msg += "\n" + tail
	}
	return errors.New(msg)
}

// run executes one engine process with captured output. A non-zero exit code
// is a runResult, not an error: only a failure to start (or an internal
// exec error) is.
func run(ctx context.Context, name string, args ...string) (*runResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	applySysProcAttr(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := &runResult{stdout: stdout.String(), stderr: stderr.String()}
	if err == nil {
		return res, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.exitCode = exitErr.ExitCode()
		return res, nil
	}
	return nil, fmt.Errorf("start %s: %w", name, err)
}

// decodeJSON unmarshals engine stdout into v, tolerating a UTF-8 BOM.
func decodeJSON(data string, v any) error {
	return json.Unmarshal(stripBOM([]byte(data)), v)
}

// decodeRawJSON validates that stdout really is JSON before handing it on.
func decodeRawJSON(data string) (json.RawMessage, error) {
	raw := json.RawMessage(stripBOM([]byte(data)))
	if !json.Valid(raw) {
		return nil, fmt.Errorf("engine output is not valid JSON")
	}
	return raw, nil
}

func stripBOM(data []byte) []byte {
	return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
}

func lastLines(s string, n int) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\r\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	var kept []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			kept = append(kept, strings.TrimRight(line, "\r"))
		}
	}
	return strings.Join(kept, "\n")
}

// executableDir is the directory of the running Hypo binary; empty on failure
// so the packaged and sibling searches degrade to a no-op.
func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}
