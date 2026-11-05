<script setup lang="ts">
import { onMounted, ref } from "vue"

import {
  AddWhitelistEntry,
  GetWhitelist,
  RemoveWhitelistEntry,
} from "../../bindings/github.com/wuqingwei/hypo/internal/app/service.js"
import type { Whitelist } from "../../bindings/github.com/wuqingwei/hypo/internal/engine/models.js"

const wl = ref<Whitelist | null>(null)
const loading = ref(false)
const error = ref("")
const newPattern = ref("")
const busy = ref(false)

async function refresh(): Promise<void> {
  if (loading.value) return
  loading.value = true
  error.value = ""
  try {
    wl.value = await GetWhitelist()
  } catch (err) {
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

async function add(): Promise<void> {
  const pattern = newPattern.value.trim()
  if (!pattern || busy.value) return
  busy.value = true
  error.value = ""
  try {
    wl.value = await AddWhitelistEntry(pattern)
    newPattern.value = ""
  } catch (err) {
    error.value = String(err)
  } finally {
    busy.value = false
  }
}

async function remove(pattern: string): Promise<void> {
  if (busy.value) return
  busy.value = true
  error.value = ""
  try {
    wl.value = await RemoveWhitelistEntry(pattern)
  } catch (err) {
    error.value = String(err)
  } finally {
    busy.value = false
  }
}

onMounted(refresh)
</script>

<template>
  <div class="whitelist">
    <header class="head">
      <div>
        <h1>Whitelist</h1>
        <p class="hint">
          Paths here are never cleaned. The engine re-reads this file on every
          run; the built-in rows below ship with the engine and are read-only.
          Wildcards (*) are supported.
        </p>
      </div>
      <div class="head-actions">
        <button class="btn" :disabled="loading" @click="refresh">
          {{ loading ? "Loading…" : "Reload" }}
        </button>
      </div>
    </header>

    <div v-if="error" class="banner error-banner">
      <span>{{ error }}</span>
    </div>

    <section class="group">
      <header class="group-head">
        <span class="group-name">Your protected paths</span>
        <span class="group-meta">{{ wl?.user_patterns?.length ?? 0 }} patterns</span>
      </header>

      <div class="add-row">
        <input v-model="newPattern" class="pattern-input" type="text"
          placeholder="C:\Users\me\ImportantProject  or  C:\Users\*\AppData\Local\MyApp"
          @keydown.enter="add" />
        <button class="btn primary" :disabled="busy || !newPattern.trim()" @click="add">Add</button>
      </div>
      <p class="file-note" v-if="wl?.path">{{ wl.path }}</p>

      <ul class="items">
        <li v-for="pattern in wl?.user_patterns ?? []" :key="pattern" class="item">
          <span class="item-path" :title="pattern">{{ pattern }}</span>
          <button class="btn small danger" :disabled="busy" @click="remove(pattern)">Remove</button>
        </li>
        <li v-if="!(wl?.user_patterns ?? []).length" class="item muted">
          Nothing yet — add a path above to protect it from every cleanup.
        </li>
      </ul>
    </section>

    <section class="group">
      <header class="group-head">
        <span class="group-name">Built-in engine protections</span>
        <span class="group-meta">read-only · {{ wl?.defaults?.length ?? 0 }} patterns</span>
      </header>
      <p v-if="wl?.error" class="warn-note">{{ wl.error }}</p>
      <ul class="items">
        <li v-for="pattern in wl?.defaults ?? []" :key="pattern" class="item">
          <span class="item-path" :title="pattern">{{ pattern }}</span>
        </li>
        <li v-if="!(wl?.defaults ?? []).length" class="item muted">
          The engine did not report its built-in list this run.
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.whitelist { padding: 20px 22px 28px; min-height: 100%; }
.head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 16px; margin-bottom: 16px;
}
.head h1 { margin: 0 0 4px; font-size: 20px; }
.hint { margin: 0; color: var(--muted); font-size: 12.5px; max-width: 560px; }
.head-actions { display: flex; align-items: center; flex: 0 0 auto; }

.banner {
  border-radius: var(--radius); padding: 10px 14px; margin-bottom: 14px;
  font-size: 13px;
}
.error-banner {
  border: 1px solid color-mix(in srgb, var(--danger) 45%, var(--border));
  background: color-mix(in srgb, var(--danger) 10%, var(--card));
  color: var(--danger); white-space: pre-wrap;
}

.group {
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--card); overflow: hidden; margin-bottom: 14px;
}
.group-head {
  display: flex; justify-content: space-between; align-items: baseline; gap: 12px;
  padding: 10px 14px; background: var(--bg-raised);
}
.group-name { font-size: 13px; font-weight: 600; }
.group-meta { font-size: 12px; color: var(--muted); }

.add-row {
  display: flex; gap: 10px; padding: 12px 14px 6px;
}
.pattern-input {
  flex: 1; background: var(--bg-raised); border: 1px solid var(--border);
  color: var(--text); border-radius: 8px; padding: 7px 12px; font-size: 13px;
  --wails-draggable: no-drag;
}
.pattern-input:focus { outline: none; border-color: var(--muted); }
.btn.primary { background: var(--ok); border-color: var(--ok); color: #06130a; }
.btn.primary:disabled { background: color-mix(in srgb, var(--ok) 40%, var(--card)); border-color: var(--border); color: var(--muted); }
.btn.danger:hover:not(:disabled) { border-color: var(--danger); color: var(--danger); }
.file-note {
  margin: 0; padding: 0 14px 8px; color: var(--muted); font-size: 11px;
  font-family: Consolas, monospace;
}
.warn-note {
  margin: 0; padding: 8px 14px; color: var(--warn); font-size: 12px;
  border-top: 1px solid var(--border);
}

.items { list-style: none; margin: 0; padding: 0; }
.item {
  display: flex; align-items: center; gap: 10px; padding: 6px 14px;
  font-size: 12.5px; border-top: 1px solid var(--border); min-width: 0;
}
.item-path {
  flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  font-family: Consolas, monospace; font-size: 12px;
}
.muted { color: var(--muted); }
</style>
