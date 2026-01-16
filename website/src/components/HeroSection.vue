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
</script>

<template>
  <section class="relative pt-32 pb-20 overflow-hidden">
    <!-- Background glow -->
    <div class="absolute top-0 left-1/2 -translate-x-1/2 w-[1000px] h-[600px] bg-violet-600/10 blur-[120px] rounded-full -z-10"></div>

    <div class="max-w-7xl mx-auto px-6 text-center">
      <h1 class="text-5xl md:text-7xl font-extrabold text-white mb-6 leading-tight">
        Your Project's <br>
        <span class="gradient-text">Memory Layer.</span>
      </h1>

      <p class="text-xl md:text-2xl text-gray-400 max-w-2xl mx-auto mb-10">
        A planning-first system of record that turns intent into specs, plans, and drift-aware roadmaps for humans and AI agents.
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

      <!-- Quick Links -->
      <div class="flex flex-col sm:flex-row items-center justify-center space-y-4 sm:space-y-0 sm:space-x-6">
        <a
          href="#commands"
          class="text-gray-300 hover:text-white flex items-center space-x-2 transition-colors"
        >
          <span>Explore the CLI</span>
          <i data-lucide="arrow-right" class="w-4 h-4"></i>
        </a>
        <a
          href="#mcp"
          class="text-gray-300 hover:text-white flex items-center space-x-2 transition-colors"
        >
          <span>MCP Integration</span>
          <i data-lucide="arrow-right" class="w-4 h-4"></i>
        </a>
      </div>
    </div>
  </section>
</template>
