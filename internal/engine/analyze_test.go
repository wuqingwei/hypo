package engine

import (
	"context"
	"testing"
)

func TestAnalyzeReportDecodesContract(t *testing.T) {
	root := t.TempDir()
	writeEngineTree(t, root, false)
	ps := writeScript(t, root, "fake-ps", `printf '%s' '{"scan_status":"complete","path":"C:/Users/t","overview":false,"entries":[{"scan_status":"complete","name":"Cache","path":"C:/Users/t/Cache","size":544000,"is_dir":true,"cleanable":true},{"scan_status":"partial","name":"blob.bin","path":"C:/Users/t/blob.bin","size":123,"is_dir":false}],"large_files":[{"name":"blob.bin","path":"C:/Users/t/blob.bin","size":123}],"total_size":544123,"total_files":2}'`)

	install, err := installationAt(root, SourceEnv)
	if err != nil {
		t.Fatal(err)
	}
	report, err := NewClientWithPowerShell(install, ps).AnalyzeReport(context.Background(), `C:\Users\t`)
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if report.ScanStatus != AnalyzeComplete || len(report.Entries) != 2 {
		t.Fatalf("report = %+v", report)
	}
	if !report.Entries[0].Cleanable || !report.Entries[0].IsDir {
		t.Errorf("entry flags lost: %+v", report.Entries[0])
	}
	if report.Entries[1].ScanStatus != AnalyzePartial {
		t.Errorf("partial state lost: %+v", report.Entries[1])
	}
	if len(report.LargeFiles) != 1 || report.LargeFiles[0].Size != 123 {
		t.Errorf("large files = %+v", report.LargeFiles)
	}
}
