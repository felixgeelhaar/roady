<script setup lang="ts">
import StatusBadge from "./StatusBadge.vue";

export interface DAGNode {
  id: string;
  title: string;
  status: string;
  dependsOn?: string[];
}

defineProps<{ nodes: DAGNode[] }>();
</script>

<template>
  <div class="space-y-1">
    <div v-for="node in nodes" :key="node.id" class="flex items-center gap-2 py-1 pl-2 border-l-2" :class="{
      'border-green-400': node.status === 'done' || node.status === 'verified',
      'border-blue-400': node.status === 'in_progress',
      'border-yellow-400': node.status === 'blocked',
      'border-gray-300': node.status === 'pending',
    }">
      <StatusBadge :status="node.status" />
      <span class="text-sm">{{ node.title }}</span>
      <span v-if="node.dependsOn?.length" class="text-xs text-gray-400">
        ← {{ node.dependsOn.join(", ") }}
      </span>
    </div>
  </div>
</template>
