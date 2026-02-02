import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool, type ToolResult } from "@/lib/mcp";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderTree, type TreeNode } from "@/lib/d3-charts";

const app = createApp("Roady Spec");

interface Feature {
  id: string;
  title: string;
  description: string;
  requirements?: { id: string; text: string }[];
}

interface Spec {
  title: string;
  features: Feature[];
}

const RoadySpec = defineComponent({
  setup() {
    const spec = ref<Spec | null>(null);
    const loading = ref(true);
    const newTitle = ref("");
    const newDesc = ref("");
    const adding = ref(false);
    const treeEl = ref<HTMLElement | null>(null);

    function load(result: ToolResult) {
      loading.value = false;
      try {
        spec.value = JSON.parse(extractText(result));
      } catch {
        const text = extractText(result);
        if (text) {
          try { spec.value = JSON.parse(text); } catch { /* raw */ }
        }
      }
    }

    app.ontoolresult = (result) => {
      loading.value = false;
      try {
        const raw = result.content?.find((c) => c.type === "text")?.text ?? "";
        spec.value = JSON.parse(raw);
      } catch {
        // may be non-JSON initial result
      }
    };

    async function refresh() {
      loading.value = true;
      const r = await callTool(app, "roady_get_spec");
      load(r);
    }

    async function addFeature() {
      if (!newTitle.value) return;
      adding.value = true;
      await callTool(app, "roady_add_feature", {
        title: newTitle.value,
        description: newDesc.value,
      });
      newTitle.value = "";
      newDesc.value = "";
      adding.value = false;
      await refresh();
    }

    watch(spec, async () => {
      await nextTick();
      if (!spec.value?.features?.length || !treeEl.value) return;
      const root: TreeNode = {
        name: spec.value.title || "Project",
        children: spec.value.features.map((f) => ({
          name: f.title,
          tooltip: f.description,
          children: (f.requirements ?? []).map((r) => ({
            name: r.text,
          })),
        })),
      };
      renderTree(treeEl.value, root, { width: 520 });
    });

    app.connect();

    return { spec, loading, newTitle, newDesc, adding, refresh, addFeature, treeEl };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Product Specification</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="refresh">Refresh</button>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="spec">
        <p class="text-sm text-gray-600 mb-3">{{ spec.title }} — {{ spec.features?.length ?? 0 }} features</p>
        <div v-if="spec.features?.length" ref="treeEl" style="position:relative" class="mb-4 border rounded p-2 overflow-x-auto"></div>
        <div v-for="f in spec.features" :key="f.id" class="border rounded p-3 mb-2">
          <div class="font-medium">{{ f.title }}</div>
          <div class="text-xs text-gray-500 mt-1">{{ f.description }}</div>
          <ul v-if="f.requirements?.length" class="mt-1 text-xs text-gray-400 list-disc pl-4">
            <li v-for="r in f.requirements" :key="r.id">{{ r.text }}</li>
          </ul>
        </div>
        <div class="mt-4 border-t pt-3">
          <h2>Add Feature</h2>
          <input v-model="newTitle" placeholder="Title" class="border rounded px-2 py-1 text-sm w-full mb-1" />
          <textarea v-model="newDesc" placeholder="Description" class="border rounded px-2 py-1 text-sm w-full mb-2" rows="2" />
          <button :disabled="adding || !newTitle" class="px-3 py-1 text-xs bg-green-500 text-white rounded hover:bg-green-600 disabled:opacity-50" @click="addFeature">
            {{ adding ? 'Adding...' : 'Add Feature' }}
          </button>
        </div>
      </template>
    </div>
  `,
});

createVueApp(RoadySpec).mount("#app");
