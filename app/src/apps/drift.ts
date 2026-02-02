import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import ActionButton from "@/components/ActionButton.vue";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderHorizontalBars, type BarDatum } from "@/lib/d3-charts";

const app = createApp("Roady Drift");

interface DriftIssue {
  id: string;
  type: string;
  description: string;
  severity: string;
}

interface DriftReport {
  id: string;
  issues: DriftIssue[];
  created_at: string;
}

const SEVERITY_COLORS: Record<string, string> = {
  critical: "#ef4444",
  high: "#f97316",
  medium: "#eab308",
  low: "#6b7280",
};

const RoadyDrift = defineComponent({
  components: { StatusBadge, ActionButton },
  setup() {
    const report = ref<DriftReport | null>(null);
    const loading = ref(true);
    const message = ref("");
    const chartEl = ref<HTMLElement | null>(null);
    const filterSev = ref<string | null>(null);

    app.ontoolresult = (result) => {
      loading.value = false;
      try {
        report.value = JSON.parse(result.content?.find((c) => c.type === "text")?.text ?? "");
      } catch {
        message.value = result.content?.find((c) => c.type === "text")?.text ?? "";
      }
    };

    async function detect() {
      loading.value = true;
      message.value = "";
      const r = await callTool(app, "roady_detect_drift");
      try { report.value = JSON.parse(extractText(r)); } catch { message.value = extractText(r); }
      loading.value = false;
    }

    async function accept() {
      const r = await callTool(app, "roady_accept_drift");
      message.value = extractText(r);
      await detect();
    }

    async function explain() {
      const r = await callTool(app, "roady_explain_drift");
      message.value = extractText(r);
    }

    function onBarClick(label: string) {
      filterSev.value = filterSev.value === label ? null : label;
    }

    watch(report, async () => {
      await nextTick();
      if (!report.value?.issues?.length || !chartEl.value) return;
      const counts: Record<string, number> = {};
      for (const issue of report.value.issues) {
        counts[issue.severity] = (counts[issue.severity] ?? 0) + 1;
      }
      const order = ["critical", "high", "medium", "low"];
      const bars: BarDatum[] = order
        .filter((s) => counts[s])
        .map((s) => ({ label: s, value: counts[s], color: SEVERITY_COLORS[s] }));
      renderHorizontalBars(chartEl.value, bars, { width: 350, onClick: onBarClick });
    });

    app.connect();

    return { report, loading, message, detect, accept, explain, chartEl, filterSev };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Drift Detection</h1>
        <div class="flex gap-2">
          <ActionButton label="Detect" @click="detect" />
          <ActionButton label="Accept" variant="secondary" @click="accept" />
          <ActionButton label="Explain" variant="secondary" @click="explain" />
        </div>
      </div>
      <div v-if="message" class="text-sm text-green-600 mb-2 whitespace-pre-wrap">{{ message }}</div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="report">
        <div class="mb-2">
          <StatusBadge :status="report.issues?.length ? 'warning' : 'pass'" />
          <span class="ml-2 text-sm">{{ report.issues?.length ? report.issues.length + ' issues found' : 'No drift detected' }}</span>
        </div>
        <div v-if="report.issues?.length" class="mb-3" ref="chartEl" style="position:relative"></div>
        <div v-if="filterSev" class="text-xs text-blue-500 mb-2 cursor-pointer" @click="filterSev = null">Filter: {{ filterSev }} (click to clear)</div>
        <div v-if="report.issues?.length" class="space-y-2">
          <div v-for="issue in report.issues.filter(i => !filterSev || i.severity === filterSev)" :key="issue.id" class="border rounded p-2">
            <div class="flex items-center gap-2">
              <StatusBadge :status="issue.severity === 'high' || issue.severity === 'critical' ? 'critical' : issue.severity === 'medium' ? 'warning' : 'pending'" />
              <span class="font-medium text-sm">{{ issue.type }}</span>
              <span class="text-xs text-gray-400">{{ issue.severity }}</span>
            </div>
            <div class="text-xs text-gray-500 mt-1">{{ issue.description }}</div>
          </div>
        </div>
      </template>
    </div>
  `,
});

createVueApp(RoadyDrift).mount("#app");
