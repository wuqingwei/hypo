package engine

import (
	"context"
	"strings"
	"testing"
)

const cleanPreviewFixture = `{
  "mode": "preview",
  "scope": { "system": false, "game_media": false, "gpu_shaders": false },
  "records": [
    { "description": "User temp files (>7d old)", "path": "C:\\Users\\u\\AppData\\Local\\Temp\\a.tmp", "size_bytes": 1348, "action": "would_remove" },
    { "description": "Notion cache", "path": "C:\\Users\\u\\AppData\\Roaming\\Notion\\GPUCache\\data_1", "size_bytes": 270336, "action": "would_remove" },
    { "description": "npm cache", "path": "C:\\Users\\u\\AppData\\Local\\npm-cache", "size_bytes": 0, "action": "skipped", "reason": "owner running" }
  ],
  "totals": { "size_bytes": 273408, "items": 2, "groups": 2, "owner_cleanups": 1 }
}`

func TestCleanPreviewDecodesEnvelope(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `printf '%s' '`+cleanPreviewFixture+`'`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	report, err := NewClientWithPowerShell(install, ps).CleanPreview(context.Background())
	if err != nil {
		t.Fatalf("clean preview failed: %v", err)
	}
	if report.Mode != "preview" {
		t.Errorf("mode = %q", report.Mode)
	}
	if len(report.Records) != 3 {
		t.Fatalf("records = %d, want 3", len(report.Records))
	}
	first := report.Records[0]
	if first.Action != CleanWouldRemove || first.SizeBytes != 1348 {
		t.Errorf("first record = %+v", first)
	}
	if !strings.Contains(first.Path, `Temp\a.tmp`) {
		t.Errorf("windows path did not round-trip: %q", first.Path)
	}
	if report.Records[2].Action != CleanSkipped || report.Records[2].Reason != "owner running" {
		t.Errorf("skipped record = %+v", report.Records[2])
	}
	if report.Totals.SizeBytes != 273408 || report.Totals.OwnerCleanups != 1 {
		t.Errorf("totals = %+v", report.Totals)
	}
}

func TestCleanPreviewPassesDryRunAndJson(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `
case " $* " in
  *" clean --dry-run --json "*) ;;
  *) echo "unexpected args: $*" >&2; exit 8 ;;
esac
printf '%s' '`+cleanPreviewFixture+`'
`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewClientWithPowerShell(install, ps).CleanPreview(context.Background()); err != nil {
		t.Fatalf("clean preview failed: %v", err)
	}
}

func TestCleanPreviewRejectsNonPreviewMode(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	// A report claiming execute mode from a preview request is a broken or
	// hostile engine; refuse it rather than rendering deletions as a preview.
	ps := writeScript(t, root, "fake-ps", `printf '%s' '{"mode":"execute","scope":{},"records":[],"totals":{"size_bytes":0,"items":0,"groups":0,"owner_cleanups":0}}'`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewClientWithPowerShell(install, ps).CleanPreview(context.Background())
	if err == nil || !strings.Contains(err.Error(), "want preview") {
		t.Fatalf("err = %v, want a mode mismatch", err)
	}
}

func TestCleanPreviewReportsEngineFailure(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `echo "engine broke" >&2; exit 3`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewClientWithPowerShell(install, ps).CleanPreview(context.Background())
	if err == nil || !strings.Contains(err.Error(), "exit code 3") {
		t.Fatalf("err = %v, want the engine exit code", err)
	}
}

const cleanExecuteFixture = `{
  "mode": "execute",
  "scope": { "system": false, "game_media": false, "gpu_shaders": false },
  "records": [
    { "description": "User temp files (>7d old)", "path": "C:\\Users\\u\\AppData\\Local\\Temp\\a.tmp", "size_bytes": 1348, "action": "removed" },
    { "description": "Notion cache", "path": "C:\\Users\\u\\AppData\\Roaming\\Notion\\Cache", "size_bytes": 544000, "action": "skipped", "reason": "file held by owner" }
  ],
  "totals": { "size_bytes": 545280, "items": 1, "groups": 2, "owner_cleanups": 0 }
}`

func TestCleanRunDecodesExecuteEnvelope(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `
case " $* " in
  *" clean --json "*) ;;
  *" dry-run "*) echo "execute must not pass --dry-run" >&2; exit 8 ;;
esac
printf '%s' '`+cleanExecuteFixture+`'
`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	report, err := NewClientWithPowerShell(install, ps).CleanRun(context.Background())
	if err != nil {
		t.Fatalf("clean run failed: %v", err)
	}
	if report.Mode != "execute" {
		t.Errorf("mode = %q", report.Mode)
	}
	if report.Records[0].Action != CleanRemoved || report.Records[0].SizeBytes != 1348 {
		t.Errorf("removed record = %+v", report.Records[0])
	}
	if report.Records[1].Action != CleanSkipped || !strings.Contains(report.Records[1].Reason, "held") {
		t.Errorf("skipped record = %+v", report.Records[1])
	}
	if report.Totals.SizeBytes != 545280 || report.Totals.Items != 1 {
		t.Errorf("totals = %+v", report.Totals)
	}
}

func TestCleanRunRejectsPreviewMode(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	// A preview envelope answering an execute request means the engine did
	// not delete; reporting it as a result would lie to the user.
	ps := writeScript(t, root, "fake-ps", `printf '%s' '{"mode":"preview","scope":{},"records":[],"totals":{"size_bytes":0,"items":0,"groups":0,"owner_cleanups":0}}'`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewClientWithPowerShell(install, ps).CleanRun(context.Background())
	if err == nil || !strings.Contains(err.Error(), "want execute") {
		t.Fatalf("err = %v, want a mode mismatch", err)
	}
}
