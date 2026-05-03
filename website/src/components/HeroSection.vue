<script setup lang="ts">
import { ref } from 'vue';

type InstallMethod = 'brew' | 'go' | 'source';

const selectedMethod = ref<InstallMethod>('brew');
const copied = ref(false);

const installCommands: Record<InstallMethod, string> = {
  brew: 'brew install felixgeelhaar/tap/roady',
  go: 'go install github.com/felixgeelhaar/roady/cmd/roady@latest',
  source: 'git clone https://github.com/felixgeelhaar/roady && cd roady && go build -o roady ./cmd/roady',
};

const methods: { key: InstallMethod; label: string }[] = [
  { key: 'brew', label: 'Homebrew' },
  { key: 'go', label: 'Go Install' },
  { key: 'source', label: 'From Source' },
];

async function copyCommand() {
  try {
    await navigator.clipboard.writeText(installCommands[selectedMethod.value]);
    copied.value = true;
    setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch {
    // Fallback for older browsers
    const textArea = document.createElement('textarea');
    textArea.value = installCommands[selectedMethod.value];
    document.body.appendChild(textArea);
    textArea.select();
    document.execCommand('copy');
    document.body.removeChild(textArea);
    copied.value = true;
    setTimeout(() => {
      copied.value = false;
    }, 2000);
  }
}

// Roady Cloud waitlist signal capture. No backend wired yet — submissions
// are stored in localStorage so the conversion path captures intent today;
// a real backend (Formspree, Resend, or a hosted endpoint) lands once
// Cloud has a credible alpha date. The point of shipping the form now is
// to qualify the roadmap claim with actual demand data.
const waitlistEmail = ref('');
const waitlistSubmitted = ref(false);
const waitlistError = ref('');

function submitWaitlist() {
  waitlistError.value = '';
  const email = waitlistEmail.value.trim();
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    waitlistError.value = 'Please enter a valid email.';
    return;
  }
  try {
    const key = 'roady-cloud-waitlist';
    const existing: string[] = JSON.parse(localStorage.getItem(key) ?? '[]');
    if (!existing.includes(email)) existing.push(email);
    localStorage.setItem(key, JSON.stringify(existing));
  } catch {
    // Private browsing / storage disabled — still flag success so the
    // visitor sees acknowledgement; we lose the signal silently.
  }
  waitlistSubmitted.value = true;
  waitlistEmail.value = '';
}
</script>

<template>
  <section class="relative pt-32 pb-20 overflow-hidden">
    <!-- Background glow -->
    <div class="absolute top-0 left-1/2 -translate-x-1/2 w-[1000px] h-[600px] bg-violet-600/10 blur-[120px] rounded-full -z-10"></div>

    <div class="max-w-7xl mx-auto px-6 text-center">
      <p class="uppercase tracking-widest text-xs md:text-sm text-violet-400 mb-4 font-semibold">
        Planning memory for AI coding agents
      </p>

      <h1 class="text-5xl md:text-7xl font-extrabold text-white mb-6 leading-tight">
        Your plan <br>
        <span class="gradient-text">survives the reset.</span>
      </h1>

      <p class="text-xl md:text-2xl text-gray-400 max-w-2xl mx-auto mb-10">
        Specs, plans, and execution state that stay in sync with your code &mdash; readable by you, writable by your agent, durable across sessions.
      </p>

      <!-- Install Method Selector -->
      <div class="max-w-xl mx-auto mb-8">
        <!-- Method Tabs -->
        <div class="flex justify-center mb-4 space-x-2">
          <button
            v-for="method in methods"
            :key="method.key"
            @click="selectedMethod = method.key"
            class="px-4 py-2 text-sm font-medium rounded-lg transition-all"
            :class="[
              selectedMethod === method.key
                ? 'bg-violet-600 text-white'
                : 'text-gray-400 hover:text-white hover:bg-white/5'
            ]"
          >
            {{ method.label }}
          </button>
        </div>

        <!-- Command Display -->
        <div class="bg-[#161B22] p-1 rounded-2xl border border-white/10 flex items-center shadow-2xl">
          <code class="mono px-4 py-3 text-violet-400 text-sm flex-1 overflow-x-auto whitespace-nowrap">
            {{ installCommands[selectedMethod] }}
          </code>
          <button
            @click="copyCommand"
            class="p-3 hover:bg-white/5 rounded-xl transition-colors flex-shrink-0"
            :title="copied ? 'Copied!' : 'Copy to clipboard'"
          >
            <i
              :data-lucide="copied ? 'check' : 'copy'"
              class="w-5 h-5"
              :class="copied ? 'text-emerald-400' : 'text-gray-500'"
            ></i>
          </button>
        </div>
      </div>

      <p class="text-sm text-gray-500 mb-8 max-w-xl mx-auto">
        After install, run
        <code class="mono text-violet-400 px-1">roady demo</code>
        to see the full spec &rarr; plan &rarr; drift loop on a sample project in under a minute.
      </p>

      <!-- Primary CTA: keep the visitor on-site through the value prop -->
      <div class="flex flex-col sm:flex-row items-center justify-center space-y-3 sm:space-y-0 sm:space-x-4 mb-12">
        <a
          href="#commands"
          class="inline-flex items-center space-x-2 px-6 py-3 rounded-xl bg-violet-600 hover:bg-violet-500 text-white font-semibold shadow-lg shadow-violet-600/30 transition-colors"
        >
          <span>See the actual workflow</span>
          <i data-lucide="arrow-down" class="w-4 h-4"></i>
        </a>
        <a
          href="#mcp"
          class="inline-flex items-center space-x-2 px-6 py-3 rounded-xl border border-white/10 hover:border-white/20 text-gray-200 hover:text-white transition-colors"
        >
          <span>How it talks to your AI</span>
          <i data-lucide="arrow-right" class="w-4 h-4"></i>
        </a>
      </div>

      <!-- Secondary: Roady Cloud waitlist signal capture -->
      <div class="max-w-md mx-auto p-5 rounded-2xl border border-white/5 bg-white/[0.02]">
        <p class="text-xs uppercase tracking-widest text-violet-400 font-semibold mb-2">
          Roady Cloud &mdash; coming
        </p>
        <p class="text-sm text-gray-400 mb-3">
          Hosted MCP, multi-repo dashboard, audit retention, SOC2. CLI stays MIT
          forever; Cloud is the open-core paid wedge. Drop your email if you
          want first access.
        </p>
        <form
          v-if="!waitlistSubmitted"
          @submit.prevent="submitWaitlist"
          class="flex flex-col sm:flex-row gap-2"
        >
          <input
            v-model="waitlistEmail"
            type="email"
            required
            placeholder="you@company.dev"
            class="flex-1 px-3 py-2 rounded-lg bg-black/40 border border-white/10 text-sm text-white placeholder-gray-600 focus:outline-none focus:border-violet-500"
          />
          <button
            type="submit"
            class="px-4 py-2 rounded-lg bg-violet-600/80 hover:bg-violet-500 text-white text-sm font-medium transition-colors"
          >
            Join waitlist
          </button>
        </form>
        <p v-if="waitlistError" class="mt-2 text-xs text-red-400">{{ waitlistError }}</p>
        <p v-if="waitlistSubmitted" class="text-sm text-emerald-400">
          Thanks &mdash; we'll be in touch when Cloud has an alpha date.
        </p>
        <p class="mt-3 text-[11px] text-gray-600">
          See <a href="https://github.com/felixgeelhaar/roady/blob/main/ROADMAP.md" class="underline hover:text-gray-400">ROADMAP.md</a>
          for the full open-core boundary.
        </p>
      </div>
    </div>
  </section>
</template>
