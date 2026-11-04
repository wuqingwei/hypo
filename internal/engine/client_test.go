package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeScript creates an executable fake and returns its path. On Linux this
// is a shell script standing in for powershell.exe or a TUI binary; the
// engine package never inspects the file type, only runs it.
func writeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

const statusFixture = `{
  "collected_at": "2026-11-04T12:00:00Z",
  "host": "TEST-PC",
  "platform": "windows",
  "uptime": "2 days",
  "uptime_seconds": 172800,
  "procs": 210,
  "hardware": {"cpu_model": "Test CPU", "total_ram": "32 GB", "os_version": "Windows 11"},
  "health_score": 88,
  "health_score_msg": "Good",
  "cpu": {"usage": 12.5, "per_core": [10, 15], "core_count": 2, "logical_cpu": 4},
  "memory": {"used": 1, "total": 2, "used_percent": 50},
  "disks": [{"mount": "C:", "used": 1, "total": 2, "used_percent": 50, "fstype": "NTFS"}],
  "network": [{"name": "Ethernet", "ip": "10.0.0.2", "rx_rate_mbs": 0.1, "tx_rate_mbs": 0.2}]
}`

func TestStatusPrefersNativeTool(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `echo "script was used" >&2; exit 9`)
	writeScript(t, filepath.Join(root, "bin"), "status.exe", `printf '%s' '`+statusFixture+`'`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	if install.StatusBin == "" {
		t.Fatal("fixture status.exe was not resolved")
	}
	snap, err := NewClientWithPowerShell(install, ps).Status(context.Background())
	if err != nil {
		t.Fatalf("status via native tool failed: %v", err)
	}
	if snap.Host != "TEST-PC" || snap.HealthScore != 88 || snap.CPU.Usage != 12.5 {
		t.Errorf("snapshot decoded wrong: %+v", snap)
	}
	if len(snap.Disks) != 1 || snap.Disks[0].Mount != "C:" {
		t.Errorf("disks decoded wrong: %+v", snap.Disks)
	}
}

func TestStatusFallsBackToScriptAndStripsBOM(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	// A BOM in front of the JSON must not break decoding; PowerShell 5.1 can
	// emit one depending on how the engine writes its output.
	ps := writeScript(t, root, "fake-ps", `printf '\xEF\xBB\xBF'; printf '%s' '`+statusFixture+`'`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := NewClientWithPowerShell(install, ps).Status(context.Background())
	if err != nil {
		t.Fatalf("status via script failed: %v", err)
	}
	if snap.Host != "TEST-PC" {
		t.Errorf("snapshot decoded wrong: %+v", snap)
	}
}

func TestStatusReportsExitCodeAndStderrTail(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `echo "line one" >&2; echo "boom: engine exploded" >&2; exit 3`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewClientWithPowerShell(install, ps).Status(context.Background())
	if err == nil {
		t.Fatal("a non-zero engine exit must be an error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "exit code 3") {
		t.Errorf("error misses exit code: %s", msg)
	}
	if !strings.Contains(msg, "boom: engine exploded") {
		t.Errorf("error misses the stderr tail: %s", msg)
	}
}

func TestStatusTimeoutIsAnError(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `sleep 5`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = NewClientWithPowerShell(install, ps).Status(ctx)
	if err == nil {
		t.Fatal("a killed engine process must be an error")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("error should name the deadline: %s", err)
	}
}

func TestHistoryPassesLimitAndDecodesReport(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `
case " $* " in
  *" history --json --limit 5 "*) ;;
  *) echo "unexpected args: $*" >&2; exit 8 ;;
esac
printf '%s' '{"logs":{"operations":"C:/ops.log"},"limit":5,"sessions":[{"command":"clean","run_id":"r1","started_at":"2026-11-04 12:00:00","operation_count":2,"actions":{"removed":1,"skipped":1}}],"operations":[{"timestamp":"2026-11-04 12:00:01","command":"clean","run_id":"r1","action":"removed","path":"C:/tmp/x","detail":""}]}'
`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	report, err := NewClientWithPowerShell(install, ps).History(context.Background(), 5)
	if err != nil {
		t.Fatalf("history failed: %v", err)
	}
	if report.Limit != 5 || len(report.Operations) != 1 {
		t.Errorf("report decoded wrong: %+v", report)
	}
	if report.Sessions[0].Actions.Skipped != 1 {
		t.Errorf("session actions decoded wrong: %+v", report.Sessions[0].Actions)
	}
}

func TestHistoryWithoutLimitOmitsTheFlag(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `
case " $* " in
  *" --limit "*) echo "limit must not be passed" >&2; exit 8 ;;
esac
printf '%s' '{"logs":{"operations":""},"limit":20,"sessions":[],"operations":[]}'
`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewClientWithPowerShell(install, ps).History(context.Background(), 0); err != nil {
		t.Fatalf("history failed: %v", err)
	}
}

func TestAnalyzeReturnsRawJSON(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `printf '%s' '{"scan_status":"complete","path":"C:/Users/t","entries":[],"total_size":10}'`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := NewClientWithPowerShell(install, ps).Analyze(context.Background(), `C:\Users\t`)
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if !strings.Contains(string(raw), `"scan_status"`) {
		t.Errorf("raw JSON passed through wrong: %s", raw)
	}
}
