import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref, watch, nextTick } from "vue";
import * as d3 from "d3";

const app = createApp("Roady Git Sync");

interface SyncResult {
  task_id: string;
  commit: string;
  transitioned: boolean;
}

function renderTimeline(container: HTMLElement, results: SyncResult[]) {
  d3.select(container).selectAll("*").remove();
  if (!results.length) return;

  const margin = { top: 10, right: 20, bottom: 10, left: 20 };
  const width = 400 - margin.left - margin.right;
  const itemH = 32;
  const height = results.length * itemH;

  const svg = d3
    .select(container)
    .append("svg")
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom)
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  // Tooltip (consistent with d3-charts.ts style)
  const tooltip = d3.select(container)
    .append("div")
    .style("position", "absolute")
    .style("background", "rgba(0,0,0,0.8)")
    .style("color", "white")
    .style("padding", "6px 10px")
    .style("border-radius", "4px")
    .style("font-size", "11px")
    .style("pointer-events", "none")
    .style("opacity", "0")
    .style("z-index", "10")
    .style("white-space", "nowrap");

  // Vertical line
  svg
    .append("line")
    .attr("x1", 8)
    .attr("y1", 0)
    .attr("x2", 8)
    .attr("y2", height)
    .attr("stroke", "#d1d5db")
    .attr("stroke-width", 2);

  // Dots + labels
  results.forEach((r, i) => {
    const y = i * itemH + itemH / 2;

    svg
      .append("circle")
      .attr("cx", 8)
      .attr("cy", y)
      .attr("r", 5)
      .attr("fill", r.transitioned ? "#22c55e" : "#9ca3af")
      .attr("stroke", "white")
      .attr("stroke-width", 1.5)
      .style("cursor", "default")
      .on("mouseover", () => {
        tooltip.html(`<strong>${r.task_id}</strong><br/>${r.transitioned ? "Transitioned" : "No transition"}<br/>${r.commit}`)
          .style("opacity", "1");
      })
      .on("mousemove", (event) => {
        const [px, py] = d3.pointer(event, container);
        tooltip.style("left", `${px + 12}px`).style("top", `${py - 10}px`);
      })
      .on("mouseout", () => tooltip.style("opacity", "0"));

    svg
      .append("text")
      .attr("x", 22)
      .attr("y", y - 4)
      .attr("font-size", "11px")
      .attr("font-weight", "600")
      .attr("fill", "#374151")
      .text(r.task_id);

    svg
      .append("text")
      .attr("x", 22)
      .attr("y", y + 10)
      .attr("font-size", "9px")
      .attr("fill", "#9ca3af")
      .text(r.commit.length > 40 ? r.commit.slice(0, 38) + "…" : r.commit);
  });
}

const RoadyGitSync = defineComponent({
  components: { StatusBadge },
  setup() {
    const results = ref<SyncResult[]>([]);
    const loading = ref(true);
    const raw = ref("");
    const timelineEl = ref<HTMLElement | null>(null);

    app.ontoolresult = (result) => {
      loading.value = false;
      const text = result.content?.find((c) => c.type === "text")?.text ?? "";
      raw.value = text;
      try { results.value = JSON.parse(text); } catch { /* raw */ }
    };

    async function sync() {
      loading.value = true;
      const r = await callTool(app, "roady_git_sync");
      const text = extractText(r);
      raw.value = text;
      try { results.value = JSON.parse(text); } catch { /* raw */ }
      loading.value = false;
    }

    watch(results, async () => {
      await nextTick();
      if (timelineEl.value && results.value.length) {
        renderTimeline(timelineEl.value, results.value);
      }
    });

    app.connect();

    return { results, loading, raw, sync, timelineEl };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Git Sync</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="sync">Sync</button>
      </div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="results.length">
        <div ref="timelineEl" style="position:relative" class="mb-3"></div>
      </template>
      <pre v-else class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
    </div>
  `,
});

createVueApp(RoadyGitSync).mount("#app");
