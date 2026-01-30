<script setup lang="ts">
const props = defineProps<{ status: string }>();

const colorMap: Record<string, string> = {
  pending: "bg-gray-200 text-gray-700",
  in_progress: "bg-blue-100 text-blue-700",
  blocked: "bg-yellow-100 text-yellow-700",
  done: "bg-green-100 text-green-700",
  verified: "bg-emerald-200 text-emerald-800",
  healthy: "bg-green-100 text-green-700",
  warning: "bg-yellow-100 text-yellow-700",
  critical: "bg-red-100 text-red-700",
  pass: "bg-green-100 text-green-700",
  fail: "bg-red-100 text-red-700",
};

const classes = colorMap[props.status] ?? "bg-gray-100 text-gray-600";
</script>

<template>
  <span :class="classes" class="inline-block px-2 py-0.5 rounded-full text-xs font-medium">
    {{ status }}
  </span>
</template>
