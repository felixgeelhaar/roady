import "@/style.css";
import { createApp as createVueApp } from "vue";
import { createApp, extractText, callTool } from "@/lib/mcp";
import { defineComponent, ref, watch, nextTick } from "vue";
import { renderDonut, renderHorizontalBars, type DonutDatum, type BarDatum } from "@/lib/d3-charts";

const app = createApp("Roady Billing");

interface Rate {
  id: string;
  name: string;
  hourly_rate: number;
  default: boolean;
}

interface RateConfig {
  currency: string;
  tax?: {
    name: string;
    percent: number;
    included: boolean;
  };
  rates: Rate[];
}

interface CostEntry {
  task_id: string;
  title: string;
  rate_name: string;
  hours: number;
  cost: number;
  tax?: number;
  total_with_tax?: number;
}

interface CostReport {
  currency: string;
  tax_name?: string;
  tax_percent?: number;
  entries: CostEntry[];
  total_hours: number;
  total_cost: number;
  total_tax?: number;
  total_with_tax?: number;
}

interface BudgetStatus {
  budget_hours: number;
  used_hours: number;
  remaining: number;
  percent_used: number;
  over_budget: boolean;
}

const RoadyBilling = defineComponent({
  setup() {
    const rates = ref<RateConfig | null>(null);
    const costReport = ref<CostReport | null>(null);
    const budget = ref<BudgetStatus | null>(null);
    const loading = ref(true);
    const raw = ref("");
    const activeTab = ref<"rates" | "add-rate" | "cost" | "budget">("rates");
    const chartEl = ref<HTMLElement | null>(null);
    const chart2El = ref<HTMLElement | null>(null);

    // Form state
    const rateId = ref("");
    const rateName = ref("");
    const rateAmount = ref(0);
    const rateDefault = ref(false);
    const formMessage = ref("");

    // Helper for currency formatting
    function fmtCurrency(value: number): string {
      return value.toFixed(2);
    }
    function fmtHours(value: number): string {
      return value.toFixed(1);
    }

    app.ontoolresult = (result) => {
      loading.value = false;
      const text = result.content?.find((c) => c.type === "text")?.text ?? "";
      raw.value = text;
      try { 
        const parsed = JSON.parse(text);
        // Check if it's a simple message (like "Rate added")
        if (typeof parsed === "string") {
          formMessage.value = parsed;
        }
      } catch { /* raw */ }
    };

    async function loadRates() {
      loading.value = true;
      const r = await callTool(app, "roady_rate_list", {});
      const text = extractText(r);
      raw.value = text;
      try { rates.value = JSON.parse(text); } catch { rates.value = null; }
      loading.value = false;
    }

    async function loadCostReport() {
      loading.value = true;
      const r = await callTool(app, "roady_cost_report", { format: "json" });
      const text = extractText(r);
      raw.value = text;
      try { costReport.value = JSON.parse(text); } catch { costReport.value = null; }
      loading.value = false;
    }

    async function loadBudget() {
      loading.value = true;
      const r = await callTool(app, "roady_cost_budget", {});
      const text = extractText(r);
      raw.value = text;
      try { budget.value = JSON.parse(text); } catch { budget.value = null; }
      loading.value = false;
    }

    async function loadTab(tab: "rates" | "add-rate" | "cost" | "budget") {
      activeTab.value = tab;
      loading.value = true;
      formMessage.value = "";
      
      if (tab === "rates" || tab === "add-rate") {
        await loadRates();
      } else if (tab === "cost") {
        await loadCostReport();
      } else if (tab === "budget") {
        await loadBudget();
      }
      loading.value = false;
    }

    async function addRate() {
      if (!rateId.value || !rateName.value || rateAmount.value <= 0) {
        formMessage.value = "Please fill in all required fields";
        return;
      }
      loading.value = true;
      formMessage.value = "";
      const r = await callTool(app, "roady_rate_add", {
        id: rateId.value,
        name: rateName.value,
        hourly_rate: rateAmount.value,
        is_default: rateDefault.value,
      });
      const text = extractText(r);
      formMessage.value = text;
      rateId.value = "";
      rateName.value = "";
      rateAmount.value = 0;
      rateDefault.value = false;
      await loadRates();
      loading.value = false;
    }

    watch([costReport], async () => {
      await nextTick();
      if (!costReport.value || !chartEl.value) return;

      // Donut chart: cost by rate
      const byRate: Record<string, number> = {};
      for (const e of costReport.value.entries) {
        byRate[e.rate_name] = (byRate[e.rate_name] || 0) + e.cost;
      }
      const donutData: DonutDatum[] = Object.entries(byRate).map(([label, value]) => ({
        label,
        value,
      }));
      if (donutData.length > 0) {
        renderDonut(chartEl.value, donutData, { width: 200, height: 200 });
      }
    });

    watch([costReport], async () => {
      await nextTick();
      if (!costReport.value || !chart2El.value) return;

      // Horizontal bars: cost by task
      const bars: BarDatum[] = costReport.value.entries
        .slice(0, 10)
        .map(e => ({ label: e.task_id, value: e.cost }));
      if (bars.length > 0) {
        renderHorizontalBars(chart2El.value, bars, { width: 400 });
      }
    });

    app.connect();

    return { 
      rates, costReport, budget, loading, raw, activeTab, loadTab, 
      chartEl, chart2El,
      rateId, rateName, rateAmount, rateDefault, formMessage, addRate,
      fmtCurrency, fmtHours
    };
  },
  template: `
    <div>
      <div class="flex items-center justify-between mb-3">
        <h1>Billing</h1>
      </div>
      <div class="flex gap-1 mb-3">
        <button v-for="tab in ['rates', 'add-rate', 'cost', 'budget']" :key="tab"
          :class="activeTab === tab ? 'bg-blue-500 text-white' : 'bg-gray-200'"
          class="px-3 py-1 text-xs rounded hover:opacity-80"
          @click="loadTab(tab)">
          {{ tab === 'add-rate' ? 'Add Rate' : tab }}
        </button>
      </div>
      
      <div v-if="formMessage" class="mb-3 p-2 bg-blue-50 text-blue-700 text-sm rounded">
        {{ formMessage }}
      </div>

      <div v-if="loading" class="text-gray-400">Loading...</div>
      
      <!-- Rates Tab -->
      <template v-else-if="activeTab === 'rates' && rates">
        <div class="mb-3">
          <div class="text-sm text-gray-500 mb-2">
            Currency: {{ rates.currency || 'USD' }}
            <span v-if="rates.tax"> | Tax: {{ rates.tax.name }} ({{ rates.tax.percent }}%)</span>
          </div>
          <div class="space-y-2">
            <div v-for="rate in rates.rates" :key="rate.id" class="border rounded p-2 flex justify-between items-center">
              <div>
                <span class="font-medium">{{ rate.name }}</span>
                <span class="text-xs text-gray-500 ml-2">({{ rate.id }})</span>
                <span v-if="rate.default" class="ml-2 px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded">default</span>
              </div>
              <div class="text-right">
                <div class="font-bold">{{ fmtCurrency(rate.hourly_rate) }}/hr</div>
              </div>
            </div>
            <div v-if="rates.rates.length === 0" class="text-gray-500 text-sm">
              No rates configured. Use "Add Rate" to create one.
            </div>
          </div>
        </div>
      </template>

      <!-- Add Rate Tab -->
      <template v-else-if="activeTab === 'add-rate'">
        <div class="border rounded p-3 max-w-md">
          <div class="space-y-3">
            <div>
              <label class="block text-sm font-medium mb-1">Rate ID *</label>
              <input v-model="rateId" type="text" placeholder="e.g., senior" class="w-full border rounded px-2 py-1 text-sm" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">Name *</label>
              <input v-model="rateName" type="text" placeholder="e.g., Senior Developer" class="w-full border rounded px-2 py-1 text-sm" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">Hourly Rate *</label>
              <input v-model.number="rateAmount" type="number" step="0.01" min="0" placeholder="0.00" class="w-full border rounded px-2 py-1 text-sm" />
            </div>
            <div>
              <label class="flex items-center gap-2">
                <input v-model="rateDefault" type="checkbox" class="rounded" />
                <span class="text-sm">Set as default rate</span>
              </label>
            </div>
            <button @click="addRate" :disabled="loading" class="w-full bg-blue-500 text-white py-1 rounded hover:bg-blue-600 disabled:opacity-50">
              Add Rate
            </button>
          </div>
        </div>
      </template>

      <!-- Cost Report Tab -->
      <template v-else-if="activeTab === 'cost' && costReport">
        <div class="mb-3">
          <div class="text-sm text-gray-500 mb-2">
            Total: {{ fmtCurrency(costReport.total_cost) }} | 
            Hours: {{ fmtHours(costReport.total_hours) }}
            <span v-if="costReport.total_tax"> | Tax: {{ fmtCurrency(costReport.total_tax) }}</span>
            <span v-if="costReport.total_with_tax"> | Total with Tax: {{ fmtCurrency(costReport.total_with_tax) }}</span>
          </div>
          
          <div class="flex gap-4 mb-3">
            <div ref="chartEl" style="position:relative"></div>
            <div ref="chart2El" style="position:relative;flex:1"></div>
          </div>

          <table class="w-full text-sm border-collapse">
            <thead>
              <tr class="border-b">
                <th class="text-left py-1">Task</th>
                <th class="text-left py-1">Rate</th>
                <th class="text-right py-1">Hours</th>
                <th class="text-right py-1">Cost</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="entry in costReport.entries" :key="entry.task_id" class="border-b">
                <td class="py-1">
                  <div class="font-medium">{{ entry.title || entry.task_id }}</div>
                </td>
                <td class="py-1 text-gray-500">{{ entry.rate_name }}</td>
                <td class="py-1 text-right">{{ fmtHours(entry.hours) }}</td>
                <td class="py-1 text-right">{{ fmtCurrency(entry.cost) }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="font-bold">
                <td class="py-1">Total</td>
                <td></td>
                <td class="text-right">{{ fmtHours(costReport.total_hours) }}</td>
                <td class="text-right">{{ fmtCurrency(costReport.total_cost) }}</td>
              </tr>
            </tfoot>
          </table>
          
          <div v-if="costReport.entries.length === 0" class="text-gray-500 text-sm">
            No time entries found. Log time to tasks to see cost reports.
          </div>
        </div>
      </template>

      <!-- Budget Tab -->
      <template v-else-if="activeTab === 'budget' && budget">
        <div class="mb-3">
          <div class="grid grid-cols-2 gap-3 mb-3">
            <div class="border rounded p-3 text-center">
              <div class="text-2xl font-bold">{{ budget.budget_hours }}</div>
              <div class="text-xs text-gray-500">Budget (hours)</div>
            </div>
            <div class="border rounded p-3 text-center">
              <div class="text-2xl font-bold" :class="budget.over_budget ? 'text-red-500' : ''">
                {{ budget.used_hours.toFixed(1) }}
              </div>
              <div class="text-xs text-gray-500">Used (hours)</div>
            </div>
            <div class="border rounded p-3 text-center">
              <div class="text-2xl font-bold" :class="budget.remaining < 0 ? 'text-red-500' : 'text-green-500'">
                {{ budget.remaining.toFixed(1) }}
              </div>
              <div class="text-xs text-gray-500">Remaining</div>
            </div>
            <div class="border rounded p-3 text-center">
              <div class="text-2xl font-bold" :class="budget.percent_used > 90 ? 'text-red-500' : budget.percent_used > 75 ? 'text-yellow-500' : ''">
                {{ budget.percent_used.toFixed(0) }}%
              </div>
              <div class="text-xs text-gray-500">Used</div>
            </div>
          </div>
          
          <div v-if="budget.over_budget" class="p-3 bg-red-100 text-red-700 rounded text-sm">
            Warning: You are over budget by {{ (-budget.remaining).toFixed(1) }} hours!
          </div>
        </div>
      </template>
      
      <pre v-else class="text-xs whitespace-pre-wrap">{{ raw }}</pre>
    </div>
  `,
});

createVueApp(RoadyBilling).mount("#app");
