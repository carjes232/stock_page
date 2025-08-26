<template>
  <div v-if="loading" class="card rounded-xl flex items-center justify-center py-12">
    <div class="flex items-center text-slate-600">
      <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-brand-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
      <span>Loading…</span>
    </div>
  </div>
  <div v-else-if="error" class="card rounded-xl text-danger text-center py-12">
    {{ error }}
  </div>
  <div v-else-if="items.length === 0" class="card rounded-xl text-slate-600 text-center py-12">
    No results found
  </div>
  <div v-else class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
    <div
      v-for="s in items"
      :key="s.id"
      class="card rounded-xl transition-all hover:shadow-lg hover:-translate-y-1 flex flex-col cursor-pointer"
    >
      <div class="p-5 flex-grow" @click="$emit('goDetail', s.ticker)">
        <div class="flex items-center mb-3">
          <div class="text-xl font-bold text-brand-600 hover:text-brand-700 mr-3">{{ s.ticker }}</div>
          <span class="badge">{{ s.action }}</span>
        </div>
        <div class="text-sm text-slate-700 truncate mb-1" :title="s.company">{{ s.company }}</div>
      </div>
      <div class="px-5 py-4 bg-slate-50 border-t border-slate-100 text-xs">
        <div class="flex items-center justify-between mb-2">
          <div class="flex items-center">
            <button @click.stop="$emit('toggleWatchlist', s.ticker)" class="btn-icon mr-2">
              <svg v-if="isWatched(s.ticker)" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
              </svg>
              <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.783-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
              </svg>
            </button>
            <div class="text-xs text-slate-500 truncate" :title="s.brokerage">{{ s.brokerage }}</div>
          </div>
          <div class="font-medium text-right">{{ s.rating_from }} → {{ s.rating_to }}</div>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-slate-600">Target Δ</span>
          <div class="font-medium" :class="deltaClass(s.price_target_delta)">
            {{ money(s.price_target_delta) }}
          </div>
        </div>
        <div v-if="enrichList" class="mt-2 pt-2 border-t border-slate-200">
          <div class="flex justify-between items-center mb-1">
            <span class="text-slate-600">IV</span>
            <div class="font-medium">{{ money(s.intrinsic_value as any) }}</div>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-slate-600">IV (AAA)</span>
            <div class="font-medium">{{ money(s.intrinsic_value_2 as any) }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { StockItem, WatchlistItem } from '../stores/stock';

const props = defineProps<{
  items: StockItem[];
  loading: boolean;
  error: string | null;
  enrichList: boolean;
  watchlistItems: WatchlistItem[];
}>();

const emit = defineEmits<{
  goDetail: [ticker: string];
  toggleWatchlist: [ticker: string];
}>();

// Compute a fast lookup set for watchlist membership
const watchlistSet = computed(() => new Set(props.watchlistItems.map(i => i.ticker)));

function isWatched(ticker: string): boolean {
  return watchlistSet.value.has(ticker);
}

function money(v?: number | null): string {
  if (v == null) return '-';
  return `${v.toFixed(2)}`;
}

function deltaClass(v?: number | null) {
  if (v == null) return '';
  return v >= 0 ? 'text-success' : 'text-danger';
}
// Note: use emit in template via $emit; no runtime changes needed here
</script>

<style scoped>
.btn-icon {
  @apply p-1 rounded-full hover:bg-slate-100;
}
</style>
