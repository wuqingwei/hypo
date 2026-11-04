<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue"

import {
  GetEngineInfo,
  GetHistory,
  GetStatus,
  RefreshEngine,
} from "../bindings/github.com/wuqingwei/hypo/internal/app/service.js"
import type { EngineInfo } from "../bindings/github.com/wuqingwei/hypo/internal/app/models.js"
import type { HistoryReport, StatusSnapshot } from "../bindings/github.com/wuqingwei/hypo/internal/engine/models.js"

import HealthRing from "./components/HealthRing.vue"
import MetricCard from "./components/MetricCard.vue"
import PercentBar from "./components/PercentBar.vue"
import {
  formatBytes,
  formatCount,
  formatPercent,
  formatRate,
  usageClass,
} from "./lib/format"

const POLL_INTERVAL_MS = 5000

const engine = ref<EngineInfo | null>(null)
const snapshot = ref<StatusSnapshot | null>(null)
const history = ref<HistoryReport | null>(null)

const statusError = ref("")
const historyError = ref("")
const refreshing = ref(false)
const autoRefresh = ref(true)
const lastUpdated = ref<Date | null>(null)

let pollTimer: number | undefined

const engineReady = computed(() => engine.value?.found === true)

async function loadEngineInfo(): Promise<void> {
  try {
    engine.value = await GetEngineInfo()
  } catch (err) {
    engine.value = { found: false, error: String(err) }
  }
}

async function retryEngine(): Promise<void> {
  refreshing.value = true
  try {
    engine.value = await RefreshEngine()
    if (engine.value.found) {
      statusError.value = ""
      await Promise.all([refreshStatus(), refreshHistory()])
    }
  } finally {
    refreshing.value = false
  }
}

async function refreshStatus(): Promise<void> {
  try {
    snapshot.value = await GetStatus()
    statusError.value = ""
    lastUpdated.value = new Date()
  } catch (err) {
    statusError.value = String(err)
  }
}

async function refreshHistory(): Promise<void> {
  try {
    history.value = await GetHistory(10)
    historyError.value = ""
  } catch (err) {
    historyError.value = String(err)
  }
}

async function refreshAll(): Promise<void> {
  if (refreshing.value) return
  refreshing.value = true
  try {
    if (!engine.value) await loadEngineInfo()
    if (!engineReady.value) return
    await Promise.all([refreshStatus(), refreshHistory()])
  } finally {
    refreshing.value = false
  }
}

function startPolling(): void {
  stopPolling()
  pollTimer = window.setInterval(() => {
    if (!autoRefresh.value || document.hidden || !engineReady.value) return
    void refreshStatus()
  }, POLL_INTERVAL_MS)
}

function stopPolling(): void {
  if (pollTimer !== undefined) {
    window.clearInterval(pollTimer)
    pollTimer = undefined
  }
}

function onVisibility(): void {
  // One immediate catch-up after returning to the window.
  if (!document.hidden && autoRefresh.value && engineReady.value) void refreshStatus()
}

const recentOperations = computed(() => history.value?.operations ?? [])
const topProcesses = computed(() => (snapshot.value?.top_processes ?? []).slice(0, 6))

onMounted(() => {
  void refreshAll().then(() => startPolling())
  document.addEventListener("visibilitychange", onVisibility)
})

onUnmounted(() => {
  stopPolling()
  document.removeEventListener("visibilitychange", onVisibility)
})
</script>

<template>
  <div class="app">
    <header class="topbar">
      <div class="brand">
        <span class="brand-name">Hypo</span>
        <span class="brand-tag">community</span>
      </div>

      <div v-if="engineReady" class="engine-chip" :title="engine?.root">
        <span class="chip-dot" />
        Mole engine <strong>{{ engine?.version || "unknown version" }}</strong>
        <span class="chip-source">({{ engine?.source }})</span>
      </div>
      <div v-else class="engine-chip missing">
        <span class="chip-dot" />
        no engine
      </div>

      <div class="spacer" />

      <label class="auto-toggle" title="Refresh metrics every 5 seconds">
        <input v-model="autoRefresh" type="checkbox" />
        <span>auto</span>
      </label>
      <button class="btn" :disabled="refreshing" @click="refreshAll">
        {{ refreshing ? "Refreshing…" : "Refresh" }}
      </button>
    </header>

    <main class="content">
      <div v-if="engine && !engine.found" class="panel error-panel">
        <h2>Mole engine not found</h2>
        <p class="error-text">{{ engine.error }}</p>
        <button class="btn" :disabled="refreshing" @click="retryEngine">Retry</button>
      </div>

      <template v-else>
        <div v-if="statusError" class="banner error-banner">
          <span>{{ statusError }}</span>
          <button class="btn small" @click="refreshAll">Retry</button>
        </div>

        <div v-if="!snapshot && !statusError" class="panel loading-panel">
          Collecting machine metrics…
        </div>

        <div v-if="snapshot" class="grid">
          <MetricCard title="System" :span="2" class="system-card">
            <div class="system-row">
              <HealthRing :score="snapshot.health_score" :message="snapshot.health_score_msg" />
              <dl class="facts">
                <div class="fact"><dt>Host</dt><dd>{{ snapshot.host }}</dd></div>
                <div class="fact"><dt>OS</dt><dd>{{ snapshot.hardware.os_version || snapshot.platform }}</dd></div>
                <div class="fact"><dt>Uptime</dt><dd>{{ snapshot.uptime }}</dd></div>
                <div class="fact"><dt>CPU</dt><dd>{{ snapshot.hardware.cpu_model }}</dd></div>
                <div class="fact"><dt>RAM</dt><dd>{{ snapshot.hardware.total_ram }}</dd></div>
                <div class="fact"><dt>Processes</dt><dd>{{ formatCount(snapshot.procs) }}</dd></div>
              </dl>
            </div>
          </MetricCard>

          <MetricCard title="CPU" :subtitle="`${snapshot.cpu.core_count} cores · ${snapshot.cpu.logical_cpu} logical`">
            <div class="big-metric" :class="usageClass(snapshot.cpu.usage)">
              {{ formatPercent(snapshot.cpu.usage) }}
            </div>
            <PercentBar :percent="snapshot.cpu.usage" />
            <div v-if="snapshot.cpu.per_core?.length" class="core-grid">
              <div v-for="(usage, index) in snapshot.cpu.per_core" :key="index" class="core-cell" :title="`Core ${index}`">
                <PercentBar :percent="usage" />
              </div>
            </div>
          </MetricCard>

          <MetricCard title="Memory" :subtitle="snapshot.hardware.total_ram">
            <div class="big-metric" :class="usageClass(snapshot.memory.used_percent)">
              {{ formatPercent(snapshot.memory.used_percent) }}
            </div>
            <PercentBar :percent="snapshot.memory.used_percent" />
            <dl class="facts compact">
              <div class="fact"><dt>Used</dt><dd>{{ formatBytes(snapshot.memory.used) }}</dd></div>
              <div class="fact"><dt>Total</dt><dd>{{ formatBytes(snapshot.memory.total) }}</dd></div>
              <div class="fact"><dt>Swap</dt><dd>{{ formatBytes(snapshot.memory.swap_used) }} / {{ formatBytes(snapshot.memory.swap_total) }}</dd></div>
            </dl>
          </MetricCard>

          <MetricCard v-for="disk in snapshot.disks" :key="disk.mount" :title="`Disk ${disk.mount}`"
            :subtitle="disk.external ? 'external' : disk.fstype">
            <div class="big-metric" :class="usageClass(disk.used_percent)">
              {{ formatPercent(disk.used_percent) }}
            </div>
            <PercentBar :percent="disk.used_percent" />
            <dl class="facts compact">
              <div class="fact"><dt>Used</dt><dd>{{ formatBytes(disk.used) }}</dd></div>
              <div class="fact"><dt>Total</dt><dd>{{ formatBytes(disk.total) }}</dd></div>
              <div v-if="disk.smart_status" class="fact"><dt>SMART</dt><dd>{{ disk.smart_status }}</dd></div>
            </dl>
          </MetricCard>

          <MetricCard title="Disk I/O">
            <dl class="facts compact">
              <div class="fact">
                <dt>Read</dt><dd>{{ formatRate(snapshot.disk_io.read_rate) }}</dd>
              </div>
              <div class="fact">
                <dt>Write</dt><dd>{{ formatRate(snapshot.disk_io.write_rate) }}</dd>
              </div>
            </dl>
          </MetricCard>

          <MetricCard v-for="(battery, index) in snapshot.batteries" :key="index"
            :title="'Battery' + ((snapshot.batteries?.length ?? 0) > 1 ? ' ' + (index + 1) : '')"
            :subtitle="battery.status">
            <div class="big-metric">{{ formatPercent(battery.percent) }}</div>
            <PercentBar :percent="battery.percent" />
            <dl class="facts compact">
              <div class="fact"><dt>Health</dt><dd>{{ battery.health || "—" }}</dd></div>
              <div class="fact"><dt>Time left</dt><dd>{{ battery.time_left || "—" }}</dd></div>
            </dl>
          </MetricCard>

          <MetricCard v-if="topProcesses.length" title="Top processes" :span="2">
            <table class="mini-table">
              <thead>
                <tr><th>Name</th><th class="num">CPU</th><th class="num">Memory</th></tr>
              </thead>
              <tbody>
                <tr v-for="proc in topProcesses" :key="proc.pid">
                  <td class="proc-name" :title="proc.command">{{ proc.name }}</td>
                  <td class="num">{{ formatPercent(proc.cpu) }}</td>
                  <td class="num">{{ formatBytes(proc.memory_bytes) }}</td>
                </tr>
              </tbody>
            </table>
          </MetricCard>

          <MetricCard title="Recent operations" :span="3"
            :subtitle="history ? `${history.operations?.length ?? 0} of ${history.limit} entries` : ''">
            <p v-if="historyError" class="error-text">{{ historyError }}</p>
            <table v-else-if="recentOperations.length" class="mini-table ops-table">
              <thead>
                <tr><th>When</th><th>Action</th><th>Command</th><th>Path</th></tr>
              </thead>
              <tbody>
                <tr v-for="(op, index) in recentOperations" :key="index">
                  <td class="muted">{{ op.timestamp }}</td>
                  <td><span class="action-chip" :class="op.action">{{ op.action }}</span></td>
                  <td class="muted">{{ op.command }}</td>
                  <td class="path-cell" :title="op.detail || op.path">{{ op.path }}</td>
                </tr>
              </tbody>
            </table>
            <p v-else class="muted">
              No recorded operations yet. Everything the engine removes shows up here.
            </p>
          </MetricCard>
        </div>
      </template>
    </main>

    <footer class="footer">
      <span>GPL-3.0 · community project, not affiliated with Mole</span>
      <span v-if="lastUpdated">updated {{ lastUpdated.toLocaleTimeString() }}</span>
    </footer>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.topbar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 22px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-raised);
  --wails-draggable: drag;
}
.brand {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.brand-name {
  font-size: 19px;
  font-weight: 700;
  letter-spacing: -0.01em;
}
.brand-tag {
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--muted);
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 2px 8px;
}
.engine-chip {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: 12.5px;
  color: var(--muted);
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 4px 12px;
  max-width: 46vw;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.engine-chip strong {
  color: var(--text);
  font-weight: 600;
}
.engine-chip .chip-source {
  opacity: 0.75;
}
.chip-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--ok);
  flex: 0 0 auto;
}
.engine-chip.missing .chip-dot {
  background: var(--danger);
}
.spacer {
  flex: 1;
}
.auto-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: var(--muted);
  cursor: pointer;
  --wails-draggable: no-drag;
}
.auto-toggle input {
  accent-color: var(--ok);
}

.content {
  flex: 1;
  padding: 20px 22px 28px;
  overflow-y: auto;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.panel {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--card);
  padding: 26px;
}
.error-panel {
  max-width: 640px;
  margin: 40px auto;
  text-align: center;
  border-color: color-mix(in srgb, var(--danger) 40%, var(--border));
}
.error-panel h2 {
  margin: 0 0 12px;
  font-size: 17px;
}
.error-text {
  color: var(--danger);
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  text-align: left;
}
.error-panel .error-text {
  text-align: center;
}
.loading-panel {
  margin: 40px auto;
  color: var(--muted);
  max-width: 360px;
  text-align: center;
}

.banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-radius: var(--radius);
  padding: 10px 14px;
  margin-bottom: 16px;
  font-size: 13px;
}
.error-banner {
  border: 1px solid color-mix(in srgb, var(--danger) 45%, var(--border));
  background: color-mix(in srgb, var(--danger) 10%, var(--card));
  color: var(--danger);
  white-space: pre-wrap;
}

.system-row {
  display: flex;
  gap: 24px;
  align-items: center;
  flex-wrap: wrap;
}
.facts {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 8px 22px;
  flex: 1;
  min-width: 240px;
}
.facts.compact {
  grid-template-columns: 1fr;
  margin-top: 12px;
}
.fact {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
  font-size: 13px;
}
.fact dt {
  color: var(--muted);
  flex: 0 0 auto;
}
.fact dd {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: right;
}

.big-metric {
  font-size: 30px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  margin-bottom: 8px;
}
.big-metric.good { color: var(--ok); }
.big-metric.warn { color: var(--warn); }
.big-metric.bad { color: var(--danger); }

.core-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(56px, 1fr));
  gap: 8px 10px;
  margin-top: 14px;
}
.core-cell :deep(.bar-label) {
  display: none;
}

.mini-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12.5px;
}
.mini-table th {
  text-align: left;
  color: var(--muted);
  font-weight: 500;
  text-transform: uppercase;
  font-size: 10.5px;
  letter-spacing: 0.06em;
  padding: 4px 10px 8px 0;
  border-bottom: 1px solid var(--border);
}
.mini-table td {
  padding: 6px 10px 6px 0;
  border-bottom: 1px solid var(--border);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 320px;
}
.mini-table tr:last-child td {
  border-bottom: none;
}
.mini-table .num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.proc-name {
  max-width: 220px;
}
.path-cell {
  direction: rtl;
  text-align: left;
  max-width: 420px;
}
.muted {
  color: var(--muted);
}

.action-chip {
  display: inline-block;
  font-size: 10.5px;
  font-weight: 600;
  border-radius: 999px;
  padding: 2px 9px;
  border: 1px solid var(--border);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.action-chip.removed { color: var(--ok); border-color: color-mix(in srgb, var(--ok) 45%, var(--border)); }
.action-chip.recycled { color: var(--ok); border-color: color-mix(in srgb, var(--ok) 45%, var(--border)); }
.action-chip.failed { color: var(--danger); border-color: color-mix(in srgb, var(--danger) 45%, var(--border)); }
.action-chip.skipped { color: var(--warn); border-color: color-mix(in srgb, var(--warn) 45%, var(--border)); }

.footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 9px 22px;
  border-top: 1px solid var(--border);
  color: var(--muted);
  font-size: 11.5px;
}
</style>
