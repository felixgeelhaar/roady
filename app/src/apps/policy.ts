import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderGauge } from "@/lib/d3-charts";

const app = createApp("Roady Policy");

const RoadyPolicy = defineComponent({
  components: { StatusBadge },
  setup() {
    const violations = ref<{ rule: string; message: string }[]>([]);
    const clean = ref(false);
    const loading = ref(true);
    const raw = ref("");
    const gaugeEl = ref<HTMLElement | null>(null);

    function parseResult(text: string) {
      raw.value = text;
      if (text.includes("No policy violations")) {
        clean.value = true;
        violations.value = [];
      } else {
        try { violations.value = JSON.parse(text); clean.value = false; } catch { /* keep raw */ }
      }
    }

    app.ontoolresult = (result) => {
      loading.value = false;
      parseResult(result.content?.find((c) => c.type === "text")?.text ?? "");
    };

    async function check() {
      loading.value = true;
      const r = await callTool(app, "roady_check_policy");
      parseResult(extractText(r));
      loading.value = false;
    }

    watch([clean, violations], async () => {
      await nextTick();
      if (!gaugeEl.value) return;
      if (clean.value) {
        renderGauge(gaugeEl.value, 100, {
          max: 100,
          label: "compliant",
          suffix: "%",
          thresholds: [
            { value: 0.5, color: "#ef4444" },
            { value: 0.8, color: "#eab308" },
            { value: 1.0, color: "#22c55e" },
          ],
        });
      } else if (violations.value.length) {
        // Show compliance score inversely proportional to violations
        const score = Math.max(0, 100 - violations.value.length * 20);
        renderGauge(gaugeEl.value, score, {
          max: 100,
          label: `${violations.value.length} violations`,
          suffix: "%",
          thresholds: [
            { value: 0.5, color: "#ef4444" },
            { value: 0.8, color: "#eab308" },
            { value: 1.0, color: "#22c55e" },
          ],
        });
      }
    });

    app.connect();

    return { violations, clean, loading, raw, check, gaugeEl };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Policy Compliance</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="check">Check</button>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else>
        <div ref="gaugeEl" style="position:relative" class="mb-3"></div>
        <div v-if="clean" class="flex items-center gap-2">
          <StatusBadge status="pass" />
          <span class="text-sm">All policy checks passed</span>
        </div>
        <div v-else-if="violations.length" class="space-y-2">
          <div v-for="(v, i) in violations" :key="i" class="border border-red-200 rounded p-2 bg-red-50">
            <div class="font-medium text-sm text-red-700">{{ v.rule }}</div>
            <div class="text-xs text-red-500">{{ v.message }}</div>
          </div>
        </div>
        <pre v-else class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
      </template>
    </div>
  `,
});

createVueApp(RoadyPolicy).mount("#app");
