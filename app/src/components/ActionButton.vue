<script setup lang="ts">
import { ref } from "vue";

const props = defineProps<{
  label: string;
  variant?: "primary" | "danger" | "secondary";
  disabled?: boolean;
}>();

const emit = defineEmits<{ click: [] }>();
const loading = ref(false);

const variants: Record<string, string> = {
  primary: "bg-blue-500 hover:bg-blue-600 text-white",
  danger: "bg-red-500 hover:bg-red-600 text-white",
  secondary: "bg-gray-200 hover:bg-gray-300 text-gray-700",
};

const cls = variants[props.variant ?? "primary"];

async function handleClick() {
  if (loading.value || props.disabled) return;
  loading.value = true;
  emit("click");
  // Reset after a short delay so parent can control state
  setTimeout(() => (loading.value = false), 2000);
}
</script>

<template>
  <button
    :class="[cls, { 'opacity-50 cursor-not-allowed': disabled || loading }]"
    class="px-3 py-1 rounded text-xs font-medium transition-colors"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <span v-if="loading">…</span>
    <span v-else>{{ label }}</span>
  </button>
</template>
