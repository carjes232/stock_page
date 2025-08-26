<template>
  <div v-if="loading" class="card rounded-xl flex items-center justify-center py-12">
    <div class="flex items-center text-slate-600">
      <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-brand-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
      <span>Loading watchlist…</span>
    </div>
  </div>
  <div v-else-if="error" class="card rounded-xl text-danger text-center py-12">
    {{ error }}
  </div>
  <div v-else-if="items.length === 0" class="card rounded-xl text-slate-600 text-center py-12">
    Your watchlist is empty.
  </div>
  <ul v-else class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
    <li
      v-for="item in items"
      :key="item.ticker"
      class="card rounded-xl transition-all hover:shadow-lg hover:-translate-y-1 flex flex-col cursor-pointer"
      @click="$emit('goDetail', item.ticker)"
    >
      <div class="p-5 flex-grow">
        <div class="flex items-center justify-between mb-3">
          <div class="text-xl font-bold text-brand-600 hover:text-brand-700">{{ item.ticker }}</div>
          <button @click.stop="$emit('toggleWatchlist', item.ticker)" class="btn-icon">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
            </svg>
          </button>
        </div>
        <div class="text-sm text-slate-600 mb-2">
          Added on {{ formatDate(item.added_at) }}
        </div>
      </div>
    </li>
  </ul>
</template>

<script setup lang="ts">
import type { WatchlistItem } from '../stores/stock';

defineProps<{
  items: WatchlistItem[];
  loading: boolean;
  error: string | null;
}>();

defineEmits<{
  goDetail: [ticker: string];
  toggleWatchlist: [ticker: string];
}>();

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  });
}
</script>

<style scoped>
.btn-icon {
  @apply p-1 rounded-full hover:bg-slate-100;
}
</style>