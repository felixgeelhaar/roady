import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import { defineComponent, ref } from "vue";

const app = createApp("Roady Org");

const RoadyOrg = defineComponent({
  setup() {
    const text = ref("");
    const loading = ref(true);

    app.ontoolresult = (result) => {
      loading.value = false;
      text.value = result.content?.find((c) => c.type === "text")?.text ?? "";
    };

    async function refresh() {
      loading.value = true;
      text.value = extractText(await callTool(app, "roady_org_status"));
      loading.value = false;
    }

    app.connect();

    return { text, loading, refresh };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Organization Status</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="refresh">Refresh</button>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <div v-else class="text-sm whitespace-pre-wrap">{{ text }}</div>
    </div>
  `,
});

createVueApp(RoadyOrg).mount("#app");
