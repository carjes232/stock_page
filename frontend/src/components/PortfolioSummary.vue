<template>
  <div v-if="portfolioItems.length > 0" class="card rounded-xl p-6 mb-6">
    <h2 class="text-xl font-bold mb-4">{{ title }} Summary</h2>
    <div class="grid grid-cols-1 md:grid-cols-4 gap-6">
      <div class="text-center">
        <div class="text-sm text-slate-600 mb-1">Total Investment</div>
        <div class="text-2xl font-bold text-slate-900">
          {{ money(totalInvestment) }}
        </div>
      </div>
      <div class="text-center">
        <div class="text-sm text-slate-600 mb-1">Total P&L</div>
        <div 
          class="text-2xl font-bold"
          :class="totalPnL >= 0 ? 'text-success' : 'text-danger'"
        >
          {{ money(totalPnL) }}
        </div>
      </div>
      <div class="text-center" v-if="biggestWinner">
        <div class="text-sm text-slate-600 mb-1">Biggest Winner</div>
        <div class="text-lg font-medium text-success">{{ biggestWinner.ticker }}</div>
        <div class="text-sm text-success">{{ money(biggestWinner.pnl) }}</div>
      </div>
      <div class="text-center" v-if="biggestLoser">
        <div class="text-sm text-slate-600 mb-1">Biggest Loser</div>
        <div class="text-lg font-medium text-danger">{{ biggestLoser.ticker }}</div>
        <div class="text-sm text-danger">{{ money(biggestLoser.pnl) }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PortfolioItem } from '../stores/portfolio';

defineProps<{
  title: string;
  portfolioItems: PortfolioItem[];
  totalInvestment: number;
  totalPnL: number;
  biggestWinner: PortfolioItem | null;
  biggestLoser: PortfolioItem | null;
}>();

function money(v?: number | null): string {
  if (v == null) return '-';
  return `${v.toFixed(2)}`;
}
</script>