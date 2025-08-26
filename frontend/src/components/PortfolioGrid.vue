<template>
  <div v-if="portfolioItems.length > 0" class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
    <div
      v-for="(item, index) in portfolioItems"
      :key="index"
      class="card rounded-xl transition-all hover:shadow-lg hover:-translate-y-1 flex flex-col cursor-pointer"
      @click="$emit('goDetail', item.ticker)"
    >
      <div class="p-5 flex-grow">
        <div class="flex items-center justify-between mb-3">
          <div class="text-xl font-bold text-brand-600 hover:text-brand-700">{{ item.ticker }}</div>
          <div class="flex gap-2">
            <button @click.stop="$emit('addMoreShares', index)" class="text-slate-400 hover:text-success" title="Buy / Add Shares">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd" />
              </svg>
            </button>
            <button @click.stop="$emit('sellShares', index)" class="text-slate-400 hover:text-danger" title="Sell Shares">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M4 10a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1z" clip-rule="evenodd" />
              </svg>
            </button>
            <button @click.stop="$emit('editItem', index)" class="text-slate-400 hover:text-brand-600" title="Edit">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
              </svg>
            </button>
            <button @click.stop="$emit('removeItem', index)" class="text-slate-400 hover:text-danger" title="Delete">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
              </svg>
            </button>
          </div>
        </div>
        <div class="text-sm text-slate-600 mb-2">
          {{ item.shares }} shares @ ${{ item.avgPrice.toFixed(2) }}
        </div>
        <div class="text-sm text-slate-600">
          Investment: ${{ (item.shares * item.avgPrice).toFixed(2) }}
        </div>
      </div>
      <div class="px-5 py-4 bg-slate-50 border-t border-slate-100 text-xs">
        <div class="flex justify-between items-center mb-1">
          <span class="text-slate-600">Current Price</span>
          <div class="font-medium">{{ item.currentPrice ? `${item.currentPrice.toFixed(2)}` : 'Loading...' }}</div>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-slate-600">P&L</span>
          <div 
            class="font-medium" 
            :class="item.pnl && item.pnl >= 0 ? 'text-success' : 'text-danger'"
          >
            {{ item.pnl !== undefined ? `${item.pnl.toFixed(2)}` : 'Calculating...' }}
          </div>
        </div>
      </div>
    </div>
  </div>
  <div v-else class="card rounded-xl text-slate-600 text-center py-12">
    {{ emptyMessage }}
  </div>
</template>

<script setup lang="ts">
import type { PortfolioItem } from '../stores/portfolio';

defineProps<{
  portfolioItems: PortfolioItem[];
  emptyMessage: string;
}>();

defineEmits<{
  goDetail: [ticker: string];
  addMoreShares: [index: number];
  sellShares: [index: number];
  editItem: [index: number];
  removeItem: [index: number];
}>();
</script>
