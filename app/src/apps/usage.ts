import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderGauge } from "@/lib/d3-charts";

const app = createApp("Roady Usage");

interface Usage {
  total_tokens: number;
  token_limit: number;
  calls: number;
  provider: string;
  model: string;
}

const RoadyUsage = defineComponent({
  setup() {
    const usage = ref<Usage | null>(null);
    const loading = ref(true);
    const raw = ref("");
    const gaugeEl = ref<HTMLElement | null>(null);

    app.ontoolresult = (result) => {
      loading.value = false;
      const text = result.content?.find((c) => c.type === "text")?.text ?? "";
      raw.value = text;
      try { usage.value = JSON.parse(text); } catch { /* raw display */ }
    };

    async function refresh() {
      loading.value = true;
      const r = await callTool(app, "roady_get_usage");
      const text = extractText(r);
      raw.value = text;
      try { usage.value = JSON.parse(text); } catch { /* raw */ }
      loading.value = false;
    }

    watch(usage, async () => {
      await nextTick();
      if (!usage.value || !gaugeEl.value) return;
      const u = usage.value;
      const limit = u.token_limit || 100000;
      renderGauge(gaugeEl.value, u.total_tokens, {
        max: limit,
        label: "tokens used",
        suffix: "",
      });
    });

    app.connect();

    return { usage, loading, raw, refresh, gaugeEl };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>AI Usage</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="refresh">Refresh</button>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="usage">
        <div class="flex gap-6 items-start mb-3">
          <div ref="gaugeEl" style="position:relative"></div>
          <div class="grid grid-cols-1 gap-3 flex-1">
            <div class="border rounded p-2">
              <div class="text-lg font-bold">{{ usage.total_tokens.toLocaleString() }} <span class="text-xs font-normal text-gray-400">/ {{ (usage.token_limit || 100000).toLocaleString() }}</span></div>
              <div class="text-xs text-gray-500">Tokens Used</div>
            </div>
            <div class="border rounded p-2">
              <div class="text-lg font-bold">{{ usage.calls }}</div>
              <div class="text-xs text-gray-500">API Calls</div>
            </div>
            <div class="border rounded p-2">
              <div class="text-sm font-medium">{{ usage.provider }}</div>
              <div class="text-xs text-gray-500">{{ usage.model }}</div>
            </div>
          </div>
        </div>
      </template>
      <pre v-else class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
    </div>
  `,
});

createVueApp(RoadyUsage).mount("#app");
