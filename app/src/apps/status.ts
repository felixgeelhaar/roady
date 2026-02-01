import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool, type ToolResult } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref, watch, nextTick, onMounted } from "vue";
import { renderDonut, type DonutDatum } from "@/lib/d3-charts";

const app = createApp("Roady Status");

const RoadyStatus = defineComponent({
  components: { StatusBadge },
  setup() {
    const data = ref<Record<string, unknown> | null>(null);
    const raw = ref("");
    const loading = ref(true);
    const error = ref("");
    const filter = ref<string | null>(null);
    const donutEl = ref<HTMLElement | null>(null);

    function load(result: ToolResult) {
      loading.value = false;
      const text = extractText(result);
      raw.value = text;
      try {
        data.value = JSON.parse(text);
      } catch {
        data.value = null;
      }
    }

    app.ontoolresult = (result: ToolResult) => {
      load(result);
    };

    async function refresh() {
      loading.value = true;
      load(await callTool(app, "roady_status", { json: true }));
    }

    function onDonutClick(label: string) {
      filter.value = filter.value === label ? null : label;
    }

    watch(data, async () => {
      await nextTick();
      if (!data.value || !donutEl.value) return;
      const counts = (data.value.counts ?? {}) as Record<string, number>;
      const donutData: DonutDatum[] = Object.entries(counts).map(([label, value]) => ({ label, value }));
      renderDonut(donutEl.value, donutData, { width: 200, height: 200, onClick: onDonutClick });
    });

    // Connect after Vue has mounted to avoid blocking render
    onMounted(() => {
      app.connect().catch((e: unknown) => {
        error.value = String(e);
        console.error("MCP connect failed:", e);
      });
    });

    return { data, raw, loading, error, refresh, filter, donutEl };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Project Status</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="refresh">Refresh</button>
      </div>
      <div v-if="error" class="text-red-500 text-xs mb-2">Error: {{ error }}</div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="data">
        <div class="flex gap-6 mb-4">
          <div ref="donutEl" style="position:relative;flex-shrink:0"></div>
          <div class="flex-1">
            <div class="grid grid-cols-2 gap-3">
              <div v-for="(v, k) in (data.counts || {})" :key="k"
                class="border rounded p-2 text-center cursor-pointer transition-opacity"
                :class="{ 'opacity-40': filter && filter !== k }"
                @click="filter = filter === k ? null : k">
                <div class="text-lg font-bold">{{ v }}</div>
                <div class="text-xs text-gray-500">{{ k }}</div>
              </div>
            </div>
            <div class="text-sm text-gray-600 mt-3">Total: {{ data.total_tasks }} tasks</div>
            <div v-if="filter" class="text-xs text-blue-500 mt-1 cursor-pointer" @click="filter = null">Clear filter: {{ filter }}</div>
          </div>
        </div>
        <div v-if="data.tasks && filter" class="space-y-1">
          <div v-for="t in data.tasks.filter(t => t.status === filter)" :key="t.id" class="border rounded p-2 flex items-center gap-2">
            <StatusBadge :status="t.status" />
            <span class="text-sm">{{ t.title || t.id }}</span>
          </div>
        </div>
      </template>
      <pre v-else class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
    </div>
  `,
});

createVueApp(RoadyStatus).mount("#app");
