import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderDonut, type DonutDatum } from "@/lib/d3-charts";

const app = createApp("Roady Sync");

interface SyncData {
  synced?: number;
  failed?: number;
  skipped?: number;
  results?: Array<{ task_id: string; status: string; message?: string }>;
  [key: string]: unknown;
}

const RoadySync = defineComponent({
  components: { StatusBadge },
  setup() {
    const data = ref<SyncData | null>(null);
    const loading = ref(true);
    const raw = ref("");
    const donutEl = ref<HTMLElement | null>(null);

    function parse(text: string) {
      raw.value = text;
      try {
        data.value = JSON.parse(text);
      } catch {
        data.value = null;
      }
    }

    app.ontoolresult = (result) => {
      loading.value = false;
      parse(result.content?.find((c) => c.type === "text")?.text ?? "");
    };

    async function refresh() {
      loading.value = true;
      parse(extractText(await callTool(app, "roady_sync", { plugin_path: "" })));
      loading.value = false;
    }

    watch(data, async () => {
      await nextTick();
      if (!data.value || !donutEl.value) return;

      // Build donut from sync result counts
      const counts: Record<string, number> = {};
      if (data.value.synced != null) counts["synced"] = data.value.synced;
      if (data.value.failed != null) counts["failed"] = data.value.failed;
      if (data.value.skipped != null) counts["skipped"] = data.value.skipped;

      // Fallback: count from results array
      if (!Object.keys(counts).length && data.value.results) {
        for (const r of data.value.results) {
          const s = r.status ?? "unknown";
          counts[s] = (counts[s] ?? 0) + 1;
        }
      }

      const donutData: DonutDatum[] = Object.entries(counts)
        .filter(([, v]) => v > 0)
        .map(([label, value]) => ({ label, value }));

      if (donutData.length) {
        renderDonut(donutEl.value, donutData, { width: 180, height: 180 });
      }
    });

    app.connect();

    return { data, loading, raw, refresh, donutEl };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Plugin Sync</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="refresh">Sync</button>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="data">
        <div class="flex gap-6 mb-4">
          <div ref="donutEl" style="position:relative;flex-shrink:0"></div>
          <div class="flex-1">
            <div v-if="data.results" class="space-y-1">
              <div v-for="r in (data.results as any[])" :key="r.task_id" class="border rounded p-2 flex items-center gap-2">
                <StatusBadge :status="r.status === 'synced' ? 'done' : r.status === 'failed' ? 'blocked' : 'pending'" />
                <span class="text-sm">{{ r.task_id }}</span>
                <span v-if="r.message" class="text-xs text-gray-400 ml-auto">{{ r.message }}</span>
              </div>
            </div>
          </div>
        </div>
        <pre v-if="!data.results" class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
      </template>
      <pre v-else class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
    </div>
  `,
});

createVueApp(RoadySync).mount("#app");
