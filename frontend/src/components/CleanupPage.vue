<script setup lang="ts">
import { computed, onMounted, ref } from "vue"

import { GetCleanPreview } from "../../bindings/github.com/wuqingwei/hypo/internal/app/service.js"
import type { CleanRecord, CleanReport } from "../../bindings/github.com/wuqingwei/hypo/internal/engine/models.js"
import { formatBytes } from "../lib/format"

const report = ref<CleanReport | null>(null)
const loading = ref(false)
const error = ref("")
const selected = ref(new Set<number>())

async function refresh(): Promise<void> {
  if (loading.value) return
  loading.value = true
  error.value = ""
  try {
    const fetched = await GetCleanPreview()
    report.value = fetched
    selected.value = new Set(
      (fetched?.records ?? [])
        .map((record: CleanRecord, index: number) =>
          record.action === "would_remove" || record.action === "would_recycle" ? index : -1)
        .filter((index: number) => index >= 0),
    )
  } catch (err) {
    error.value = String(err)
    report.value = null
  } finally {
    loading.value = false
  }
}

// Groups preserve the engine's order; each group is one description.
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
const selectedCount = computed(() => selected.value.size)

function toggle(index: number): void {
  const next = new Set(selected.value)
  if (next.has(index)) next.delete(index)
  else next.add(index)
  selected.value = next
}

function toggleGroup(items: Array<{ index: number }>): void {
  const removable = items.filter((item) => {
    const action = report.value?.records?.[item.index]?.action
    return action === "would_remove" || action === "would_recycle"
  })
  const allSelected = removable.every((item) => selected.value.has(item.index))
  const next = new Set(selected.value)
  removable.forEach((item) => {
    if (allSelected) next.delete(item.index)
    else next.add(item.index)
  })
  selected.value = next
}

function actionClass(action: string): string {
  if (action === "would_remove" || action === "would_recycle") return "ok"
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
          Preview from the engine's dry run. Nothing is deleted from this page yet —
          execution arrives with explicit confirmation in a later build.
        </p>
      </div>
      <div class="head-actions">
        <span v-if="report" class="preview-total">
          {{ formatBytes(previewBytes) }} · {{ selectedCount }} selected
        </span>
        <button class="btn" :disabled="loading" @click="refresh">
          {{ loading ? "Scanning…" : "Rescan" }}
        </button>
      </div>
    </header>

    <div v-if="error" class="banner error-banner">
      <span>{{ error }}</span>
      <button class="btn small" @click="refresh">Retry</button>
    </div>

    <div v-if="loading && !report" class="panel loading-panel">
      The engine is walking every cleanup target. This can take a while on a full disk.
    </div>

    <div v-if="report" class="groups">
      <section v-for="group in groups" :key="group.description" class="group">
        <header class="group-head" @click="toggleGroup(group.items)">
          <span class="group-name">{{ group.description }}</span>
          <span class="group-meta">{{ group.items.length }} items · {{ formatBytes(group.items.reduce((sum, item) => sum + (item.record.size_bytes ?? 0), 0)) }}</span>
        </header>
        <ul class="items">
          <li v-for="item in group.items" :key="item.index"
            class="item" :class="{ selectable: item.record.action === 'would_remove' || item.record.action === 'would_recycle' }"
            @click="item.record.action === 'would_remove' || item.record.action === 'would_recycle' ? toggle(item.index) : null">
            <span v-if="item.record.action === 'would_remove' || item.record.action === 'would_recycle'"
              class="checkbox" :class="{ checked: selected.has(item.index) }" />
            <span class="item-path" :title="item.record.path">{{ item.record.path }}</span>
            <span class="item-size">{{ formatBytes(item.record.size_bytes) }}</span>
            <span class="action-chip" :class="actionClass(item.record.action)">{{ item.record.action }}</span>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<style scoped>
.cleanup {
  padding: 20px 22px 28px;
  overflow-y: auto;
  height: 100%;
}
.cleanup-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}
.cleanup-head h1 {
  margin: 0 0 4px;
  font-size: 20px;
}
.hint {
  margin: 0;
  color: var(--muted);
  font-size: 12.5px;
  max-width: 560px;
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 0 0 auto;
}
.preview-total {
  font-size: 13px;
  color: var(--text);
  font-variant-numeric: tabular-nums;
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

.loading-panel {
  color: var(--muted);
  text-align: center;
  padding: 40px 26px;
}

.groups {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.group {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--card);
  overflow: hidden;
}
.group-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
  padding: 10px 14px;
  cursor: pointer;
  background: var(--bg-raised);
}
.group-name {
  font-size: 13px;
  font-weight: 600;
}
.group-meta {
  font-size: 12px;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}
.items {
  list-style: none;
  margin: 0;
  padding: 0;
}
.item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 14px;
  font-size: 12.5px;
  border-top: 1px solid var(--border);
  min-width: 0;
}
.item.selectable {
  cursor: pointer;
}
.item.selectable:hover {
  background: color-mix(in srgb, var(--ok) 5%, transparent);
}
.checkbox {
  width: 14px;
  height: 14px;
  border: 1px solid var(--border);
  border-radius: 4px;
  flex: 0 0 auto;
}
.checkbox.checked {
  background: var(--ok);
  border-color: var(--ok);
}
.item-path {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  direction: rtl;
  text-align: left;
}
.item-size {
  flex: 0 0 auto;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
  min-width: 64px;
  text-align: right;
}
.action-chip {
  flex: 0 0 auto;
  font-size: 10px;
  font-weight: 600;
  border-radius: 999px;
  padding: 2px 8px;
  border: 1px solid var(--border);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.action-chip.ok { color: var(--ok); border-color: color-mix(in srgb, var(--ok) 45%, var(--border)); }
.action-chip.warn { color: var(--warn); border-color: color-mix(in srgb, var(--warn) 45%, var(--border)); }
.action-chip.bad { color: var(--danger); border-color: color-mix(in srgb, var(--danger) 45%, var(--border)); }
.action-chip.muted { color: var(--muted); }
</style>
