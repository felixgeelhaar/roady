import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool, type ToolResult } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderForceGraph, type GraphNode, type GraphLink } from "@/lib/d3-charts";

const app = createApp("Roady Plan");

interface Task {
  id: string;
  title: string;
  priority: string;
  depends_on?: string[];
}

interface Plan {
  id: string;
  approved: boolean;
  tasks: Task[];
}

const RoadyPlan = defineComponent({
  components: { StatusBadge },
  setup() {
    const plan = ref<Plan | null>(null);
    const taskStates = ref<Record<string, string>>({});
    const loading = ref(true);
    const message = ref("");
    const graphEl = ref<HTMLElement | null>(null);
    const selectedNode = ref<string | null>(null);

    app.ontoolresult = (result) => {
      loading.value = false;
      try {
        const raw = result.content?.find((c) => c.type === "text")?.text ?? "";
        const parsed = JSON.parse(raw);
        if (parsed.tasks) plan.value = parsed;
        else message.value = raw;
      } catch {
        message.value = result.content?.find((c) => c.type === "text")?.text ?? "";
      }
    };

    async function refresh() {
      loading.value = true;
      const [planR, stateR] = await Promise.all([
        callTool(app, "roady_get_plan"),
        callTool(app, "roady_get_state"),
      ]);
      try {
        plan.value = JSON.parse(extractText(planR));
      } catch {
        message.value = extractText(planR);
      }
      try {
        const state = JSON.parse(extractText(stateR));
        const ts: Record<string, string> = {};
        for (const [id, s] of Object.entries(state.task_states ?? {})) {
          ts[id] = (s as any).status ?? "pending";
        }
        taskStates.value = ts;
      } catch { /* no state */ }
      loading.value = false;
    }

    async function generate() {
      loading.value = true;
      const r = await callTool(app, "roady_generate_plan");
      message.value = extractText(r);
      await refresh();
    }

    async function approve() {
      const r = await callTool(app, "roady_approve_plan");
      message.value = extractText(r);
      await refresh();
    }

    function onNodeClick(id: string) {
      selectedNode.value = selectedNode.value === id ? null : id;
    }

    watch([plan, taskStates], async () => {
      await nextTick();
      if (!plan.value?.tasks?.length || !graphEl.value) return;
      const tasks = plan.value.tasks;
      const states = taskStates.value;

      const nodes: GraphNode[] = tasks.map((t) => ({
        id: t.id,
        label: t.title,
        status: states[t.id] ?? "pending",
        group: t.priority,
      }));

      const links: GraphLink[] = [];
      for (const t of tasks) {
        for (const dep of t.depends_on ?? []) {
          links.push({ source: dep, target: t.id });
        }
      }

      renderForceGraph(graphEl.value, nodes, links, {
        width: 520,
        height: 350,
        onNodeClick,
      });
    });

    app.connect();

    return { plan, loading, message, graphEl, selectedNode, refresh, generate, approve };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Execution Plan</h1>
        <div class="flex gap-2">
          <button class="px-3 py-1 text-xs bg-gray-200 rounded hover:bg-gray-300" @click="generate">Generate</button>
          <button class="px-3 py-1 text-xs bg-green-500 text-white rounded hover:bg-green-600" @click="approve">Approve</button>
          <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="refresh">Refresh</button>
        </div>
      </div>
      <div v-if="message" class="text-sm text-green-600 mb-2">{{ message }}</div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="plan">
        <div class="text-sm text-gray-600 mb-2">
          Plan {{ plan.id }} — {{ plan.tasks.length }} tasks
          <StatusBadge :status="plan.approved ? 'pass' : 'pending'" />
        </div>
        <div ref="graphEl" style="position:relative;border:1px solid #e5e7eb;border-radius:6px;overflow:hidden"></div>
        <div v-if="selectedNode" class="mt-2 text-xs text-blue-600">Selected: {{ selectedNode }}</div>
      </template>
    </div>
  `,
});

createVueApp(RoadyPlan).mount("#app");
