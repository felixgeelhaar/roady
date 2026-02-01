import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, callTool } from "@/lib/mcp";
import ProgressBar from "@/components/ProgressBar.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import { defineComponent, ref, computed, onMounted, watch, nextTick } from "vue";
import * as d3 from "d3";
import { renderGauge } from "@/lib/d3-charts";

const app = createApp("Roady Forecast");

interface BurndownPoint {
  date: string;
  actual: number;
  projected: number;
}

interface VelocityWindow {
  days: number;
  velocity: number;
  count: number;
}

interface Forecast {
  remaining: number;
  completed: number;
  total: number;
  velocity: number;
  estimated_days: number;
  completion_rate: number;
  trend: string;
  trend_slope: number;
  confidence: number;
  ci_low: number;
  ci_expected: number;
  ci_high: number;
  burndown: BurndownPoint[];
  windows: VelocityWindow[];
  data_points: number;
}

function renderBurndownChart(container: HTMLElement, data: BurndownPoint[], total: number) {
  d3.select(container).selectAll("*").remove();

  const margin = { top: 20, right: 20, bottom: 40, left: 45 };
  const width = 520 - margin.left - margin.right;
  const height = 220 - margin.top - margin.bottom;

  const svg = d3.select(container)
    .append("svg")
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom)
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  const parseDate = d3.timeParse("%Y-%m-%d");
  const points = data.map(d => ({
    date: parseDate(d.date)!,
    actual: d.actual,
    projected: d.projected,
  }));

  const x = d3.scaleTime()
    .domain(d3.extent(points, d => d.date) as [Date, Date])
    .range([0, width]);

  const y = d3.scaleLinear()
    .domain([0, total || d3.max(points, d => Math.max(d.actual, d.projected)) || 1])
    .nice()
    .range([height, 0]);

  // Grid lines
  svg.append("g")
    .attr("class", "grid")
    .call(d3.axisLeft(y).tickSize(-width).tickFormat(() => ""))
    .call(g => g.selectAll(".tick line").attr("stroke", "#e5e7eb").attr("stroke-dasharray", "3,3"))
    .call(g => g.select(".domain").remove());

  // X axis
  svg.append("g")
    .attr("transform", `translate(0,${height})`)
    .call(d3.axisBottom(x).ticks(6).tickFormat(d3.timeFormat("%b %d") as any))
    .call(g => g.select(".domain").attr("stroke", "#d1d5db"))
    .call(g => g.selectAll(".tick text").attr("fill", "#6b7280").attr("font-size", "10px"));

  // Y axis
  svg.append("g")
    .call(d3.axisLeft(y).ticks(5))
    .call(g => g.select(".domain").attr("stroke", "#d1d5db"))
    .call(g => g.selectAll(".tick text").attr("fill", "#6b7280").attr("font-size", "10px"));

  // Y axis label
  svg.append("text")
    .attr("transform", "rotate(-90)")
    .attr("y", -margin.left + 12)
    .attr("x", -height / 2)
    .attr("text-anchor", "middle")
    .attr("fill", "#9ca3af")
    .attr("font-size", "10px")
    .text("Tasks remaining");

  // Actual burndown line — a point is "actual" unless it's projected-only (actual=0 AND projected>0)
  const actualPoints = points.filter(d => !(d.actual === 0 && d.projected > 0));
  if (actualPoints.length > 0) {
    const line = d3.line<typeof actualPoints[0]>()
      .x(d => x(d.date))
      .y(d => y(d.actual))
      .curve(d3.curveLinear);

    svg.append("path")
      .datum(actualPoints)
      .attr("fill", "none")
      .attr("stroke", "#3b82f6")
      .attr("stroke-width", 2.5)
      .attr("d", line);

    // Dots on actual data
    svg.selectAll(".dot-actual")
      .data(actualPoints)
      .enter()
      .append("circle")
      .attr("cx", d => x(d.date))
      .attr("cy", d => y(d.actual))
      .attr("r", 3)
      .attr("fill", "#3b82f6");
  }

  // Projected burndown line (dashed)
  const projectedPoints = points.filter(d => d.projected > 0);
  if (projectedPoints.length > 0) {
    const projLine = d3.line<typeof projectedPoints[0]>()
      .x(d => x(d.date))
      .y(d => y(d.projected))
      .curve(d3.curveLinear);

    svg.append("path")
      .datum(projectedPoints)
      .attr("fill", "none")
      .attr("stroke", "#f59e0b")
      .attr("stroke-width", 2)
      .attr("stroke-dasharray", "6,4")
      .attr("d", projLine);

    // Dots on projected
    svg.selectAll(".dot-proj")
      .data(projectedPoints)
      .enter()
      .append("circle")
      .attr("cx", d => x(d.date))
      .attr("cy", d => y(d.projected))
      .attr("r", 3)
      .attr("fill", "#f59e0b")
      .attr("opacity", 0.7);
  }

  // Ideal burndown line (straight from total to 0)
  if (points.length >= 2) {
    svg.append("line")
      .attr("x1", x(points[0].date))
      .attr("y1", y(total))
      .attr("x2", x(points[points.length - 1].date))
      .attr("y2", y(0))
      .attr("stroke", "#d1d5db")
      .attr("stroke-width", 1)
      .attr("stroke-dasharray", "4,4");
  }

  // Tooltip
  const tooltip = d3.select(container)
    .append("div")
    .style("position", "absolute")
    .style("background", "rgba(0,0,0,0.8)")
    .style("color", "white")
    .style("padding", "6px 10px")
    .style("border-radius", "4px")
    .style("font-size", "11px")
    .style("pointer-events", "none")
    .style("opacity", 0);

  const bisect = d3.bisector<typeof points[0], Date>(d => d.date).left;

  svg.append("rect")
    .attr("width", width)
    .attr("height", height)
    .attr("fill", "transparent")
    .on("mousemove", function(event) {
      const [mx] = d3.pointer(event);
      const date = x.invert(mx);
      const idx = bisect(points, date, 1);
      const d0 = points[idx - 1];
      const d1 = points[idx];
      if (!d0) return;
      const d = d1 && (date.getTime() - d0.date.getTime()) > (d1.date.getTime() - date.getTime()) ? d1 : d0;
      const val = d.actual > 0 ? d.actual : d.projected;
      const type = d.actual > 0 ? "Actual" : "Projected";

      tooltip
        .html(`<strong>${d3.timeFormat("%b %d")(d.date)}</strong><br/>${type}: ${val} tasks`)
        .style("left", `${x(d.date) + margin.left + 10}px`)
        .style("top", `${y(val) + margin.top - 10}px`)
        .style("opacity", 1);
    })
    .on("mouseleave", () => tooltip.style("opacity", 0));

  // Legend
  const legend = d3.select(container)
    .append("div")
    .style("display", "flex")
    .style("gap", "16px")
    .style("margin-top", "8px")
    .style("font-size", "11px")
    .style("color", "#6b7280");

  legend.append("span").html('<svg width="16" height="2" style="vertical-align:middle;margin-right:4px"><line x1="0" y1="1" x2="16" y2="1" stroke="#3b82f6" stroke-width="2"/></svg>Actual');
  legend.append("span").html('<svg width="16" height="2" style="vertical-align:middle;margin-right:4px"><line x1="0" y1="1" x2="16" y2="1" stroke="#f59e0b" stroke-width="2" stroke-dasharray="4,3"/></svg>Projected');
  legend.append("span").html('<svg width="16" height="2" style="vertical-align:middle;margin-right:4px"><line x1="0" y1="1" x2="16" y2="1" stroke="#d1d5db" stroke-width="1" stroke-dasharray="4,4"/></svg>Ideal');
}

function renderVelocityChart(container: HTMLElement, windows: VelocityWindow[]) {
  d3.select(container).selectAll("*").remove();

  const margin = { top: 10, right: 20, bottom: 30, left: 45 };
  const width = 300 - margin.left - margin.right;
  const height = 140 - margin.top - margin.bottom;

  const svg = d3.select(container)
    .append("svg")
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom)
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  const x = d3.scaleBand()
    .domain(windows.map(w => `${w.days}d`))
    .range([0, width])
    .padding(0.4);

  const y = d3.scaleLinear()
    .domain([0, d3.max(windows, w => w.velocity) || 1])
    .nice()
    .range([height, 0]);

  svg.append("g")
    .attr("transform", `translate(0,${height})`)
    .call(d3.axisBottom(x))
    .call(g => g.select(".domain").attr("stroke", "#d1d5db"))
    .call(g => g.selectAll(".tick text").attr("fill", "#6b7280").attr("font-size", "10px"));

  svg.append("g")
    .call(d3.axisLeft(y).ticks(4))
    .call(g => g.select(".domain").attr("stroke", "#d1d5db"))
    .call(g => g.selectAll(".tick text").attr("fill", "#6b7280").attr("font-size", "10px"));

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
    .style("opacity", 0)
    .style("z-index", "10")
    .style("white-space", "nowrap");

  svg.selectAll(".bar")
    .data(windows)
    .enter()
    .append("rect")
    .attr("x", d => x(`${d.days}d`)!)
    .attr("y", d => y(d.velocity))
    .attr("width", x.bandwidth())
    .attr("height", d => height - y(d.velocity))
    .attr("fill", "#6366f1")
    .attr("rx", 3)
    .on("mouseover", (_event, d) => {
      tooltip.html(`<strong>${d.days}-day window</strong><br/>Velocity: ${d.velocity.toFixed(2)} tasks/day<br/>Completions: ${d.count}`)
        .style("opacity", 1);
    })
    .on("mousemove", (event) => {
      const [px, py] = d3.pointer(event, container);
      tooltip.style("left", `${px + 12}px`).style("top", `${py - 10}px`);
    })
    .on("mouseout", () => tooltip.style("opacity", 0));

  // Value labels on bars
  svg.selectAll(".bar-label")
    .data(windows)
    .enter()
    .append("text")
    .attr("x", d => x(`${d.days}d`)! + x.bandwidth() / 2)
    .attr("y", d => y(d.velocity) - 4)
    .attr("text-anchor", "middle")
    .attr("fill", "#4b5563")
    .attr("font-size", "10px")
    .attr("font-weight", "600")
    .text(d => d.velocity.toFixed(2));
}

const RoadyForecast = defineComponent({
  components: { ProgressBar, StatusBadge },
  setup() {
    const forecast = ref<Forecast | null>(null);
    const loading = ref(true);
    const message = ref("");
    const burndownEl = ref<HTMLElement | null>(null);
    const velocityEl = ref<HTMLElement | null>(null);
    const confidenceEl = ref<HTMLElement | null>(null);

    function parse(raw: string) {
      try {
        const data = JSON.parse(raw);
        if (data.total !== undefined) {
          forecast.value = data;
        } else {
          message.value = raw;
        }
      } catch {
        message.value = raw;
      }
    }

    app.ontoolresult = (result) => {
      loading.value = false;
      parse((result.content?.find((c: any) => c.type === "text") as any)?.text ?? "");
    };

    async function refresh() {
      loading.value = true;
      const r = await callTool(app, "roady_forecast");
      const text = r?.content?.find((c: any) => c.type === "text")?.text ?? "";
      parse(text);
      loading.value = false;
    }

    const trendLabel = computed(() => {
      if (!forecast.value) return "";
      switch (forecast.value.trend) {
        case "accelerating": return "Accelerating";
        case "decelerating": return "Decelerating";
        default: return "Stable";
      }
    });

    const trendStatus = computed(() => {
      if (!forecast.value) return "pending";
      switch (forecast.value.trend) {
        case "accelerating": return "pass";
        case "decelerating": return "fail";
        default: return "pending";
      }
    });

    const hasBurndown = computed(() =>
      forecast.value?.burndown && forecast.value.burndown.length > 0
    );

    const hasWindows = computed(() =>
      forecast.value?.windows && forecast.value.windows.length > 0
    );

    watch(forecast, async () => {
      await nextTick();
      if (hasBurndown.value && burndownEl.value) {
        renderBurndownChart(burndownEl.value, forecast.value!.burndown, forecast.value!.total);
      }
      if (hasWindows.value && velocityEl.value) {
        renderVelocityChart(velocityEl.value, forecast.value!.windows);
      }
      if (confidenceEl.value && forecast.value!.confidence > 0) {
        renderGauge(confidenceEl.value, forecast.value!.confidence * 100, {
          width: 160,
          height: 100,
          label: "Confidence",
          suffix: "%",
          thresholds: [
            { value: 0.4, color: "#ef4444" },
            { value: 0.6, color: "#f97316" },
            { value: 0.8, color: "#eab308" },
            { value: 1.0, color: "#22c55e" },
          ],
        });
      }
    });

    app.connect();

    return {
      forecast, loading, message, refresh,
      trendLabel, trendStatus,
      hasBurndown, hasWindows,
      burndownEl, velocityEl, confidenceEl,
    };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Project Forecast</h1>
        <button class="px-3 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600" @click="refresh">Refresh</button>
      </div>
      <div v-if="message" class="text-sm text-green-600 mb-2">{{ message }}</div>
      <div v-if="loading" class="text-gray-400">Loading...</div>
      <template v-else-if="forecast">
        <!-- Metrics -->
        <div class="grid grid-cols-4 gap-3 mb-4">
          <div class="border rounded p-3 text-center">
            <div class="text-2xl font-bold">{{ forecast.velocity.toFixed(2) }}</div>
            <div class="text-xs text-gray-500">tasks/day</div>
          </div>
          <div class="border rounded p-3 text-center">
            <div class="text-2xl font-bold">{{ forecast.remaining }}</div>
            <div class="text-xs text-gray-500">remaining</div>
          </div>
          <div class="border rounded p-3 text-center">
            <div class="text-2xl font-bold">{{ forecast.estimated_days.toFixed(1) }}</div>
            <div class="text-xs text-gray-500">est. days</div>
          </div>
          <div class="border rounded p-3 text-center">
            <div class="text-lg font-bold"><StatusBadge :status="trendStatus" /></div>
            <div class="text-xs text-gray-500">{{ trendLabel }}</div>
          </div>
        </div>

        <!-- Progress -->
        <div class="mb-4">
          <div class="text-sm text-gray-600 mb-1">Completion: {{ forecast.completed }}/{{ forecast.total }} ({{ forecast.completion_rate.toFixed(0) }}%)</div>
          <ProgressBar :value="forecast.completion_rate" :max="100" />
        </div>

        <!-- Confidence interval with gauge -->
        <div v-if="forecast.ci_expected > 0" class="mb-4 flex items-center gap-4">
          <div ref="confidenceEl" style="position:relative;flex-shrink:0"></div>
          <div class="text-sm text-gray-600">
            <span class="font-medium">Confidence interval:</span><br/>
            {{ forecast.ci_low.toFixed(1) }} – {{ forecast.ci_expected.toFixed(1) }} – {{ forecast.ci_high.toFixed(1) }} days
          </div>
        </div>

        <!-- D3 Burndown chart -->
        <div v-if="hasBurndown" class="mb-4">
          <div class="text-sm font-medium text-gray-700 mb-2">Burndown Timeline</div>
          <div ref="burndownEl" style="position:relative"></div>
        </div>

        <!-- D3 Velocity chart -->
        <div v-if="hasWindows" class="mb-4">
          <div class="text-sm font-medium text-gray-700 mb-2">Velocity by Window</div>
          <div ref="velocityEl" style="position:relative"></div>
        </div>
      </template>
    </div>
  `,
});

createVueApp(RoadyForecast).mount("#app");
