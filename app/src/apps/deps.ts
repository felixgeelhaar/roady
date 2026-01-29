import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import DataTable from "@/components/DataTable.vue";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderForceGraph, type GraphNode, type GraphLink } from "@/lib/d3-charts";

const app = createApp("Roady Dependencies");

interface Dep {
  name: string;
  repo: string;
  source_repo?: string;
  target_repo?: string;
  type?: string;
  status?: string;
}

const TYPE_COLORS: Record<string, string> = {
  runtime: "#3b82f6",
  data: "#22c55e",
  build: "#f97316",
  intent: "#8b5cf6",
};

const RoadyDeps = defineComponent({
  components: { StatusBadge, DataTable },
  setup() {
    const deps = ref<Dep[]>([]);
    const graphData = ref<{ summary?: string; has_cycle?: boolean; nodes?: any[]; edges?: any[] } | null>(null);
    const loading = ref(true);
    const raw = ref("");
    const viewMode = ref<"list" | "graph">("list");
    const graphEl = ref<HTMLElement | null>(null);

    const columns = [
      { key: "name", label: "Name" },
      { key: "repo", label: "Repository" },
      { key: "type", label: "Type" },
      { key: "status", label: "Status" },
    ];

    app.ontoolresult = (result) => {
      loading.value = false;
      const text = result.content?.find((c) => c.type === "text")?.text ?? "";
      raw.value = text;
      try {
        const parsed = JSON.parse(text);
        if (Array.isArray(parsed)) deps.value = parsed;
        else graphData.value = parsed;
      } catch { /* raw */ }
    };

    async function list() {
      viewMode.value = "list";
      loading.value = true;
      const r = await callTool(app, "roady_deps_list");
      const text = extractText(r);
      raw.value = text;
      try { deps.value = JSON.parse(text); } catch { /* raw */ }
      loading.value = false;
    }

    async function scan() {
      loading.value = true;
      const r = await callTool(app, "roady_deps_scan");
      raw.value = extractText(r);
      loading.value = false;
    }

    async function graph() {
      viewMode.value = "graph";
      loading.value = true;
      const r = await callTool(app, "roady_deps_graph", { check_cycles: true });
      const text = extractText(r);
      raw.value = text;
      try { graphData.value = JSON.parse(text); } catch { /* raw */ }
      loading.value = false;
    }

    watch([graphData, viewMode], async () => {
      await nextTick();
      if (viewMode.value !== "graph" || !graphEl.value) return;

      // Build graph from deps list if we have it, or from graph data
      const allDeps = deps.value.length ? deps.value : [];
      const repoSet = new Set<string>();
      const links: GraphLink[] = [];

      for (const d of allDeps) {
        const src = d.source_repo ?? d.repo ?? d.name;
        const tgt = d.target_repo ?? d.name ?? d.repo;
        repoSet.add(src);
        repoSet.add(tgt);
        if (src !== tgt) {
          links.push({
            source: src,
            target: tgt,
            color: TYPE_COLORS[d.type ?? ""] ?? "#d1d5db",
            label: d.type,
          });
        }
      }

      // Count connections for node sizing
      const connCount: Record<string, number> = {};
      for (const l of links) {
        connCount[l.source] = (connCount[l.source] ?? 0) + 1;
        connCount[l.target] = (connCount[l.target] ?? 0) + 1;
      }

      const nodes: GraphNode[] = Array.from(repoSet).map((id) => ({
        id,
        label: id,
        group: "repo",
        size: Math.max(1, Math.min(2, (connCount[id] ?? 1) / 2)),
      }));

      if (nodes.length) {
        renderForceGraph(graphEl.value, nodes, links, { width: 500, height: 350 });
      }
    });

    app.connect();

    return { deps, graphData, loading, raw, columns, viewMode, graphEl, list, scan, graph };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Dependencies</h1>
        <div class="flex gap-2">
          <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="list">List</button>
          <button class="px-3 py-1 text-xs bg-gray-200 rounded hover:bg-gray-300" @click="scan">Scan</button>
          <button class="px-3 py-1 text-xs bg-gray-200 rounded hover:bg-gray-300" @click="graph">Graph</button>
        </div>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else>
        <div v-if="viewMode === 'graph'" ref="graphEl" style="position:relative;border:1px solid #e5e7eb;border-radius:6px;overflow:hidden"></div>
        <div v-if="viewMode === 'graph' && graphData?.has_cycle" class="mt-2 text-xs text-red-500 font-medium">Cycle detected in dependency graph</div>
        <div v-if="viewMode === 'graph' && graphData?.summary" class="mt-1 text-xs text-gray-500">{{ graphData.summary }}</div>
        <DataTable v-if="viewMode === 'list' && deps.length" :columns="columns" :rows="deps" />
        <pre v-if="viewMode === 'list' && !deps.length" class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
      </template>
    </div>
  `,
});

createVueApp(RoadyDeps).mount("#app");
