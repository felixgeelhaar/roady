import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderGauge, renderDonut, renderHorizontalBars, renderLineChart, type DonutDatum, type BarDatum, type LinePoint } from "@/lib/d3-charts";

const app = createApp("Roady Debt");

const RoadyDebt = defineComponent({
  components: { StatusBadge },
  setup() {
    const data = ref<Record<string, unknown> | null>(null);
    const loading = ref(true);
    const raw = ref("");
    const activeTab = ref<"summary" | "report" | "sticky" | "trend">("summary");
    const chartEl = ref<HTMLElement | null>(null);
    const chart2El = ref<HTMLElement | null>(null);

    app.ontoolresult = (result) => {
      loading.value = false;
      const text = result.content?.find((c) => c.type === "text")?.text ?? "";
      raw.value = text;
      try { data.value = JSON.parse(text); } catch { /* raw */ }
    };

    async function loadTab(tab: "summary" | "report" | "sticky" | "trend") {
      activeTab.value = tab;
      loading.value = true;
      const tools: Record<string, string> = {
        report: "roady_debt_report",
        summary: "roady_debt_summary",
        sticky: "roady_sticky_drift",
        trend: "roady_debt_trend",
      };
      const r = await callTool(app, tools[tab], tab === "trend" ? { days: 30 } : {});
      const text = extractText(r);
      raw.value = text;
      try { data.value = JSON.parse(text); } catch { data.value = null; }
      loading.value = false;
    }

    watch([data, activeTab], async () => {
      await nextTick();
      if (!data.value) return;
      const d = data.value as any;
      const tab = activeTab.value;

      if (tab === "summary" && chartEl.value) {
        const score = d.average_score ?? d.health_score ?? 0;
        const max = 100;
        renderGauge(chartEl.value, score, {
          max,
          label: d.health_level ?? "health",
          thresholds: [
            { value: 0.3, color: "#22c55e" },
            { value: 0.6, color: "#eab308" },
            { value: 0.8, color: "#f97316" },
            { value: 1.0, color: "#ef4444" },
          ],
        });
      }

      if (tab === "report" && chartEl.value) {
        const byCategory = d.by_category ?? d.categories ?? {};
        const donutData: DonutDatum[] = Object.entries(byCategory).map(([label, value]) => ({
          label,
          value: value as number,
        }));
        renderDonut(chartEl.value, donutData, { width: 180, height: 180 });
      }

      if (tab === "report" && chart2El.value) {
        const scores = d.scores ?? d.components ?? {};
        const bars: BarDatum[] = Object.entries(scores)
          .map(([label, value]) => ({ label, value: value as number }))
          .sort((a, b) => b.value - a.value)
          .slice(0, 10);
        renderHorizontalBars(chart2El.value, bars, { width: 380 });
      }

      if (tab === "trend" && chartEl.value) {
        const points: LinePoint[] = [];
        if (d.current_period !== undefined && d.previous_period !== undefined) {
          points.push({ x: "Previous", y: d.previous_period });
          points.push({ x: "Current", y: d.current_period });
        }
        if (d.data_points && Array.isArray(d.data_points)) {
          for (const p of d.data_points) {
            points.push({ x: p.label ?? p.date ?? points.length, y: p.value ?? p.score ?? 0 });
          }
        }
        if (points.length >= 2) {
          renderLineChart(chartEl.value, points, { width: 400, height: 180, yLabel: "Debt Score" });
        }
      }
    });

    app.connect();

    return { data, loading, raw, activeTab, loadTab, chartEl, chart2El };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Technical Debt</h1>
      </div>
      <div class="flex gap-1 mb-3">
        <button v-for="tab in ['summary', 'report', 'sticky', 'trend']" :key="tab"
          :class="activeTab === tab ? 'bg-blue-500 text-white' : 'bg-gray-200'"
          class="px-3 py-1 text-xs rounded hover:opacity-80"
          @click="loadTab(tab as any)">
          {{ tab }}
        </button>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="data">
        <!-- Summary tab: gauge + metrics -->
        <template v-if="activeTab === 'summary'">
          <div class="flex gap-6 items-start mb-3">
            <div ref="chartEl" style="position:relative"></div>
            <div class="grid grid-cols-2 gap-3 flex-1">
              <div class="border rounded p-2 text-center" v-if="(data as any).total_items != null">
                <div class="text-lg font-bold">{{ (data as any).total_items }}</div>
                <div class="text-xs text-gray-500">Total Items</div>
              </div>
              <div class="border rounded p-2 text-center" v-if="(data as any).sticky_items != null">
                <div class="text-lg font-bold">{{ (data as any).sticky_items }}</div>
                <div class="text-xs text-gray-500">Sticky Items</div>
              </div>
              <div class="border rounded p-2 text-center" v-if="(data as any).health_level">
                <StatusBadge :status="(data as any).health_level" />
                <div class="text-xs text-gray-500 mt-1">Health Level</div>
              </div>
            </div>
          </div>
        </template>

        <!-- Report tab: donut + bar -->
        <template v-else-if="activeTab === 'report'">
          <div class="flex gap-6 items-start mb-3">
            <div ref="chartEl" style="position:relative"></div>
            <div ref="chart2El" style="position:relative;flex:1"></div>
          </div>
        </template>

        <!-- Trend tab: line chart -->
        <template v-else-if="activeTab === 'trend'">
          <div ref="chartEl" style="position:relative" class="mb-3"></div>
          <div v-if="(data as any).direction" class="text-sm text-gray-600">
            Direction: <strong>{{ (data as any).direction }}</strong>
            <span v-if="(data as any).change != null"> (change: {{ (data as any).change }})</span>
          </div>
        </template>

        <!-- Sticky tab: card list -->
        <template v-else-if="activeTab === 'sticky'">
          <div v-if="Array.isArray(data)" class="space-y-2">
            <div v-for="(item, i) in (data as any[])" :key="i" class="border rounded p-2">
              <div class="font-medium text-sm">{{ item.type || item.id || 'Issue ' + (i+1) }}</div>
              <div class="text-xs text-gray-500">{{ item.description || item.message || JSON.stringify(item) }}</div>
            </div>
          </div>
          <pre v-else class="text-xs whitespace-pre-wrap bg-gray-50 rounded p-2">{{ JSON.stringify(data, null, 2) }}</pre>
        </template>
      </template>
      <pre v-else class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
    </div>
  `,
});

createVueApp(RoadyDebt).mount("#app");
