<script setup lang="ts">
import { computed } from "vue"

const props = defineProps<{
  percent: number | null | undefined
  label?: string
}>()

const width = computed(() => {
  const pct = props.percent ?? 0
  return `${Math.max(0, Math.min(100, pct))}%`
})
const colourClass = computed(() => {
  const pct = props.percent
  if (pct === null || pct === undefined) return "unknown"
  if (pct >= 90) return "bad"
  if (pct >= 75) return "warn"
  return "good"
})
</script>

<template>
  <div class="percent-bar">
    <div class="track" :class="colourClass">
      <div class="fill" :style="{ width }" />
    </div>
    <span v-if="label" class="bar-label">{{ label }}</span>
  </div>
</template>

<style scoped>
.percent-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.track {
  flex: 1;
  height: 6px;
  border-radius: 999px;
  background: var(--border);
  overflow: hidden;
}
.fill {
  height: 100%;
  border-radius: 999px;
  transition: width 0.5s ease;
}
.good .fill {
  background: var(--ok);
}
.warn .fill {
  background: var(--warn);
}
.bad .fill {
  background: var(--danger);
}
.unknown .fill {
  background: var(--muted);
}
.bar-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}
</style>
