<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"

import { GetAnalyze } from "../../bindings/github.com/wuqingwei/hypo/internal/app/service.js"
import type { AnalyzeEntry, AnalyzeReport } from "../../bindings/github.com/wuqingwei/hypo/internal/engine/models.js"
import { formatBytes } from "../lib/format"

// One scan level in the drill-down history.
interface ScanLevel {
  path: string
  entries: AnalyzeEntry[]
  scanStatus: string
}

const report = ref<AnalyzeReport | null>(null)
const loading = ref(false)
const error = ref("")
// Drill-down stack; index 0 is the overview or the user's first scan.
const stack = ref<ScanLevel[]>([])
const selectedEntry = ref<AnalyzeEntry | null>(null)

const current = computed<ScanLevel | null>(() =>
  stack.value.length ? stack.value[stack.value.length - 1] : null,
)

const crumbs = computed(() => stack.value.map((level) => level.path || "Overview"))

async function scan(path: string): Promise<void> {
  if (loading.value) return
  loading.value = true
  error.value = ""
  try {
    const fetched = await GetAnalyze(path)
    report.value = fetched
    stack.value = [
      {
        path: fetched?.path ?? "",
        entries: fetched?.entries ?? [],
        scanStatus: fetched?.scan_status ?? "unknown",
      },
    ]
    selectedEntry.value = null
  } catch (err) {
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

async function drillInto(entry: AnalyzeEntry): Promise<void> {
  if (!entry.is_dir || loading.value) return
  loading.value = true
  error.value = ""
  try {
    const fetched = await GetAnalyze(entry.path)
    stack.value.push({
      path: fetched?.path ?? entry.path,
      entries: fetched?.entries ?? [],
      scanStatus: fetched?.scan_status ?? "unknown",
    })
    selectedEntry.value = null
  } catch (err) {
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

function popTo(index: number): void {
  if (index < 0 || index >= stack.value.length - 1) return
  stack.value = stack.value.slice(0, index + 1)
  selectedEntry.value = null
}

const largeFiles = computed(() => report.value?.large_files ?? [])

// ---- ECharts treemap --------------------------------------------------------
import * as echarts from "echarts"
import TreemapChart from "./TreemapChart.vue"

const chartData = computed(() =>
  (current.value?.entries ?? []).map((entry: AnalyzeEntry) => ({
    name: entry.name,
    value: Math.max(entry.size, 1),
    entry,
    itemStyle: entry.cleanable
      ? { color: "#2ea04366", borderColor: "#2ea043" }
      : undefined,
  })),
)

function onDrill(params: { name: string }): void {
  const entry = (current.value?.entries ?? []).find((e: AnalyzeEntry) => e.name === params.name)
  if (entry?.is_dir) void drillInto(entry)
}

// ---- selection panel --------------------------------------------------------
const detailRows = computed(() => {
  const entry = selectedEntry.value
  if (!entry) return []
  return [
    { label: "Path", value: entry.path },
    { label: "Size", value: formatBytes(entry.size) },
    { label: "Kind", value: entry.is_dir ? "directory" : "file" },
    { label: "Scan", value: String(entry.scan_status) },
    ...(entry.last_access ? [{ label: "Last access", value: entry.last_access }] : []),
    ...(entry.cleanable ? [{ label: "Cleanable", value: "the engine tags this as cleanup content" }] : []),
    ...(entry.insight ? [{ label: "Insight", value: "the engine marked this row as notable" }] : []),
  ]
})

onMounted(() => void scan(""))

onBeforeUnmount(() => {
  // nothing persistent; chart disposes inside its own component
})
</script>

<template>
  <div class="analyze">
    <header class="head">
      <div>
        <h1>Analyze</h1>
        <p class="hint">
          Read-only disk usage from the engine's scanner. Double-click a directory
          to drill in; green rows are content the engine tags as cleanable.
        </p>
      </div>
      <div class="head-actions">
        <button class="btn" :disabled="loading" @click="scan(current?.path ?? '')">
          {{ loading ? "Scanning…" : "Rescan" }}
        </button>
      </div>
    </header>

    <nav v-if="crumbs.length > 1" class="crumbs">
      <button v-for="(crumb, index) in crumbs" :key="index" class="crumb"
        :class="{ active: index === crumbs.length - 1 }" @click="popTo(index)">
        {{ crumb }}
      </button>
    </nav>

    <div v-if="error" class="banner error-banner">
      <span>{{ error }}</span>
      <button class="btn small" @click="scan(current?.path ?? '')">Retry</button>
    </div>

    <div v-if="loading" class="panel loading-panel">
      The engine is walking the tree. Large directories can take a while; the
      scan reports partial coverage rather than blocking on every file.
    </div>

    <template v-if="current && !loading">
      <div class="chart-wrap">
        <TreemapChart :data="chartData" @drill="onDrill"
          @select="(entry) => (selectedEntry = entry as AnalyzeEntry)" />
      </div>

      <section v-if="selectedEntry" class="detail">
        <header class="group-head">
          <span class="group-name">{{ selectedEntry.name }}</span>
          <button class="btn small" @click="selectedEntry = null">Close</button>
        </header>
        <dl class="detail-rows">
          <div v-for="row in detailRows" :key="row.label" class="detail-row">
            <dt>{{ row.label }}</dt>
            <dd :title="row.value">{{ row.value }}</dd>
          </div>
        </dl>
        <button v-if="selectedEntry.is_dir" class="btn" @click="drillInto(selectedEntry)">
          Open directory
        </button>
      </section>

      <section v-if="largeFiles.length" class="group">
        <header class="group-head">
          <span class="group-name">Large files</span>
          <span class="group-meta">{{ largeFiles.length }} files ≥ the engine's large-file threshold</span>
        </header>
        <ul class="items">
          <li v-for="file in largeFiles" :key="file.path" class="item">
            <span class="item-path" :title="file.path">{{ file.path }}</span>
            <span class="item-size">{{ formatBytes(file.size) }}</span>
          </li>
        </ul>
      </section>
    </template>
  </div>
</template>

<style scoped>
.analyze { padding: 20px 22px 28px; min-height: 100%; }
.head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 16px; margin-bottom: 14px;
}
.head h1 { margin: 0 0 4px; font-size: 20px; }
.hint { margin: 0; color: var(--muted); font-size: 12.5px; max-width: 560px; }
.head-actions { display: flex; align-items: center; gap: 10px; flex: 0 0 auto; }

.crumbs {
  display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 12px;
}
.crumb {
  border: 1px solid var(--border); background: var(--card); color: var(--muted);
  border-radius: 999px; padding: 3px 12px; font-size: 12px; cursor: pointer;
  max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.crumb.active { color: var(--text); border-color: var(--muted); }

.banner {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  border-radius: var(--radius); padding: 10px 14px; margin-bottom: 16px; font-size: 13px;
}
.error-banner {
  border: 1px solid color-mix(in srgb, var(--danger) 45%, var(--border));
  background: color-mix(in srgb, var(--danger) 10%, var(--card));
  color: var(--danger); white-space: pre-wrap;
}
.loading-panel { color: var(--muted); text-align: center; padding: 40px 26px; }

.chart-wrap {
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--card); overflow: hidden; margin-bottom: 14px;
}

.detail, .group {
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--card); overflow: hidden; margin-bottom: 14px;
}
.group-head {
  display: flex; justify-content: space-between; align-items: center; gap: 12px;
  padding: 10px 14px; background: var(--bg-raised);
}
.group-name { font-size: 13px; font-weight: 600; }
.group-meta { font-size: 12px; color: var(--muted); }
.detail-rows { margin: 0; padding: 8px 14px; }
.detail-row {
  display: flex; gap: 14px; padding: 4px 0; font-size: 12.5px; min-width: 0;
}
.detail-row dt { color: var(--muted); flex: 0 0 90px; }
.detail-row dd {
  margin: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  direction: rtl; text-align: left; flex: 1;
}
.detail .btn { margin: 0 14px 12px; }

.items { list-style: none; margin: 0; padding: 0; }
.item {
  display: flex; align-items: center; gap: 10px; padding: 6px 14px;
  font-size: 12.5px; border-top: 1px solid var(--border); min-width: 0;
}
.item-path {
  flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  direction: rtl; text-align: left;
}
.item-size { flex: 0 0 auto; color: var(--muted); font-variant-numeric: tabular-nums; }
</style>
