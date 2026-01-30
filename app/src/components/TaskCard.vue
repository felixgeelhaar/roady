<script setup lang="ts">
import StatusBadge from "./StatusBadge.vue";

defineProps<{
  id: string;
  title: string;
  status: string;
  priority?: string;
  dependsOn?: string[];
}>();

defineEmits<{
  action: [event: string, taskId: string];
}>();
</script>

<template>
  <div class="border rounded-lg p-3 mb-2 hover:bg-gray-50">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <StatusBadge :status="status" />
        <span class="font-medium">{{ title }}</span>
      </div>
      <span class="text-xs text-gray-400 font-mono">{{ id }}</span>
    </div>
    <div v-if="priority || (dependsOn && dependsOn.length)" class="mt-1.5 flex gap-3 text-xs text-gray-500">
      <span v-if="priority">Priority: {{ priority }}</span>
      <span v-if="dependsOn?.length">Deps: {{ dependsOn.join(", ") }}</span>
    </div>
    <div class="mt-2 flex gap-1.5">
      <button
        v-if="status === 'pending'"
        class="px-2 py-0.5 text-xs bg-blue-500 text-white rounded hover:bg-blue-600"
        @click="$emit('action', 'start', id)"
      >
        Start
      </button>
      <button
        v-if="status === 'in_progress'"
        class="px-2 py-0.5 text-xs bg-green-500 text-white rounded hover:bg-green-600"
        @click="$emit('action', 'complete', id)"
      >
        Complete
      </button>
      <button
        v-if="status === 'in_progress'"
        class="px-2 py-0.5 text-xs bg-yellow-500 text-white rounded hover:bg-yellow-600"
        @click="$emit('action', 'block', id)"
      >
        Block
      </button>
    </div>
  </div>
</template>
