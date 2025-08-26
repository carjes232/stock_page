<template>
  <div class="border-t border-slate-200 pt-6">
    <h3 class="text-lg font-semibold mb-4">{{ formTitle }}</h3>
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Ticker</label>
        <input 
          v-model="portfolioEntry.ticker" 
          type="text" 
          placeholder="e.g. NVDA" 
          class="input w-full"
          @keyup.enter="handleSubmit"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Shares</label>
        <input 
          v-model="portfolioEntry.shares" 
          type="number" 
          step="any" 
          placeholder="e.g. 0.0141" 
          class="input w-full"
          @keyup.enter="handleSubmit"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-1">Avg. Price</label>
        <input 
          v-model="portfolioEntry.avgPrice" 
          type="number" 
          step="0.01" 
          placeholder="e.g. 120.50" 
          class="input w-full"
          @keyup.enter="handleSubmit"
        />
      </div>
      <div class="flex items-end">
        <button 
          @click="handleSubmit"
          class="btn w-full"
        >
          {{ buttonText }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PortfolioEntry } from '../stores/portfolio';

defineProps<{
  formTitle: string;
  buttonText: string;
  portfolioEntry: PortfolioEntry;
}>();

const emit = defineEmits<{
  submit: [];
}>();

function handleSubmit() {
  emit('submit');
}
</script>