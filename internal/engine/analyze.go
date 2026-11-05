package engine

import "context"

// Types mirroring `analyze --json` (contract: the engine's
// cmd/analyze/json.go, shared field-for-field with the macOS CLI). The JSON
// mode is read-only: no TUI, no deletions, no operations-log writes.

// AnalyzeScanState is measurement coverage for a row, not an atomic snapshot.
type AnalyzeScanState string

const (
	AnalyzeComplete    AnalyzeScanState = "complete"
	AnalyzePartial     AnalyzeScanState = "partial"
	AnalyzeUnavailable AnalyzeScanState = "unavailable"
)

// AnalyzeEntry is one child of the scanned directory (or one overview row).
type AnalyzeEntry struct {
	ScanStatus AnalyzeScanState `json:"scan_status"`
	Name       string           `json:"name"`
	Path       string           `json:"path"`
	Size       int64            `json:"size"`
	IsDir      bool             `json:"is_dir"`
	Insight    bool             `json:"insight,omitempty"`
	Cleanable  bool             `json:"cleanable,omitempty"`
	LastAccess string           `json:"last_access,omitempty"`
}

// AnalyzeFile is one large file in the scan's large-file list.
type AnalyzeFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// AnalyzeReport is one `analyze --json` output.
type AnalyzeReport struct {
	ScanStatus AnalyzeScanState `json:"scan_status"`
	Path       string           `json:"path"`
	Overview   bool             `json:"overview"`
	Entries    []AnalyzeEntry   `json:"entries"`
	LargeFiles []AnalyzeFile    `json:"large_files,omitempty"`
	TotalSize  int64            `json:"total_size"`
	TotalFiles int64            `json:"total_files,omitempty"`
}

// Analyze runs the engine's `analyze --json [path]`. An empty path scans the
// engine's overview set. This is a read-only call by the engine's own
// contract; Hypo adds no flag that could change that.
func (c *Client) AnalyzeReport(ctx context.Context, path string) (*AnalyzeReport, error) {
	raw, err := c.Analyze(ctx, path)
	if err != nil {
		return nil, err
	}
	var report AnalyzeReport
	if err := decodeJSON(string(raw), &report); err != nil {
		return nil, err
	}
	return &report, nil
}
