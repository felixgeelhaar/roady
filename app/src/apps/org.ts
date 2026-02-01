import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderDonut, renderHorizontalBars, type DonutDatum, type BarDatum } from "@/lib/d3-charts";

const app = createApp("Roady Org");

interface ProjectStatus {
  name: string;
  total: number;
  done: number;
  in_progress: number;
  pending: number;
  blocked: number;
}

interface OrgData {
  projects?: ProjectStatus[];
  [key: string]: unknown;
}

const RoadyOrg = defineComponent({
  components: { StatusBadge },
  setup() {
    const data = ref<OrgData | null>(null);
    const raw = ref("");
    const loading = ref(true);
    const donutEl = ref<HTMLElement | null>(null);
    const barsEl = ref<HTMLElement | null>(null);

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
      parse(extractText(await callTool(app, "roady_org_status")));
      loading.value = false;
    }

    watch(data, async () => {
      await nextTick();
      if (!data.value?.projects?.length) return;

      const projects = data.value.projects;

      // Aggregate status counts across all projects for donut
      if (donutEl.value) {
        const counts: Record<string, number> = {};
        for (const p of projects) {
          counts["done"] = (counts["done"] ?? 0) + (p.done ?? 0);
          counts["in_progress"] = (counts["in_progress"] ?? 0) + (p.in_progress ?? 0);
          counts["pending"] = (counts["pending"] ?? 0) + (p.pending ?? 0);
          counts["blocked"] = (counts["blocked"] ?? 0) + (p.blocked ?? 0);
        }
        const donutData: DonutDatum[] = Object.entries(counts)
          .filter(([, v]) => v > 0)
          .map(([label, value]) => ({ label, value }));
        renderDonut(donutEl.value, donutData, { width: 200, height: 200 });
      }

      // Horizontal bars showing progress per project
      if (barsEl.value) {
        const barData: BarDatum[] = projects.map((p) => ({
          label: p.name,
          value: p.total > 0 ? Math.round((p.done / p.total) * 100) : 0,
          tooltip: `<strong>${p.name}</strong>: ${p.done}/${p.total} tasks done`,
        }));
        renderHorizontalBars(barsEl.value, barData, { width: 400 });
      }
    });

    app.connect();

    return { data, raw, loading, refresh, donutEl, barsEl };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Organization Status</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="refresh">Refresh</button>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="data && data.projects">
        <div class="flex gap-6 mb-4">
          <div ref="donutEl" style="position:relative;flex-shrink:0"></div>
          <div class="flex-1">
            <div class="text-sm font-medium text-gray-700 mb-2">Progress by Project (%)</div>
            <div ref="barsEl" style="position:relative"></div>
          </div>
        </div>
      </template>
      <pre v-else class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
    </div>
  `,
});

createVueApp(RoadyOrg).mount("#app");
