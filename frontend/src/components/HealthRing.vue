<script setup lang="ts">
import { computed } from "vue"

const props = defineProps<{
  score: number | null | undefined
  message?: string
}>()

// Ring geometry: radius 52 in a 120x120 viewBox.
const circumference = computed(() => 2 * Math.PI * 52)
const dashOffset = computed(() => {
  const score = props.score ?? 0
  return circumference.value * (1 - Math.max(0, Math.min(100, score)) / 100)
})
const colourClass = computed(() => {
  if (props.score === null || props.score === undefined) return "unknown"
  if (props.score >= 80) return "good"
  if (props.score >= 55) return "warn"
  return "bad"
})
</script>

<template>
  <div class="health-ring" :class="colourClass">
    <svg viewBox="0 0 120 120" role="img" aria-label="Health score">
      <circle class="ring-track" cx="60" cy="60" r="52" />
      <circle
        class="ring-value"
        cx="60"
        cy="60"
        r="52"
        :stroke-dasharray="circumference"
        :stroke-dashoffset="dashOffset"
      />
    </svg>
    <div class="ring-label">
      <span class="ring-score">{{ score ?? "—" }}</span>
      <span class="ring-caption">health</span>
    </div>
    <p v-if="message" class="ring-message">{{ message }}</p>
  </div>
</template>

<style scoped>
.health-ring {
  position: relative;
  width: 132px;
  flex: 0 0 auto;
  text-align: center;
}
svg {
  width: 100%;
  height: auto;
  transform: rotate(-90deg);
}
.ring-track,
.ring-value {
  fill: none;
  stroke-width: 9;
  stroke-linecap: round;
}
.ring-track {
  stroke: var(--border);
}
.ring-value {
  transition: stroke-dashoffset 0.6s ease;
}
.good .ring-value {
  stroke: var(--ok);
}
.warn .ring-value {
  stroke: var(--warn);
}
.bad .ring-value {
  stroke: var(--danger);
}
.unknown .ring-value {
  stroke: var(--muted);
}
.ring-label {
  position: absolute;
  top: 44px;
  left: 0;
  right: 0;
  display: flex;
  flex-direction: column;
  line-height: 1.1;
}
.ring-score {
  font-size: 30px;
  font-weight: 700;
}
.ring-caption {
  font-size: 11px;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}
.ring-message {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--muted);
}
</style>
