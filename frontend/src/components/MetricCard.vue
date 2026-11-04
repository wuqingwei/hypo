<script setup lang="ts">
defineProps<{
  title?: string
  subtitle?: string
  span?: number
}>()
</script>

<template>
  <section class="card" :style="span ? { gridColumn: `span ${span}` } : undefined">
    <header v-if="title || $slots.actions" class="card-head">
      <div>
        <h2 class="card-title">{{ title }}</h2>
        <p v-if="subtitle" class="card-subtitle">{{ subtitle }}</p>
      </div>
      <div class="card-actions"><slot name="actions" /></div>
    </header>
    <div class="card-body"><slot /></div>
  </section>
</template>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px 18px;
}
.card-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
}
.card-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
.card-subtitle {
  margin: 2px 0 0;
  font-size: 12px;
  color: var(--muted);
  opacity: 0.8;
}
.card-body {
  flex: 1;
  min-width: 0;
}
</style>
