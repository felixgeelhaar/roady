<script setup lang="ts">
const props = defineProps<{
  value: number;
  max: number;
  label?: string;
  color?: string;
}>();

const pct = Math.min(100, Math.round((props.value / Math.max(props.max, 1)) * 100));
const barColor = props.color ?? (pct > 80 ? "bg-red-500" : pct > 50 ? "bg-yellow-500" : "bg-green-500");
</script>

<template>
  <div>
    <div v-if="label" class="flex justify-between text-xs text-gray-500 mb-0.5">
      <span>{{ label }}</span>
      <span>{{ value }} / {{ max }} ({{ pct }}%)</span>
    </div>
    <div class="h-2 bg-gray-200 rounded-full overflow-hidden">
      <div :class="barColor" class="h-full rounded-full transition-all" :style="{ width: pct + '%' }" />
    </div>
  </div>
</template>
