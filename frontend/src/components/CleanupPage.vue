<script setup lang="ts">
import { computed, onMounted, ref } from "vue"

import { GetCleanPreview, GetCleanRun } from "../../bindings/github.com/wuqingwei/hypo/internal/app/service.js"
import type { CleanRecord, CleanReport } from "../../bindings/github.com/wuqingwei/hypo/internal/engine/models.js"
import { formatBytes } from "../lib/format"

type Phase = "preview" | "confirm" | "executing" | "result" | "error"

const report = ref<CleanReport | null>(null)
const runReport = ref<CleanReport | null>(null)
const phase = ref<Phase>("preview")
const loading = ref(false)
const error = ref("")
const selected = ref(new Set<number>())
const runError = ref("")

const REMOVABLE = ["would_remove", "would_recycle"]
const EXECUTE_ACTIONS = ["removed", "recycled"]

function isRemovable(record: CleanRecord): boolean {
  return REMOVABLE.includes(record.action)
}

async function refresh(): Promise<void> {
  if (loading.value) return
  loading.value = true
  error.value = ""
  try {
    const fetched = await GetCleanPreview()
    report.value = fetched
    runReport.value = null
    runError.value = ""
    phase.value = "preview"
    selected.value = new Set(
      (fetched?.records ?? [])
        .map((record: CleanRecord, index: number) => (isRemovable(record) ? index : -1))
        .filter((index: number) => index >= 0),
    )
  } catch (err) {
    error.value = String(err)
    report.value = null
    phase.value = "error"
  } finally {
    loading.value = false
  }
}

// Preview groups preserve the engine's order; each group is one description.
const groups = computed(() => {
  const byDescription = new Map<string, { description: string; items: Array<{ record: CleanRecord; index: number }> }>()
  ;(report.value?.records ?? []).forEach((record: CleanRecord, index: number) => {
    const key = record.description || record.path
    if (!byDescription.has(key)) byDescription.set(key, { description: key, items: [] })
    byDescription.get(key)!.items.push({ record, index })
  })
  return Array.from(byDescription.values())
})

const previewBytes = computed(() => report.value?.totals?.size_bytes ?? 0)
const removableCount = computed(
  () => (report.value?.records ?? []).filter((record: CleanRecord) => isRemovable(record)).length,
)

// Result reconciliation: how the run compares with what was previewed.
const outcome = computed(() => {
  const run = runReport.value
  if (!run) return null
  const removed = (run.records ?? []).filter((record: CleanRecord) => EXECUTE_ACTIONS.includes(record.action))
  const skipped = (run.records ?? []).filter((record: CleanRecord) => record.action === "skipped")
  const failed = (run.records ?? []).filter((record: CleanRecord) => record.action === "failed")
  const removedBytes = removed.reduce((sum: number, record: CleanRecord) => sum + (record.size_bytes ?? 0), 0)
  const previewed = report.value?.totals?.size_bytes ?? 0
  const runTotal = run.totals?.size_bytes ?? 0
  return { removed, skipped, failed, removedBytes, previewed, runTotal }
})

function toggle(index: number): void {
  const next = new Set(selected.value)
  if (next.has(index)) next.delete(index)
  else next.add(index)
  selected.value = next
}

function toggleGroup(items: Array<{ record: CleanRecord; index: number }>): void {
  const removable = items.filter((item) => isRemovable(item.record))
  const allSelected = removable.every((item) => selected.value.has(item.index))
  const next = new Set(selected.value)
  removable.forEach((item) => {
    if (allSelected) next.delete(item.index)
    else next.add(item.index)
  })
  selected.value = next
}

function openConfirm(): void {
  phase.value = "confirm"
}

function cancelConfirm(): void {
  phase.value = "preview"
}

async function execute(): Promise<void> {
  if (loading.value) return
  loading.value = true
  phase.value = "executing"
  runError.value = ""
  try {
    // The engine re-enumerates and re-runs every guard now; the preview's
    // selection is intent the engine re-derives, not a file list.
    const result = await GetCleanRun()
    runReport.value = result
    phase.value = "result"
  } catch (err) {
    runError.value = String(err)
    phase.value = "error"
  } finally {
    loading.value = false
  }
}

function backToPreview(): void {
  void refresh()
}

function actionClass(action: string): string {
  if (REMOVABLE.includes(action) || EXECUTE_ACTIONS.includes(action)) return "ok"
  if (action === "skipped") return "warn"
  if (action === "failed") return "bad"
  return "muted"
}

onMounted(refresh)
</script>

<template>
  <div class="cleanup">
    <header class="cleanup-head">
      <div>
        <h1>Cleanup</h1>
        <p class="hint">
          Preview from the engine's dry run. Execution re-runs the engine's own
          guards — your selection is intent, not a file list.
        </p>
      </div>
      <div class="head-actions">
        <span v-if="phase === 'preview' && report" class="preview-total">
          {{ formatBytes(previewBytes) }} · {{ removableCount }} removable
        </span>
        <button class="btn" :disabled="loading" @click="refresh">
          {{ loading && phase !== 'executing' ? "Scanning…" : "Rescan" }}
        </button>
      </div>
    </header>

    <div v-if="error" class="banner error-banner">
      <span>{{ error || runError }}</span>
      <button class="btn small" @click="refresh">Retry</button>
    </div>

    <div v-if="phase === 'preview' || phase === 'confirm'" class="groups">
      <section v-for="group in groups" :key="group.description" class="group"
        :class="{ dimmed: phase === 'confirm' && !group.items.some((item) => selected.has(item.index)) }">
        <header class="group-head" @click="phase === 'preview' ? toggleGroup(group.items) : null">
          <span class="group-name">{{ group.description }}</span>
          <span class="group-meta">{{ group.items.length }} items · {{ formatBytes(group.items.reduce((sum, item) => sum + (item.record.size_bytes ?? 0), 0)) }}</span>
        </header>
        <ul class="items">
          <li v-for="item in group.items" :key="item.index" class="item"
            :class="{ selectable: phase === 'preview' && isRemovable(item.record) }"
            @click="phase === 'preview' && isRemovable(item.record) ? toggle(item.index) : null">
            <span v-if="isRemovable(item.record)" class="checkbox"
              :class="{ checked: selected.has(item.index), fixed: phase === 'confirm' }" />
            <span class="item-path" :title="item.record.path">{{ item.record.path }}</span>
            <span class="item-size">{{ formatBytes(item.record.size_bytes) }}</span>
            <span class="action-chip" :class="actionClass(item.record.action)">{{ item.record.action }}</span>
          </li>
        </ul>
      </section>
    </div>

    <div v-if="phase === 'preview' && !loading && !error && (!report || removableCount === 0)" class="panel loading-panel">
      {{ report ? "The engine found nothing to clean — system is already tidy." : "No preview yet." }}
    </div>

    <div v-if="phase === 'executing'" class="panel loading-panel">
      The engine is cleaning. Files that fail their final guards are skipped and reported.
    </div>

    <div v-if="phase === 'result' && outcome" class="result">
      <header class="result-head">
        <h1>Cleanup complete</h1>
        <div class="result-totals">
          <div class="result-number ok">{{ outcome.removed.length }}</div>
          <div class="result-label">removed</div>
          <div class="result-number warn">{{ outcome.skipped.length }}</div>
          <div class="result-label">skipped</div>
          <div v-if="outcome.failed.length" class="result-number bad">{{ outcome.failed.length }}</div>
          <div v-if="outcome.failed.length" class="result-label">failed</div>
          <div class="result-number ok">{{ formatBytes(outcome.runTotal) }}</div>
          <div class="result-label">freed (engine accounting)</div>
        </div>
      </header>

      <p v-if="outcome.previewed && outcome.runTotal < outcome.previewed" class="reconcile warn">
        Previewed {{ formatBytes(outcome.previewed) }}, freed {{ formatBytes(outcome.runTotal) }}: some targets
        changed or were held while the run re-checked them. Skipped rows below say why.
      </p>
      <p v-else class="reconcile ok">
        Freed the previewed space ({{ formatBytes(outcome.runTotal) }}).
      </p>

      <section v-for="section in [
        { title: 'Removed', records: outcome.removed, cls: 'ok' },
        { title: 'Skipped', records: outcome.skipped, cls: 'warn' },
        { title: 'Failed', records: outcome.failed, cls: 'bad' },
      ]" :key="section.title" class="group">
        <template v-if="section.records.length">
          <header class="group-head"><span class="group-name">{{ section.title }}</span></header>
          <ul class="items">
            <li v-for="(record, index) in section.records" :key="index" class="item">
              <span class="item-path" :title="record.path">{{ record.path }}</span>
              <span class="item-size">{{ formatBytes(record.size_bytes) }}</span>
              <span class="action-chip" :class="section.cls">{{ record.action }}</span>
              <span v-if="record.reason" class="item-reason" :title="record.reason">{{ record.reason }}</span>
            </li>
          </ul>
        </template>
      </section>

      <div class="result-actions">
        <button class="btn" @click="backToPreview">Back to preview</button>
      </div>
    </div>

    <footer v-if="phase === 'preview' && report && removableCount > 0" class="execute-bar">
      <span class="execute-info">
        {{ formatBytes(previewBytes) }} · {{ selected.size }} of {{ removableCount }} selected
      </span>
      <button class="btn primary" :disabled="selected.size === 0" @click="openConfirm">
        Clean now…
      </button>
    </footer>

    <div v-if="phase === 'confirm'" class="modal-backdrop">
      <div class="modal">
        <h2>Run the cleanup?</h2>
        <p class="modal-text">
          The engine will run the full cleanup now and re-check every target at removal
          time: locked or changed targets are skipped, not forced. The engine's own
          guards decide what is removed — your selection is intent.
        </p>
        <dl class="modal-facts">
          <div class="fact"><dt>Previewed space</dt><dd>{{ formatBytes(previewBytes) }}</dd></div>
          <div class="fact"><dt>Removable targets</dt><dd>{{ removableCount }}</dd></div>
          <div class="fact"><dt>Recorded in</dt><dd>operations.log</dd></div>
        </dl>
        <p class="modal-note">
          Every removal is written to the operations log and recoverable files go to the
          Recycle Bin where the engine decides to recycle.
        </p>
        <div class="modal-actions">
          <button class="btn" :disabled="loading" @click="cancelConfirm">Cancel</button>
          <button class="btn primary" :disabled="loading" @click="execute">Run cleanup</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cleanup {
  padding: 20px 22px 76px;
  min-height: 100%;
}
.cleanup-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}
.cleanup-head h1 { margin: 0 0 4px; font-size: 20px; }
.hint { margin: 0; color: var(--muted); font-size: 12.5px; max-width: 560px; }
.head-actions { display: flex; align-items: center; gap: 12px; flex: 0 0 auto; }
.preview-total { font-size: 13px; color: var(--text); font-variant-numeric: tabular-nums; }

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

.groups { display: flex; flex-direction: column; gap: 14px; }
.group {
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--card); overflow: hidden;
}
.group.dimmed { opacity: 0.55; }
.group-head {
  display: flex; justify-content: space-between; align-items: baseline; gap: 12px;
  padding: 10px 14px; background: var(--bg-raised);
}
.group-head[style], .group-head { cursor: pointer; }
.group-name { font-size: 13px; font-weight: 600; }
.group-meta { font-size: 12px; color: var(--muted); font-variant-numeric: tabular-nums; }
.items { list-style: none; margin: 0; padding: 0; }
.item {
  display: flex; align-items: center; gap: 10px; padding: 6px 14px;
  font-size: 12.5px; border-top: 1px solid var(--border); min-width: 0;
}
.item.selectable { cursor: pointer; }
.item.selectable:hover { background: color-mix(in srgb, var(--ok) 5%, transparent); }
.checkbox {
  width: 14px; height: 14px; border: 1px solid var(--border); border-radius: 4px; flex: 0 0 auto;
}
.checkbox.checked { background: var(--ok); border-color: var(--ok); }
.checkbox.fixed { opacity: 0.6; }
.item-path {
  flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  direction: rtl; text-align: left;
}
.item-size { flex: 0 0 auto; color: var(--muted); font-variant-numeric: tabular-nums; min-width: 64px; text-align: right; }
.item-reason {
  flex: 0 1 auto; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  color: var(--muted); font-size: 11.5px; max-width: 220px;
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

.execute-bar {
  position: fixed; left: 0; right: 0; bottom: 0;
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
  padding: 12px 22px;
  background: color-mix(in srgb, var(--bg-raised) 92%, transparent);
  border-top: 1px solid var(--border);
  backdrop-filter: blur(8px);
  --wails-draggable: no-drag;
}
.execute-info { font-size: 13px; color: var(--muted); font-variant-numeric: tabular-nums; }

.btn.primary {
  background: var(--ok); border-color: var(--ok); color: #06130a;
}
.btn.primary:disabled { background: color-mix(in srgb, var(--ok) 40%, var(--card)); border-color: var(--border); color: var(--muted); }

.modal-backdrop {
  position: fixed; inset: 0; background: rgba(0, 0, 0, 0.55);
  display: flex; align-items: center; justify-content: center; z-index: 30;
  --wails-draggable: no-drag;
}
.modal {
  width: min(92vw, 560px); background: var(--card); border: 1px solid var(--border);
  border-radius: 12px; padding: 22px 24px;
  box-shadow: 0 18px 60px rgba(0, 0, 0, 0.5);
}
.modal h2 { margin: 0 0 10px; font-size: 17px; }
.modal-text { color: var(--muted); font-size: 13px; line-height: 1.6; margin: 0 0 14px; }
.modal-facts { margin: 0 0 14px; display: grid; grid-template-columns: 1fr 1fr; gap: 8px 20px; }
.modal-facts .fact { display: flex; justify-content: space-between; font-size: 12.5px; }
.modal-facts dt { color: var(--muted); }
.modal-facts dd { margin: 0; font-variant-numeric: tabular-nums; }
.modal-note { color: var(--muted); font-size: 11.5px; margin: 0 0 16px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; }

.result-head { display: flex; flex-wrap: wrap; align-items: baseline; gap: 18px; margin-bottom: 10px; }
.result-head h1 { margin: 0; font-size: 20px; }
.result-totals { display: flex; align-items: baseline; gap: 6px 14px; flex-wrap: wrap; }
.result-number { font-size: 22px; font-weight: 700; font-variant-numeric: tabular-nums; }
.result-number.ok { color: var(--ok); }
.result-number.warn { color: var(--warn); }
.result-number.bad { color: var(--danger); }
.result-label { font-size: 11px; color: var(--muted); margin-right: 8px; }
.reconcile { font-size: 12.5px; margin: 0 0 16px; }
.reconcile.ok { color: var(--ok); }
.reconcile.warn { color: var(--warn); }
.result-actions { margin-top: 16px; display: flex; justify-content: flex-end; }
</style>
