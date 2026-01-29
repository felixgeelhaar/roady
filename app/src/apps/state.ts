import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import TaskCard from "@/components/TaskCard.vue";
import { defineComponent, ref, computed, watch, nextTick } from "vue";
import { renderDonut, type DonutDatum } from "@/lib/d3-charts";

const app = createApp("Roady State");

interface TaskState {
  status: string;
  evidence?: string;
}

interface ExecutionState {
  task_states: Record<string, TaskState>;
}

interface Task {
  id: string;
  title: string;
  priority: string;
  depends_on?: string[];
}

interface TaskItem {
  id: string;
  title: string;
  status: string;
  priority: string;
  dependsOn?: string[];
}

const LANES = ["pending", "in_progress", "blocked", "done"] as const;

const RoadyState = defineComponent({
  components: { TaskCard },
  setup() {
    const tasks = ref<TaskItem[]>([]);
    const loading = ref(true);
    const message = ref("");
    const donutEl = ref<HTMLElement | null>(null);

    const laneData = computed(() => {
      const lanes: Record<string, TaskItem[]> = {};
      for (const l of LANES) lanes[l] = [];
      for (const t of tasks.value) {
        const lane = LANES.includes(t.status as any) ? t.status : "pending";
        (lanes[lane] ??= []).push(t);
      }
      return lanes;
    });

    async function loadState() {
      loading.value = true;
      const [planR, stateR] = await Promise.all([
        callTool(app, "roady_get_plan"),
        callTool(app, "roady_get_state"),
      ]);
      try {
        const plan = JSON.parse(extractText(planR)) as { tasks: Task[] };
        const state = JSON.parse(extractText(stateR)) as ExecutionState;
        tasks.value = plan.tasks.map((t) => ({
          id: t.id,
          title: t.title,
          status: state.task_states?.[t.id]?.status ?? "pending",
          priority: t.priority,
          dependsOn: t.depends_on,
        }));
      } catch {
        message.value = "Failed to load state data";
      }
      loading.value = false;
    }

    async function onAction(event: string, taskId: string) {
      const r = await callTool(app, "roady_transition_task", { task_id: taskId, event });
      message.value = extractText(r);
      await loadState();
    }

    watch(tasks, async () => {
      await nextTick();
      if (!donutEl.value || !tasks.value.length) return;
      const counts: Record<string, number> = {};
      for (const t of tasks.value) counts[t.status] = (counts[t.status] ?? 0) + 1;
      const donutData: DonutDatum[] = Object.entries(counts).map(([label, value]) => ({ label, value }));
      renderDonut(donutEl.value, donutData, { width: 140, height: 140 });
    });

    app.ontoolresult = () => { loadState(); };
    app.connect();

    return { tasks, loading, message, onAction, loadState, laneData, donutEl };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Task Board</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="loadState">Refresh</button>
      </div>
      <div v-if="message" class="text-sm text-green-600 mb-2">{{ message }}</div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else>
        <div v-if="tasks.length" class="flex gap-4 mb-4 items-start">
          <div ref="donutEl" style="position:relative;flex-shrink:0"></div>
          <div class="text-sm text-gray-500">{{ tasks.length }} tasks</div>
        </div>
        <div class="grid grid-cols-4 gap-3">
          <div v-for="lane in ['pending', 'in_progress', 'blocked', 'done']" :key="lane">
            <div class="text-xs font-semibold text-gray-500 uppercase mb-2 px-1">{{ lane.replace('_', ' ') }}</div>
            <div class="space-y-2 min-h-[60px]">
              <TaskCard
                v-for="t in laneData[lane]"
                :key="t.id"
                :id="t.id"
                :title="t.title"
                :status="t.status"
                :priority="t.priority"
                :depends-on="t.dependsOn"
                @action="onAction"
              />
            </div>
          </div>
        </div>
      </template>
    </div>
  `,
});

createVueApp(RoadyState).mount("#app");
