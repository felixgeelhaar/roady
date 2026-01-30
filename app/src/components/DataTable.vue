<script setup lang="ts">
defineProps<{
  columns: { key: string; label: string }[];
  rows: Record<string, unknown>[];
}>();
</script>

<template>
  <div class="overflow-x-auto">
    <table class="w-full text-sm border-collapse">
      <thead>
        <tr class="border-b border-gray-200">
          <th
            v-for="col in columns"
            :key="col.key"
            class="text-left py-1.5 px-2 font-medium text-gray-600"
          >
            {{ col.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(row, i) in rows"
          :key="i"
          class="border-b border-gray-100 hover:bg-gray-50"
        >
          <td v-for="col in columns" :key="col.key" class="py-1.5 px-2">
            <slot :name="col.key" :value="row[col.key]" :row="row">
              {{ row[col.key] }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
