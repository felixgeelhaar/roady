import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref } from "vue";

const app = createApp("Roady Sync");

const RoadySync = defineComponent({
  components: { StatusBadge },
  setup() {
    const data = ref<unknown>(null);
    const loading = ref(true);
    const raw = ref("");

    app.ontoolresult = (result) => {
      loading.value = false;
      const text = result.content?.find((c) => c.type === "text")?.text ?? "";
      raw.value = text;
      try { data.value = JSON.parse(text); } catch { /* raw */ }
    };

    async function refresh() {
      loading.value = true;
      const r = await callTool(app, "roady_sync", { plugin_path: "" });
      raw.value = extractText(r);
      try { data.value = JSON.parse(raw.value); } catch { /* raw */ }
      loading.value = false;
    }

    app.connect();

    return { data, loading, raw, refresh };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Plugin Sync</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="refresh">Sync</button>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <pre v-else class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
    </div>
  `,
});

createVueApp(RoadySync).mount("#app");
