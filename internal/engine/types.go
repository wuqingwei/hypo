package engine

import "time"

// The types in this file mirror the engine's JSON contracts field for field:
//
//   - StatusSnapshot: cmd/status/metrics.go, printed by `mo status --json`
//   - HistoryReport:  lib/core/history.ps1, printed by `mo history --json`
//
// They are read-only display facts for Hypo. A size shown here is a
// measurement, never deletion authority; only the engine's own guards decide
// what may be removed.

// StatusSnapshot is one machine-readable metrics snapshot.
type StatusSnapshot struct {
	CollectedAt    time.Time    `json:"collected_at"`
	Host           string       `json:"host"`
	Platform       string       `json:"platform"`
	Uptime         string       `json:"uptime"`
	UptimeSeconds  uint64       `json:"uptime_seconds"`
	Procs          uint64       `json:"procs"`
	Hardware       HardwareInfo `json:"hardware"`
	HealthScore    int          `json:"health_score"`
	HealthScoreMsg string       `json:"health_score_msg"`

	CPU            CPUStatus          `json:"cpu"`
	Memory         MemoryStatus       `json:"memory"`
	Disks          []DiskStatus       `json:"disks"`
	DiskIO         DiskIOStatus       `json:"disk_io"`
	Network        []NetworkStatus    `json:"network"`
	NetworkHistory NetworkHistory     `json:"network_history"`
	Proxy          ProxyStatus        `json:"proxy"`
	Batteries      []BatteryStatus    `json:"batteries"`
	TopProcesses   []ProcessInfo      `json:"top_processes"`
	ProcessWatch   ProcessWatchConfig `json:"process_watch"`
	ProcessAlerts  []ProcessAlert     `json:"process_alerts"`
}

// HardwareInfo describes the machine.
type HardwareInfo struct {
	Model       string `json:"model"`
	CPUModel    string `json:"cpu_model"`
	TotalRAM    string `json:"total_ram"`
	DiskSize    string `json:"disk_size"`
	OSVersion   string `json:"os_version"`
	RefreshRate string `json:"refresh_rate"`
}

// CPUStatus is overall and per-core CPU usage.
type CPUStatus struct {
	Usage            float64   `json:"usage"`
	PerCore          []float64 `json:"per_core"`
	PerCoreEstimated bool      `json:"per_core_estimated"`
	CoreCount        int       `json:"core_count"`
	LogicalCPU       int       `json:"logical_cpu"`
}

// MemoryStatus is physical memory and swap usage.
type MemoryStatus struct {
	Used        uint64  `json:"used"`
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapTotal   uint64  `json:"swap_total"`
	Cached      uint64  `json:"cached"`
	// Pressure stays empty on Windows; the engine never sets it.
	Pressure string `json:"pressure"`
}

// DiskStatus is one mounted volume.
type DiskStatus struct {
	Mount       string  `json:"mount"`
	Device      string  `json:"device"`
	Used        uint64  `json:"used"`
	Total       uint64  `json:"total"`
	UsedPercent float64 `json:"used_percent"`
	Fstype      string  `json:"fstype"`
	External    bool    `json:"external"`
	SmartStatus string  `json:"smart_status"`
}

// DiskIOStatus is aggregate disk throughput.
type DiskIOStatus struct {
	ReadRate  float64 `json:"read_rate"`  // MB/s
	WriteRate float64 `json:"write_rate"` // MB/s
}

// NetworkStatus is one network interface.
type NetworkStatus struct {
	Name      string  `json:"name"`
	RxRateMBs float64 `json:"rx_rate_mbs"`
	TxRateMBs float64 `json:"tx_rate_mbs"`
	IP        string  `json:"ip"`
}

// NetworkHistory is the interface rate history the TUI charts.
type NetworkHistory struct {
	RxHistory []float64 `json:"rx_history"`
	TxHistory []float64 `json:"tx_history"`
}

// ProxyStatus is the detected system proxy.
type ProxyStatus struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"` // HTTP, HTTPS, SOCKS, PAC, TUN
	Host    string `json:"host"`
}

// BatteryStatus is one system battery.
type BatteryStatus struct {
	Percent    float64 `json:"percent"`
	Status     string  `json:"status"`
	TimeLeft   string  `json:"time_left"`
	Health     string  `json:"health"`
	CycleCount int     `json:"cycle_count"`
	Capacity   int     `json:"capacity"`
}

// ProcessInfo is one entry of the top-processes table.
type ProcessInfo struct {
	PID         int     `json:"pid"`
	PPID        int     `json:"ppid"`
	Name        string  `json:"name"`
	Command     string  `json:"command"`
	CPU         float64 `json:"cpu"`    // percent of one logical CPU
	Memory      float64 `json:"memory"` // percent of physical memory
	MemoryBytes uint64  `json:"memory_bytes,omitempty"`
}

// ProcessWatchConfig is the process-watch setting snapshot.
type ProcessWatchConfig struct {
	Enabled      bool    `json:"enabled"`
	CPUThreshold float64 `json:"cpu_threshold"`
	Window       string  `json:"window"`
}

// ProcessAlert is one raised process-watch alert.
type ProcessAlert struct {
	PID         int       `json:"pid"`
	Name        string    `json:"name"`
	Command     string    `json:"command,omitempty"`
	CPU         float64   `json:"cpu"`
	Threshold   float64   `json:"threshold"`
	Window      string    `json:"window"`
	TriggeredAt time.Time `json:"triggered_at"`
	Status      string    `json:"status"`
}

// HistoryReport is `mo history --json`.
type HistoryReport struct {
	Logs       HistoryLogs        `json:"logs"`
	Limit      int                `json:"limit"`
	Sessions   []HistorySession   `json:"sessions"`
	Operations []HistoryOperation `json:"operations"`
}

// HistoryLogs points at the operations log the report was read from.
type HistoryLogs struct {
	Operations string `json:"operations"`
}

// HistorySession is one command run reconstructed from the operations log.
type HistorySession struct {
	Command        string         `json:"command"`
	RunID          string         `json:"run_id"`
	StartedAt      string         `json:"started_at"`
	OperationCount int            `json:"operation_count"`
	Actions        HistoryActions `json:"actions"`
}

// HistoryActions counts the outcomes of one session.
type HistoryActions struct {
	Removed     int `json:"removed"`
	Recycled    int `json:"recycled"`
	Uninstalled int `json:"uninstalled"`
	Skipped     int `json:"skipped"`
	Failed      int `json:"failed"`
	Other       int `json:"other"`
}

// HistoryOperation is one operations.log line.
type HistoryOperation struct {
	Timestamp string `json:"timestamp"`
	Command   string `json:"command"`
	RunID     string `json:"run_id"`
	Action    string `json:"action"`
	Path      string `json:"path"`
	Detail    string `json:"detail"`
}
