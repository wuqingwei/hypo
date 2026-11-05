<script setup lang="ts">
import { computed, onMounted, ref } from "vue"

import { GetUninstallList, GetUninstallSelect } from "../../bindings/github.com/wuqingwei/hypo/internal/app/service.js"
import type { UninstallApp } from "../../bindings/github.com/wuqingwei/hypo/internal/engine/models.js"
import type { UninstallRun } from "../../bindings/github.com/wuqingwei/hypo/internal/engine/models.js"
import { formatBytes } from "../lib/format"

type Phase = "list" | "confirm" | "executing" | "result" | "error"

const apps = ref<UninstallApp[]>([])
const phase = ref<Phase>("list")
const loading = ref(false)
const error = ref("")
const query = ref("")
const selected = ref(new Set<string>())
const runResult = ref<UninstallRun | null>(null)
const runError = ref("")

async function refresh(): Promise<void> {
  if (loading.value) return
  loading.value = true
  error.value = ""
  try {
    apps.value = (await GetUninstallList()) ?? []
    phase.value = "list"
    selected.value = new Set()
    runResult.value = null
  } catch (err) {
    error.value = String(err)
    phase.value = "error"
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return apps.value
  return apps.value.filter((app: UninstallApp) =>
    (app.name ?? "").toLowerCase().includes(q) ||
    (app.publisher ?? "").toLowerCase().includes(q),
  )
})

const selectedApps = computed(() => apps.value.filter((app: UninstallApp) => selected.value.has(app.id)))
const selectedBytes = computed(() =>
  selectedApps.value.reduce((sum: number, app: UninstallApp) => sum + (app.size_bytes ?? 0), 0),
)

function toggle(app: UninstallApp): void {
  const next = new Set(selected.value)
  if (next.has(app.id)) next.delete(app.id)
  else next.add(app.id)
  selected.value = next
}

function openConfirm(): void {
  phase.value = "confirm"
}

function cancelConfirm(): void {
  phase.value = "list"
}

async function execute(): Promise<void> {
  if (loading.value) return
  loading.value = true
  phase.value = "executing"
  runError.value = ""
  try {
    const run = await GetUninstallSelect(Array.from(selected.value))
    runResult.value = run
    phase.value = "result"
    // The cache the list was built from is gone after a run.
    apps.value = []
  } catch (err) {
    runError.value = String(err)
    phase.value = "error"
  } finally {
    loading.value = false
  }
}

function outcomeClass(outcome: string): string {
  if (outcome === "uninstalled") return "ok"
  if (outcome === "skipped") return "warn"
  return "bad"
}

onMounted(refresh)
</script>

<template>
  <div class="uninstall">
    <header class="head">
      <div>
        <h1>Uninstall</h1>
        <p class="hint">
          Inventory from the engine's own scan. Uninstallers run exactly as
          registered, leftovers go to the Recycle Bin only after an ownership check.
        </p>
      </div>
      <div class="head-actions">
        <input v-model="query" class="search" type="text" placeholder="Search name or publisher" />
        <button class="btn" :disabled="loading" @click="refresh">
          {{ loading && phase !== 'executing' ? "Scanning…" : "Rescan" }}
        </button>
      </div>
    </header>

    <div v-if="error || runError" class="banner error-banner">
      <span>{{ error || runError }}</span>
      <button class="btn small" @click="refresh">Retry</button>
    </div>

    <div v-if="phase === 'executing'" class="panel loading-panel">
      Uninstallers are running with the engine's own descendant tracking and
      timeout ceiling. Some vendors show their own windows; let them finish.
    </div>

    <div v-if="phase === 'result' && runResult" class="result">
      <header class="result-head">
        <h1>Uninstall finished</h1>
        <span class="incomplete-chip" :class="runResult.result.incomplete ? 'warn' : 'ok'">
          {{ runResult.result.incomplete ? "incomplete" : "complete" }}
        </span>
      </header>

      <div v-if="runResult.notices?.length" class="notice-box">
        <p v-for="(notice, index) in runResult.notices" :key="index">{{ notice }}</p>
      </div>

      <section class="group">
        <header class="group-head"><span class="group-name">Results</span></header>
        <ul class="items">
          <li v-for="row in runResult.result.results ?? []" :key="row.id" class="item">
            <span class="item-name">{{ row.name }}</span>
            <span class="item-detail" :title="row.detail">{{ row.detail }}</span>
            <span class="action-chip" :class="outcomeClass(row.outcome)">{{ row.outcome }}</span>
          </li>
          <li v-if="!(runResult.result.results ?? []).length" class="item muted">
            Nothing ran — the selection did not match any installed app.
          </li>
        </ul>
      </section>

      <p v-if="runResult.result.not_found?.length" class="notfound">
        Not found: {{ runResult.result.not_found.join(", ") }}
      </p>

      <div class="result-actions">
        <button class="btn" @click="refresh">Back to list</button>
      </div>
    </div>

    <template v-if="phase === 'list' || phase === 'confirm'">
      <div v-if="!loading && !error && !apps.length" class="panel loading-panel">
        No inventory. Rescan to collect installed applications.
      </div>

      <table v-else-if="apps.length" class="apps-table">
        <thead>
          <tr>
            <th class="col-check"></th>
            <th>Name</th>
            <th>Publisher</th>
            <th class="num">Size</th>
            <th>Source</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="app in filtered" :key="app.id" class="app-row"
            :class="{ selectable: phase === 'list', checked: selected.has(app.id) }"
            @click="phase === 'list' ? toggle(app) : null">
            <td class="col-check">
              <span class="checkbox" :class="{ checked: selected.has(app.id) }" />
            </td>
            <td class="name-cell">
              <span class="app-name">{{ app.name }}</span>
              <span v-if="app.version" class="app-version">{{ app.version }}</span>
            </td>
            <td class="pub-cell" :title="app.path ?? ''">{{ app.publisher || "—" }}</td>
            <td class="num">{{ app.size === "Unknown" ? "—" : app.size }}</td>
            <td class="src-cell">{{ app.source }}</td>
          </tr>
        </tbody>
      </table>
    </template>

    <footer v-if="phase === 'list' && apps.length" class="execute-bar">
      <span class="execute-info">
        {{ selected.size }} selected · {{ formatBytes(selectedBytes) }}
      </span>
      <button class="btn primary" :disabled="selected.size === 0" @click="openConfirm">
        Uninstall…
      </button>
    </footer>

    <div v-if="phase === 'confirm'" class="modal-backdrop">
      <div class="modal">
        <h2>Uninstall {{ selected.size }} {{ selected.size === 1 ? "application" : "applications" }}?</h2>
        <ul class="modal-list">
          <li v-for="app in selectedApps" :key="app.id">
            {{ app.name }} <span v-if="app.version" class="muted">({{ app.version }})</span>
          </li>
        </ul>
        <p class="modal-text">
          Each registered uninstaller runs once, exactly as installed. The engine rereads
          every entry right before launching it; a cancelled or failed uninstall leaves the
          app installed and is reported. Leftover files go to the Recycle Bin only when the
          engine's ownership check accepts them.
        </p>
        <div class="modal-actions">
          <button class="btn" :disabled="loading" @click="cancelConfirm">Cancel</button>
          <button class="btn primary" :disabled="loading" @click="execute">Run uninstallers</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.uninstall {
  padding: 20px 22px 76px;
  min-height: 100%;
}
.head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 16px; margin-bottom: 18px;
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

.apps-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.apps-table th {
  text-align: left; color: var(--muted); font-weight: 500; text-transform: uppercase;
  font-size: 10.5px; letter-spacing: 0.06em; padding: 6px 10px 8px 4px;
  border-bottom: 1px solid var(--border);
}
.apps-table td { padding: 7px 10px 7px 4px; border-bottom: 1px solid var(--border); }
.app-row.selectable { cursor: pointer; }
.app-row.selectable:hover { background: color-mix(in srgb, var(--ok) 5%, transparent); }
.app-row.checked { background: color-mix(in srgb, var(--ok) 8%, transparent); }
.checkbox { width: 14px; height: 14px; border: 1px solid var(--border); border-radius: 4px; display: inline-block; }
.checkbox.checked { background: var(--ok); border-color: var(--ok); }
.name-cell { min-width: 0; }
.app-name { display: inline-block; max-width: 380px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; vertical-align: bottom; }
.app-version { color: var(--muted); font-size: 11.5px; margin-left: 8px; }
.pub-cell { color: var(--muted); max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.src-cell { color: var(--muted); font-size: 12px; }
.num { text-align: right; font-variant-numeric: tabular-nums; }

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
.btn.primary { background: var(--ok); border-color: var(--ok); color: #06130a; }
.btn.primary:disabled { background: color-mix(in srgb, var(--ok) 40%, var(--card)); border-color: var(--border); color: var(--muted); }

.modal-backdrop {
  position: fixed; inset: 0; background: rgba(0, 0, 0, 0.55);
  display: flex; align-items: center; justify-content: center; z-index: 30;
  --wails-draggable: no-drag;
}
.modal {
  width: min(92vw, 560px); max-height: 80vh; overflow-y: auto;
  background: var(--card); border: 1px solid var(--border);
  border-radius: 12px; padding: 22px 24px;
  box-shadow: 0 18px 60px rgba(0, 0, 0, 0.5);
}
.modal h2 { margin: 0 0 12px; font-size: 17px; }
.modal-list {
  margin: 0 0 14px; padding: 8px 12px 8px 26px;
  background: var(--bg-raised); border: 1px solid var(--border); border-radius: 8px;
  font-size: 13px;
}
.modal-list li { margin: 3px 0; }
.modal-text { color: var(--muted); font-size: 12.5px; line-height: 1.6; margin: 0 0 16px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; }
.muted { color: var(--muted); }

.result-head { display: flex; align-items: baseline; gap: 14px; margin-bottom: 12px; }
.result-head h1 { margin: 0; font-size: 20px; }
.incomplete-chip {
  font-size: 11px; font-weight: 600; border-radius: 999px; padding: 3px 10px;
  border: 1px solid var(--border); text-transform: uppercase; letter-spacing: 0.05em;
}
.incomplete-chip.ok { color: var(--ok); border-color: color-mix(in srgb, var(--ok) 45%, var(--border)); }
.incomplete-chip.warn { color: var(--warn); border-color: color-mix(in srgb, var(--warn) 45%, var(--border)); }
.notice-box {
  border: 1px solid color-mix(in srgb, var(--warn) 45%, var(--border));
  background: color-mix(in srgb, var(--warn) 10%, var(--card));
  border-radius: var(--radius); padding: 10px 14px; margin-bottom: 14px;
}
.notice-box p { margin: 2px 0; font-size: 12.5px; color: var(--warn); }
.group {
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--card); overflow: hidden; margin-bottom: 12px;
}
.group-head { padding: 10px 14px; background: var(--bg-raised); }
.group-name { font-size: 13px; font-weight: 600; }
.items { list-style: none; margin: 0; padding: 0; }
.item {
  display: flex; align-items: center; gap: 10px; padding: 6px 14px;
  font-size: 12.5px; border-top: 1px solid var(--border); min-width: 0;
}
.item-name { flex: 0 1 auto; }
.item-detail { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--muted); }
.action-chip {
  flex: 0 0 auto; font-size: 10px; font-weight: 600; border-radius: 999px;
  padding: 2px 8px; border: 1px solid var(--border);
  text-transform: uppercase; letter-spacing: 0.05em;
}
.action-chip.ok { color: var(--ok); border-color: color-mix(in srgb, var(--ok) 45%, var(--border)); }
.action-chip.warn { color: var(--warn); border-color: color-mix(in srgb, var(--warn) 45%, var(--border)); }
.action-chip.bad { color: var(--danger); border-color: color-mix(in srgb, var(--danger) 45%, var(--border)); }
.notfound { color: var(--warn); font-size: 12.5px; }
.result-actions { margin-top: 14px; display: flex; justify-content: flex-end; }
</style>
