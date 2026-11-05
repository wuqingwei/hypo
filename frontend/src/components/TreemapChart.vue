<script setup lang="ts">
// A thin ECharts treemap wrapper: the parent owns the data and reacts to
// drill/select events; this component owns the chart instance lifecycle.
import { onBeforeUnmount, onMounted, ref, watch } from "vue"

import * as echarts from "echarts"

const props = defineProps<{
  data: Array<{
    name: string
    value: number
    entry: unknown
    itemStyle?: { color: string; borderColor: string }
  }>
}>()

const emit = defineEmits<{
  drill: [params: { name: string }]
  select: [entry: unknown]
}>()

const host = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null

function render(): void {
  if (!chart) return
  chart.setOption({
    series: [
      {
        type: "treemap",
        data: props.data,
        roam: false,
        nodeClick: false, // drill is emitted to the parent, which rescans
        breadcrumb: { show: false },
        width: "100%",
        height: "88%",
        top: 8,
        bottom: 24,
        left: 8,
        right: 8,
        itemStyle: { borderColor: "#0d1117", borderWidth: 1, gapWidth: 1 },
        label: {
          show: true,
          overflow: "truncate",
          fontSize: 11,
          color: "#e6edf3",
        },
        upperLabel: { show: false },
        levels: [
          {
            color: ["#1f6feb", "#238636", "#9e6a03", "#8957e5", "#bf3989", "#2986a6", "#b35b3d", "#5a7d2a"],
            colorMappingBy: "index",
          },
        ],
      },
    ],
    tooltip: {
      confine: true,
      textStyle: { color: "#e6edf3", fontSize: 11 },
      backgroundColor: "#161b22",
      borderColor: "#2d333b",
      formatter: (params: { name: string; value: number }) =>
        `${params.name}<br/>${formatTooltipBytes(params.value)}`,
    },
  })
}

function formatTooltipBytes(bytes: number): string {
  if (!Number.isFinite(bytes)) return "—"
  if (bytes < 1024) return `${Math.round(bytes)} B`
  const units = ["KB", "MB", "GB", "TB", "PB"]
  let value = bytes
  let unit = -1
  do {
    value /= 1024
    unit++
  } while (value >= 1024 && unit < units.length - 1)
  return `${value >= 100 ? Math.round(value) : value.toFixed(1)} ${units[unit]}`
}

function onResize(): void {
  chart?.resize()
}

onMounted(() => {
  if (!host.value) return
  chart = echarts.init(host.value)
  render()
  chart.on("click", (params: echarts.ECElementEvent) => {
    const data = params.data as { entry?: unknown; name?: string } | undefined
    if (data?.entry !== undefined) emit("select", data.entry)
    if (params?.name) emit("drill", { name: params.name })
  })
  window.addEventListener("resize", onResize)
})

watch(
  () => props.data,
  () => render(),
  { deep: false },
)

onBeforeUnmount(() => {
  window.removeEventListener("resize", onResize)
  chart?.dispose()
  chart = null
})
</script>

<template>
  <div ref="host" class="treemap-host" />
</template>

<style scoped>
.treemap-host {
  width: 100%;
  height: 420px;
}
</style>
