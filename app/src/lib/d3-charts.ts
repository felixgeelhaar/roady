import * as d3 from "d3";

// ─── Shared tooltip helper ──────────────────────────────────────────
function createTooltip(container: HTMLElement): d3.Selection<HTMLDivElement, unknown, null, undefined> {
  return d3.select(container)
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
}

// ─── Color palette ──────────────────────────────────────────────────
const STATUS_COLORS: Record<string, string> = {
  done: "#22c55e",
  verified: "#16a34a",
  in_progress: "#3b82f6",
  blocked: "#eab308",
  pending: "#9ca3af",
  critical: "#ef4444",
  high: "#f97316",
  medium: "#eab308",
  low: "#6b7280",
};

function colorFor(key: string): string {
  return STATUS_COLORS[key] ?? d3.schemeTableau10[Math.abs(hashStr(key)) % 10];
}

function hashStr(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = ((h << 5) - h + s.charCodeAt(i)) | 0;
  return h;
}

// ─── 1. Donut Chart ─────────────────────────────────────────────────
export interface DonutDatum {
  label: string;
  value: number;
}

export interface DonutOptions {
  width?: number;
  height?: number;
  innerRadiusRatio?: number;
  onClick?: (label: string) => void;
}

export function renderDonut(
  container: HTMLElement,
  data: DonutDatum[],
  options: DonutOptions = {},
) {
  d3.select(container).selectAll("*").remove();
  if (!data.length || data.every((d) => d.value === 0)) return;

  const width = options.width ?? 220;
  const height = options.height ?? 220;
  const radius = Math.min(width, height) / 2;
  const innerRadius = radius * (options.innerRadiusRatio ?? 0.55);
  const total = d3.sum(data, (d) => d.value);

  const svg = d3
    .select(container)
    .append("svg")
    .attr("width", width)
    .attr("height", height)
    .append("g")
    .attr("transform", `translate(${width / 2},${height / 2})`);

  const pie = d3
    .pie<DonutDatum>()
    .value((d) => d.value)
    .sort(null);

  const arc = d3.arc<d3.PieArcDatum<DonutDatum>>().innerRadius(innerRadius).outerRadius(radius);
  const arcHover = d3.arc<d3.PieArcDatum<DonutDatum>>().innerRadius(innerRadius).outerRadius(radius + 4);

  const tooltip = createTooltip(container);

  svg
    .selectAll("path")
    .data(pie(data))
    .enter()
    .append("path")
    .attr("d", arc)
    .attr("fill", (d) => colorFor(d.data.label))
    .attr("stroke", "white")
    .attr("stroke-width", 2)
    .style("cursor", options.onClick ? "pointer" : "default")
    .on("mouseover", function (event, d) {
      d3.select(this).transition().duration(150).attr("d", arcHover as any);
      const pct = ((d.data.value / total) * 100).toFixed(0);
      tooltip.html(`<strong>${d.data.label}</strong>: ${d.data.value} (${pct}%)`).style("opacity", "1");
    })
    .on("mousemove", function (event) {
      const [x, y] = d3.pointer(event, container);
      tooltip.style("left", `${x + 12}px`).style("top", `${y - 10}px`);
    })
    .on("mouseout", function () {
      d3.select(this).transition().duration(150).attr("d", arc as any);
      tooltip.style("opacity", "0");
    })
    .on("click", (_event, d) => options.onClick?.(d.data.label));

  // Center total
  svg
    .append("text")
    .attr("text-anchor", "middle")
    .attr("dy", "-0.1em")
    .attr("fill", "#374151")
    .attr("font-size", "20px")
    .attr("font-weight", "700")
    .text(total.toString());

  svg
    .append("text")
    .attr("text-anchor", "middle")
    .attr("dy", "1.2em")
    .attr("fill", "#9ca3af")
    .attr("font-size", "10px")
    .text("total");

  // Legend
  const legend = d3
    .select(container)
    .append("div")
    .style("display", "flex")
    .style("flex-wrap", "wrap")
    .style("gap", "8px")
    .style("margin-top", "8px")
    .style("font-size", "11px");

  data.forEach((d) => {
    legend
      .append("span")
      .style("display", "flex")
      .style("align-items", "center")
      .style("gap", "4px")
      .html(
        `<span style="width:8px;height:8px;border-radius:50%;background:${colorFor(d.label)};display:inline-block"></span>${d.label}: ${d.value}`,
      );
  });
}

// ─── 2. Force-Directed Graph ────────────────────────────────────────
export interface GraphNode {
  id: string;
  label: string;
  status?: string;
  group?: string;
  size?: number;
}

export interface GraphLink {
  source: string;
  target: string;
  color?: string;
  label?: string;
}

export interface ForceGraphOptions {
  width?: number;
  height?: number;
  nodeRadius?: number;
  onNodeClick?: (id: string) => void;
}

export function renderForceGraph(
  container: HTMLElement,
  nodes: GraphNode[],
  links: GraphLink[],
  options: ForceGraphOptions = {},
) {
  d3.select(container).selectAll("*").remove();
  if (!nodes.length) return;

  const width = options.width ?? 500;
  const height = options.height ?? 350;
  const baseRadius = options.nodeRadius ?? 8;

  const svg = d3
    .select(container)
    .append("svg")
    .attr("width", width)
    .attr("height", height)
    .attr("viewBox", [0, 0, width, height].join(" "));

  const g = svg.append("g");

  // Zoom
  svg.call(
    d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.3, 4])
      .on("zoom", (event) => g.attr("transform", event.transform)) as any,
  );

  const tooltip = createTooltip(container);

  const simulation = d3
    .forceSimulation(nodes as d3.SimulationNodeDatum[])
    .force("link", d3.forceLink(links).id((d: any) => d.id).distance(70))
    .force("charge", d3.forceManyBody().strength(-200))
    .force("center", d3.forceCenter(width / 2, height / 2))
    .force("collision", d3.forceCollide(baseRadius + 4));

  // Arrows
  svg
    .append("defs")
    .selectAll("marker")
    .data(["arrow"])
    .enter()
    .append("marker")
    .attr("id", "arrow")
    .attr("viewBox", "0 -5 10 10")
    .attr("refX", 20)
    .attr("refY", 0)
    .attr("markerWidth", 6)
    .attr("markerHeight", 6)
    .attr("orient", "auto")
    .append("path")
    .attr("d", "M0,-5L10,0L0,5")
    .attr("fill", "#9ca3af");

  const link = g
    .append("g")
    .selectAll("line")
    .data(links)
    .enter()
    .append("line")
    .attr("stroke", (d) => d.color ?? "#d1d5db")
    .attr("stroke-width", 1.5)
    .attr("marker-end", "url(#arrow)");

  const node = g
    .append("g")
    .selectAll("circle")
    .data(nodes)
    .enter()
    .append("circle")
    .attr("r", (d) => (d.size ?? 1) * baseRadius)
    .attr("fill", (d) => colorFor(d.status ?? d.group ?? "pending"))
    .attr("stroke", "white")
    .attr("stroke-width", 1.5)
    .style("cursor", "grab")
    .on("mouseover", (_event, d) => {
      const lines = [`<strong>${d.label}</strong>`];
      if (d.status) lines.push(`Status: ${d.status}`);
      if (d.group) lines.push(`Group: ${d.group}`);
      tooltip.html(lines.join("<br/>")).style("opacity", "1");
    })
    .on("mousemove", (event) => {
      const [x, y] = d3.pointer(event, container);
      tooltip.style("left", `${x + 12}px`).style("top", `${y - 10}px`);
    })
    .on("mouseout", () => tooltip.style("opacity", "0"))
    .on("click", (_event, d) => options.onNodeClick?.(d.id))
    .call(
      d3
        .drag<SVGCircleElement, GraphNode>()
        .on("start", (event, d: any) => {
          if (!event.active) simulation.alphaTarget(0.3).restart();
          d.fx = d.x;
          d.fy = d.y;
        })
        .on("drag", (event, d: any) => {
          d.fx = event.x;
          d.fy = event.y;
        })
        .on("end", (event, d: any) => {
          if (!event.active) simulation.alphaTarget(0);
          d.fx = null;
          d.fy = null;
        }),
    );

  // Labels
  const labels = g
    .append("g")
    .selectAll("text")
    .data(nodes)
    .enter()
    .append("text")
    .text((d) => d.label.length > 20 ? d.label.slice(0, 18) + "…" : d.label)
    .attr("font-size", "9px")
    .attr("fill", "#374151")
    .attr("dx", baseRadius + 4)
    .attr("dy", 3);

  simulation.on("tick", () => {
    link
      .attr("x1", (d: any) => d.source.x)
      .attr("y1", (d: any) => d.source.y)
      .attr("x2", (d: any) => d.target.x)
      .attr("y2", (d: any) => d.target.y);
    node.attr("cx", (d: any) => d.x).attr("cy", (d: any) => d.y);
    labels.attr("x", (d: any) => d.x).attr("y", (d: any) => d.y);
  });
}

// ─── 3. Gauge Chart ─────────────────────────────────────────────────
export interface GaugeOptions {
  width?: number;
  height?: number;
  min?: number;
  max?: number;
  thresholds?: { value: number; color: string }[];
  label?: string;
  suffix?: string;
}

export function renderGauge(
  container: HTMLElement,
  value: number,
  options: GaugeOptions = {},
) {
  d3.select(container).selectAll("*").remove();

  const width = options.width ?? 200;
  const height = options.height ?? 130;
  const min = options.min ?? 0;
  const max = options.max ?? 100;
  const clamped = Math.max(min, Math.min(max, value));
  const pct = (clamped - min) / (max - min);

  const thresholds = options.thresholds ?? [
    { value: 0.5, color: "#22c55e" },
    { value: 0.75, color: "#eab308" },
    { value: 0.9, color: "#f97316" },
    { value: 1.0, color: "#ef4444" },
  ];

  function gaugeColor(p: number): string {
    for (const t of thresholds) {
      if (p <= t.value) return t.color;
    }
    return thresholds[thresholds.length - 1]?.color ?? "#ef4444";
  }

  const svg = d3
    .select(container)
    .append("svg")
    .attr("width", width)
    .attr("height", height);

  const cx = width / 2;
  const cy = height - 10;
  const r = Math.min(cx - 10, cy - 10);
  const startAngle = -Math.PI / 2;
  const endAngle = Math.PI / 2;

  // Background arc
  const bgArc = d3.arc<any>()
    .innerRadius(r * 0.7)
    .outerRadius(r)
    .startAngle(startAngle)
    .endAngle(endAngle);

  svg
    .append("path")
    .attr("d", bgArc({}) as string)
    .attr("transform", `translate(${cx},${cy})`)
    .attr("fill", "#f3f4f6");

  // Value arc
  const valueAngle = startAngle + pct * (endAngle - startAngle);
  const valueArc = d3.arc<any>()
    .innerRadius(r * 0.7)
    .outerRadius(r)
    .startAngle(startAngle)
    .endAngle(valueAngle);

  svg
    .append("path")
    .attr("d", valueArc({}) as string)
    .attr("transform", `translate(${cx},${cy})`)
    .attr("fill", gaugeColor(pct));

  // Value text
  svg
    .append("text")
    .attr("x", cx)
    .attr("y", cy - r * 0.25)
    .attr("text-anchor", "middle")
    .attr("font-size", "18px")
    .attr("font-weight", "700")
    .attr("fill", "#374151")
    .text(`${Math.round(value)}${options.suffix ?? ""}`);

  // Label
  if (options.label) {
    svg
      .append("text")
      .attr("x", cx)
      .attr("y", cy - r * 0.05)
      .attr("text-anchor", "middle")
      .attr("font-size", "10px")
      .attr("fill", "#9ca3af")
      .text(options.label);
  }

  // Min/max labels
  svg.append("text").attr("x", cx - r).attr("y", cy + 12).attr("text-anchor", "middle").attr("font-size", "9px").attr("fill", "#9ca3af").text(min.toString());
  svg.append("text").attr("x", cx + r).attr("y", cy + 12).attr("text-anchor", "middle").attr("font-size", "9px").attr("fill", "#9ca3af").text(max.toString());

  // Tooltip
  const tooltip = createTooltip(container);
  svg
    .on("mouseover", () => {
      const pctStr = (pct * 100).toFixed(0);
      tooltip.html(`${value} / ${max} (${pctStr}%)`).style("opacity", "1");
    })
    .on("mousemove", (event) => {
      const [x, y] = d3.pointer(event, container);
      tooltip.style("left", `${x + 12}px`).style("top", `${y - 10}px`);
    })
    .on("mouseout", () => tooltip.style("opacity", "0"));
}

// ─── 4. Horizontal Bar Chart ────────────────────────────────────────
export interface BarDatum {
  label: string;
  value: number;
  color?: string;
  tooltip?: string;
}

export interface HorizontalBarOptions {
  width?: number;
  height?: number;
  barHeight?: number;
  onClick?: (label: string) => void;
}

export function renderHorizontalBars(
  container: HTMLElement,
  data: BarDatum[],
  options: HorizontalBarOptions = {},
) {
  d3.select(container).selectAll("*").remove();
  if (!data.length) return;

  const barHeight = options.barHeight ?? 24;
  const margin = { top: 5, right: 40, bottom: 5, left: 90 };
  const width = (options.width ?? 400) - margin.left - margin.right;
  const height = data.length * (barHeight + 6);

  const svg = d3
    .select(container)
    .append("svg")
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom)
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  const maxVal = d3.max(data, (d) => d.value) ?? 1;

  const x = d3.scaleLinear().domain([0, maxVal]).range([0, width]);
  const y = d3
    .scaleBand()
    .domain(data.map((d) => d.label))
    .range([0, height])
    .padding(0.2);

  const tooltip = createTooltip(container);

  // Labels
  svg
    .selectAll(".bar-label")
    .data(data)
    .enter()
    .append("text")
    .attr("x", -4)
    .attr("y", (d) => (y(d.label) ?? 0) + y.bandwidth() / 2)
    .attr("text-anchor", "end")
    .attr("dy", "0.35em")
    .attr("font-size", "11px")
    .attr("fill", "#374151")
    .text((d) => d.label.length > 12 ? d.label.slice(0, 11) + "…" : d.label);

  // Bars
  svg
    .selectAll("rect")
    .data(data)
    .enter()
    .append("rect")
    .attr("x", 0)
    .attr("y", (d) => y(d.label) ?? 0)
    .attr("width", (d) => x(d.value))
    .attr("height", y.bandwidth())
    .attr("fill", (d) => d.color ?? colorFor(d.label))
    .attr("rx", 3)
    .style("cursor", options.onClick ? "pointer" : "default")
    .on("mouseover", (_event, d) => {
      tooltip.html(d.tooltip ?? `<strong>${d.label}</strong>: ${d.value}`).style("opacity", "1");
    })
    .on("mousemove", (event) => {
      const [px, py] = d3.pointer(event, container);
      tooltip.style("left", `${px + 12}px`).style("top", `${py - 10}px`);
    })
    .on("mouseout", () => tooltip.style("opacity", "0"))
    .on("click", (_event, d) => options.onClick?.(d.label));

  // Value labels
  svg
    .selectAll(".val-label")
    .data(data)
    .enter()
    .append("text")
    .attr("x", (d) => x(d.value) + 4)
    .attr("y", (d) => (y(d.label) ?? 0) + y.bandwidth() / 2)
    .attr("dy", "0.35em")
    .attr("font-size", "10px")
    .attr("fill", "#6b7280")
    .text((d) => d.value.toString());
}

// ─── 5. Line Chart ──────────────────────────────────────────────────
export interface LinePoint {
  x: number | string;
  y: number;
}

export interface LineChartOptions {
  width?: number;
  height?: number;
  xLabel?: string;
  yLabel?: string;
  color?: string;
}

export function renderLineChart(
  container: HTMLElement,
  data: LinePoint[],
  options: LineChartOptions = {},
) {
  d3.select(container).selectAll("*").remove();
  if (data.length < 2) return;

  const margin = { top: 15, right: 20, bottom: 35, left: 45 };
  const width = (options.width ?? 420) - margin.left - margin.right;
  const height = (options.height ?? 180) - margin.top - margin.bottom;
  const color = options.color ?? "#3b82f6";

  const svg = d3
    .select(container)
    .append("svg")
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom)
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  const xVals = data.map((d, i) => (typeof d.x === "number" ? d.x : i));
  const x = d3.scaleLinear().domain(d3.extent(xVals) as [number, number]).range([0, width]);
  const y = d3
    .scaleLinear()
    .domain([0, d3.max(data, (d) => d.y) ?? 1])
    .nice()
    .range([height, 0]);

  // Grid
  svg
    .append("g")
    .call(d3.axisLeft(y).tickSize(-width).tickFormat(() => ""))
    .call((g) => g.selectAll(".tick line").attr("stroke", "#e5e7eb").attr("stroke-dasharray", "3,3"))
    .call((g) => g.select(".domain").remove());

  // Axes
  svg
    .append("g")
    .attr("transform", `translate(0,${height})`)
    .call(d3.axisBottom(x).ticks(6))
    .call((g) => g.select(".domain").attr("stroke", "#d1d5db"))
    .call((g) => g.selectAll(".tick text").attr("fill", "#6b7280").attr("font-size", "10px"));

  svg
    .append("g")
    .call(d3.axisLeft(y).ticks(5))
    .call((g) => g.select(".domain").attr("stroke", "#d1d5db"))
    .call((g) => g.selectAll(".tick text").attr("fill", "#6b7280").attr("font-size", "10px"));

  // Line
  const line = d3
    .line<LinePoint>()
    .x((d, i) => x(typeof d.x === "number" ? d.x : i))
    .y((d) => y(d.y))
    .curve(d3.curveMonotoneX);

  svg.append("path").datum(data).attr("fill", "none").attr("stroke", color).attr("stroke-width", 2).attr("d", line);

  // Dots
  svg
    .selectAll("circle")
    .data(data)
    .enter()
    .append("circle")
    .attr("cx", (d, i) => x(typeof d.x === "number" ? d.x : i))
    .attr("cy", (d) => y(d.y))
    .attr("r", 3)
    .attr("fill", color);

  // Tooltip crosshair
  const tooltip = createTooltip(container);
  const crosshair = svg.append("line").attr("stroke", "#d1d5db").attr("stroke-dasharray", "3,3").attr("y1", 0).attr("y2", height).style("opacity", "0");

  svg
    .append("rect")
    .attr("width", width)
    .attr("height", height)
    .attr("fill", "transparent")
    .on("mousemove", (event) => {
      const [mx] = d3.pointer(event);
      const xVal = x.invert(mx);
      const idx = d3.bisector<LinePoint, number>((d, v) => (typeof d.x === "number" ? d.x : 0) - v).left(data, xVal);
      const d = data[Math.min(idx, data.length - 1)];
      if (!d) return;
      crosshair.attr("x1", mx).attr("x2", mx).style("opacity", "1");
      tooltip.html(`<strong>${d.x}</strong>: ${d.y.toFixed(2)}`).style("opacity", "1");
      const [px, py] = d3.pointer(event, container);
      tooltip.style("left", `${px + 12}px`).style("top", `${py - 10}px`);
    })
    .on("mouseout", () => {
      crosshair.style("opacity", "0");
      tooltip.style("opacity", "0");
    });

  // Axis labels
  if (options.xLabel) {
    svg.append("text").attr("x", width / 2).attr("y", height + 30).attr("text-anchor", "middle").attr("font-size", "10px").attr("fill", "#9ca3af").text(options.xLabel);
  }
  if (options.yLabel) {
    svg.append("text").attr("transform", "rotate(-90)").attr("y", -margin.left + 12).attr("x", -height / 2).attr("text-anchor", "middle").attr("font-size", "10px").attr("fill", "#9ca3af").text(options.yLabel);
  }
}

// ─── 6. Collapsible Tree ────────────────────────────────────────────
export interface TreeNode {
  name: string;
  tooltip?: string;
  children?: TreeNode[];
}

export interface TreeOptions {
  width?: number;
  height?: number;
  nodeRadius?: number;
}

export function renderTree(
  container: HTMLElement,
  root: TreeNode,
  options: TreeOptions = {},
) {
  d3.select(container).selectAll("*").remove();

  const nodeR = options.nodeRadius ?? 5;
  const margin = { top: 20, right: 120, bottom: 20, left: 80 };
  const baseWidth = (options.width ?? 500) - margin.left - margin.right;

  // Count leaves to auto-size height
  function countLeaves(n: TreeNode): number {
    if (!n.children?.length) return 1;
    return n.children.reduce((s, c) => s + countLeaves(c), 0);
  }
  const leaves = countLeaves(root);
  const computedHeight = Math.max(leaves * 28, 120);
  const totalHeight = computedHeight + margin.top + margin.bottom;

  const svg = d3
    .select(container)
    .append("svg")
    .attr("width", baseWidth + margin.left + margin.right)
    .attr("height", totalHeight)
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  const tooltip = createTooltip(container);

  const hierarchy = d3.hierarchy(root);
  const tree = d3.tree<TreeNode>().size([computedHeight, baseWidth]);
  const treeData = tree(hierarchy);

  // Links
  svg
    .selectAll(".link")
    .data(treeData.links())
    .enter()
    .append("path")
    .attr("fill", "none")
    .attr("stroke", "#d1d5db")
    .attr("stroke-width", 1.5)
    .attr("d", (d) => {
      return `M${d.source.y},${d.source.x}C${(d.source.y + d.target.y) / 2},${d.source.x} ${(d.source.y + d.target.y) / 2},${d.target.x} ${d.target.y},${d.target.x}`;
    });

  // Nodes
  const nodeGroup = svg
    .selectAll(".node")
    .data(treeData.descendants())
    .enter()
    .append("g")
    .attr("transform", (d) => `translate(${d.y},${d.x})`);

  nodeGroup
    .append("circle")
    .attr("r", nodeR)
    .attr("fill", (d) => (d.children ? "#3b82f6" : "#22c55e"))
    .attr("stroke", "white")
    .attr("stroke-width", 1.5)
    .on("mouseover", (_event, d) => {
      const lines = [`<strong>${d.data.name}</strong>`];
      if (d.data.tooltip) lines.push(d.data.tooltip);
      if (d.children) lines.push(`${d.children.length} children`);
      tooltip.html(lines.join("<br/>")).style("opacity", "1");
    })
    .on("mousemove", (event) => {
      const [x, y] = d3.pointer(event, container);
      tooltip.style("left", `${x + 12}px`).style("top", `${y - 10}px`);
    })
    .on("mouseout", () => tooltip.style("opacity", "0"));

  nodeGroup
    .append("text")
    .attr("dx", (d) => (d.children ? -nodeR - 4 : nodeR + 4))
    .attr("dy", 3)
    .attr("text-anchor", (d) => (d.children ? "end" : "start"))
    .attr("font-size", "10px")
    .attr("fill", "#374151")
    .text((d) => {
      const name = d.data.name;
      return name.length > 30 ? name.slice(0, 28) + "…" : name;
    });
}
