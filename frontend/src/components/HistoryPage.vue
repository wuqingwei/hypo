<script setup lang="ts">
import { computed, onMounted, ref } from "vue"

import { GetHistory } from "../../bindings/github.com/wuqingwei/hypo/internal/app/service.js"
import type { HistoryOperation, HistoryReport, HistorySession } from "../../bindings/github.com/wuqingwei/hypo/internal/engine/models.js"

const report = ref<HistoryReport | null>(null)
const loading = ref(false)
const error = ref("")
const query = ref("")
const actionFilter = ref("all")
const limit = ref(200)

const ACTIONS = ["removed", "recycled", "uninstalled", "skipped", "failed", "owner", "other"]

async function refresh(): Promise<void> {
  if (loading.value) return
  loading.value = true
  error.value = ""
  try {
    report.value = await GetHistory(limit.value)
  } catch (err) {
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

const operations = computed<HistoryOperation[]>(() => {
  const all = report.value?.operations ?? []
  const q = query.value.trim().toLowerCase()
  return all.filter((op: HistoryOperation) => {
    if (actionFilter.value !== "all" && op.action.toLowerCase() !== actionFilter.value) return false
    if (!q) return true
    return (
      op.path.toLowerCase().includes(q) ||
      op.command.toLowerCase().includes(q) ||
      (op.detail ?? "").toLowerCase().includes(q)
    )
  })
})

// Group rows by run (command + run id), newest first as the engine reports.
const sessions = computed(() => {
  const byRun = new Map<string, { session: HistorySession | null; rows: HistoryOperation[] }>()
  const known = new Set(
    (report.value?.sessions ?? []).map(
      (s: HistorySession) => `${s.command}|${s.run_id ?? ""}`,
    ),
  )
  for (const op of operations.value) {
    const key = `${op.command}|${op.run_id ?? ""}`
    if (!byRun.has(key)) {
      byRun.set(key, {
        session:
          (report.value?.sessions ?? []).find(
            (s: HistorySession) => `${s.command}|${s.run_id ?? ""}` === key,
          ) ?? null,
        rows: [],
      })
    }
    byRun.get(key)!.rows.push(op)
  }
  // Sessions the limit trimmed out of the operations list still appear as
  // headers when they carry counts, so the history reads like a timeline.
  const groups = Array.from(byRun.values())
  for (const session of report.value?.sessions ?? []) {
    const key = `${session.command}|${session.run_id ?? ""}`
    if (!byRun.has(key) && known.has(key)) {
      groups.push({ session, rows: [] })
    }
  }
  return groups
})

function actionClass(action: string): string {
  const a = action.toLowerCase()
  if (a === "removed" || a === "recycled" || a === "uninstalled") return "ok"
  if (a === "skipped" || a === "owner") return "warn"
  if (a === "failed") return "bad"
  return "muted"
}

function sessionSummary(session: HistorySession | null, rows: HistoryOperation[]): string {
  if (session) {
    const parts: string[] = []
    const a = session.actions
    if (a.removed) parts.push(`${a.removed} removed`)
    if (a.recycled) parts.push(`${a.recycled} recycled`)
    if (a.uninstalled) parts.push(`${a.uninstalled} uninstalled`)
    if (a.skipped) parts.push(`${a.skipped} skipped`)
    if (a.failed) parts.push(`${a.failed} failed`)
    if (a.other) parts.push(`${a.other} other`)
    if (parts.length) return parts.join(" · ")
  }
  if (rows.length) return `${rows.length} rows`
  return "no rows in this page"
}

onMounted(refresh)
</script>

<template>
  <div class="history">
    <header class="head">
      <div>
        <h1>History</h1>
        <p class="hint">
          Everything the engine changed, from its operations log — what ran,
          which path, and whether it went to the Recycle Bin.
        </p>
      </div>
      <div class="head-actions">
        <input v-model="query" class="search" type="text" placeholder="Filter path / command / detail" />
        <select v-model="actionFilter" class="action-select">
          <option value="all">All actions</option>
          <option v-for="action in ACTIONS" :key="action" :value="action">{{ action }}</option>
        </select>
        <button class="btn" :disabled="loading" @click="refresh">
          {{ loading ? "Loading…" : "Reload" }}
        </button>
      </div>
    </header>

    <div v-if="error" class="banner error-banner">
      <span>{{ error }}</span>
      <button class="btn small" @click="refresh">Retry</button>
    </div>

    <div v-if="!loading && !error && !operations.length && !(report?.operations ?? []).length"
      class="panel loading-panel">
      No recorded operations yet.
    </div>
    <div v-else-if="!loading && !operations.length" class="panel loading-panel">
      No rows match the current filter.
    </div>

    <section v-for="(group, index) in sessions" :key="index" class="group">
      <header class="group-head">
        <span class="group-name">{{ group.session?.command ?? group.rows[0]?.command ?? "run" }}</span>
        <span class="group-meta">
          {{ group.session?.started_at || group.rows[0]?.timestamp || "" }}
          · {{ sessionSummary(group.session, group.rows) }}
        </span>
      </header>
      <ul class="items">
        <li v-for="(op, rowIndex) in group.rows" :key="rowIndex" class="item">
          <span class="item-time">{{ op.timestamp }}</span>
          <span class="action-chip" :class="actionClass(op.action)">{{ op.action }}</span>
          <span class="item-path" :title="op.detail || op.path">{{ op.path }}</span>
          <span v-if="op.detail" class="item-detail" :title="op.detail">{{ op.detail }}</span>
        </li>
        <li v-if="!group.rows.length" class="item muted">
          Rows are outside this page's limit — reload with a higher limit to see them.
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.history { padding: 20px 22px 28px; min-height: 100%; }
.head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 16px; margin-bottom: 16px; flex-wrap: wrap;
}
.head h1 { margin: 0 0 4px; font-size: 20px; }
.hint { margin: 0; color: var(--muted); font-size: 12.5px; max-width: 560px; }
.head-actions { display: flex; align-items: center; gap: 10px; flex: 0 0 auto; }
.search {
  background: var(--card); border: 1px solid var(--border); color: var(--text);
  border-radius: 8px; padding: 6px 12px; font-size: 13px; width: 240px;
  --wails-draggable: no-drag;
}
.search:focus { outline: none; border-color: var(--muted); }
.action-select {
  background: var(--card); border: 1px solid var(--border); color: var(--text);
  border-radius: 8px; padding: 6px 10px; font-size: 13px;
  --wails-draggable: no-drag;
}

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

.group {
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--card); overflow: hidden; margin-bottom: 12px;
}
.group-head {
  display: flex; justify-content: space-between; align-items: baseline; gap: 12px;
  padding: 9px 14px; background: var(--bg-raised);
}
.group-name { font-size: 13px; font-weight: 600; }
.group-meta { font-size: 12px; color: var(--muted); font-variant-numeric: tabular-nums; }
.items { list-style: none; margin: 0; padding: 0; }
.item {
  display: flex; align-items: center; gap: 10px; padding: 5px 14px;
  font-size: 12.5px; border-top: 1px solid var(--border); min-width: 0;
}
.item-time { color: var(--muted); font-size: 11.5px; flex: 0 0 auto; font-variant-numeric: tabular-nums; }
.item-path {
  flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  direction: rtl; text-align: left;
}
.item-detail {
  flex: 0 1 auto; max-width: 200px; overflow: hidden; text-overflow: ellipsis;
  white-space: nowrap; color: var(--muted); font-size: 11.5px;
}
.action-chip {
  flex: 0 0 auto; font-size: 10px; font-weight: 600; border-radius: 999px;
  padding: 2px 8px; border: 1px solid var(--border);
  text-transform: uppercase; letter-spacing: 0.05em;
}
.action-chip.ok { color: var(--ok); border-color: color-mix(in srgb, var(--ok) 45%, var(--border)); }
.action-chip.warn { color: var(--warn); border-color: color-mix(in srgb, var(--warn) 45%, var(--border)); }
.action-chip.bad { color: var(--danger); border-color: color-mix(in srgb, var(--danger) 45%, var(--border)); }
.action-chip.muted { color: var(--muted); }
</style>
