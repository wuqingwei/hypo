// Display formatters for engine facts. These render measurements only; they
// never influence what the engine does.

export function formatBytes(bytes: number | undefined | null): string {
  if (bytes === undefined || bytes === null || Number.isNaN(bytes)) return "—"
  if (bytes < 1024) return `${Math.round(bytes)} B`
  const units = ["KB", "MB", "GB", "TB", "PB"]
  let value = bytes
  let unit = -1
  do {
    value /= 1024
    unit++
  } while (value >= 1024 && unit < units.length - 1)
  return `${value >= 100 ? Math.round(value) : value.toFixed(1)} ${units[unit]}`
}

export function formatPercent(pct: number | undefined | null): string {
  if (pct === undefined || pct === null || Number.isNaN(pct)) return "—"
  return `${pct.toFixed(1)}%`
}

// The engine reports interface rates in MB/s.
export function formatRate(mbs: number | undefined | null): string {
  if (mbs === undefined || mbs === null || Number.isNaN(mbs)) return "—"
  if (mbs >= 1) return `${mbs.toFixed(1)} MB/s`
  return `${(mbs * 1024).toFixed(0)} KB/s`
}

export function formatCount(n: number | undefined | null): string {
  if (n === undefined || n === null) return "—"
  return n.toLocaleString("en-US")
}

// Health score 0-100 → colour class used by the ring and bars.
export function healthClass(score: number | undefined | null): string {
  if (score === undefined || score === null) return "unknown"
  if (score >= 80) return "good"
  if (score >= 55) return "warn"
  return "bad"
}

export function usageClass(pct: number | undefined | null): string {
  if (pct === undefined || pct === null) return "unknown"
  if (pct >= 90) return "bad"
  if (pct >= 75) return "warn"
  return "good"
}
