import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref } from "vue";

const app = createApp("Roady Init");

const RoadyInit = defineComponent({
  components: { StatusBadge },
  setup() {
    const projectName = ref("");
    const message = ref("");
    const success = ref(false);
    const loading = ref(false);

    app.ontoolresult = (result) => {
      const text = result.content?.find((c) => c.type === "text")?.text ?? "";
      message.value = text;
      success.value = text.includes("successfully");
    };

    async function init() {
      if (!projectName.value) return;
      loading.value = true;
      const r = await callTool(app, "roady_init", { name: projectName.value });
      message.value = extractText(r);
      success.value = message.value.includes("successfully");
      loading.value = false;
    }

    app.connect();

    return { projectName, message, success, loading, init };
  },
  template: `
    <div>
      <h1>Initialize Project</h1>
      <div v-if="success" class="border border-green-200 rounded p-4 bg-green-50">
        <StatusBadge status="done" />
        <span class="ml-2 text-sm">{{ message }}</span>
      </div>
      <template v-else>
        <div class="mb-3">
          <label class="block text-xs text-gray-500 mb-1">Project Name</label>
          <input v-model="projectName" placeholder="my-project" class="border rounded px-2 py-1.5 text-sm w-full" @keyup.enter="init" />
        </div>
        <button :disabled="loading || !projectName" class="px-4 py-1.5 text-sm bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50" @click="init">
          {{ loading ? 'Initializing...' : 'Initialize' }}
        </button>
        <div v-if="message" class="mt-2 text-sm text-red-500">{{ message }}</div>
      </template>
    </div>
  `,
});

createVueApp(RoadyInit).mount("#app");
