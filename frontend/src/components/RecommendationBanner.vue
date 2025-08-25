<template>
  <section class="mb-8">
    <div class="card">
      <div class="p-5 border-b border-slate-200">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-4">
            <button @click="activeTab = 'picks'" :class="['btn-tab', { 'btn-tab-active': activeTab === 'picks' }]">
              Today's Top Picks
            </button>
            <button @click="activeTab = 'watchlist'" :class="['btn-tab', { 'btn-tab-active': activeTab === 'watchlist' }]">
              My Watchlist
            </button>
          </div>
          <button
            class="btn-ghost text-sm flex items-center gap-1"
            @click="refresh"
            :disabled="loading || watchlistLoading"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {{ (loading || watchlistLoading) ? 'Refreshing…' : 'Refresh' }}
          </button>
        </div>
      </div>
      
      <div class="p-5">
        <!-- Today's Picks -->
        <div v-if="activeTab === 'picks'">
          <div v-if="error" class="text-danger text-center py-4">{{ error }}</div>
          <div v-else-if="loading" class="flex justify-center items-center py-6">
            <svg class="animate-spin h-5 w-5 text-brand-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span class="ml-2 text-slate-600">Loading recommendations…</span>
          </div>
          <div v-else-if="items.length === 0" class="text-slate-600 text-center py-6">No recommendations yet.</div>
          <ul v-else class="divide-y divide-slate-100">
            <li
              v-for="rec in items"
              :key="rec.ticker"
              class="py-4 first:pt-0 last:pb-0"
            >
              <div class="flex items-center justify-between">
                <div class="flex-grow">
                  <div class="font-medium">
                    <router-link :to="`/stock/${rec.ticker}`" class="text-brand-600 hover:text-brand-700">
                      {{ rec.ticker }}
                    </router-link>
                    <span class="text-slate-700"> · {{ rec.company }}</span>
                  </div>
                  <div class="text-xs text-slate-600 mt-1">
                    {{ rec.brokerage }} · {{ rec.rating_from }} → {{ rec.rating_to }}
                    <span v-if="rec.price_target_delta != null" :class="deltaClass(rec.price_target_delta)">
                      · Δ ${{ rec.price_target_delta?.toFixed(2) }}
                    </span>
                    <span v-if="rec.current_price != null" class="ml-1">· Price {{ money(rec.current_price) }}</span>
                    <span v-if="rec.percent_upside != null" :class="rec.percent_upside >= 0 ? 'text-success' : 'text-danger'">
                      · ↑ {{ percent(rec.percent_upside) }}
                    </span>
                    <span v-if="rec.eps != null"> · EPS {{ money(rec.eps) }}</span>
                    <span v-if="rec.intrinsic_value != null"> · IV {{ money(rec.intrinsic_value) }}</span>
                  </div>
                  <div class="text-xs text-slate-500 mt-1" v-if="rec.reasons?.length">
                    Reasons: {{ rec.reasons.join(', ') }}
                  </div>
                </div>
                <div class="text-right ml-4 flex-shrink-0">
                  <div class="text-sm font-semibold text-slate-900">Score: {{ rec.score.toFixed(2) }}</div>
                  <div class="text-xs text-slate-500">{{ formatDate(rec.updated_at) }}</div>
                </div>
                <button @click="toggleWatchlist(rec.ticker)" class="btn-icon ml-4">
                  <svg v-if="isWatched(rec.ticker)" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
                  </svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.783-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                  </svg>
                </button>
              </div>
            </li>
          </ul>
        </div>

        <!-- Watchlist -->
        <div v-if="activeTab === 'watchlist'">
          <div v-if="watchlistError" class="text-danger text-center py-4">{{ watchlistError }}</div>
          <div v-else-if="watchlistLoading" class="flex justify-center items-center py-6">
            <svg class="animate-spin h-5 w-5 text-brand-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <span class="ml-2 text-slate-600">Loading watchlist…</span>
          </div>
          <div v-else-if="watchlistItems.length === 0" class="text-slate-600 text-center py-6">Your watchlist is empty.</div>
          <ul v-else class="divide-y divide-slate-100">
            <li
              v-for="item in watchlistItems"
              :key="item.ticker"
              class="py-4 first:pt-0 last:pb-0"
            >
              <div class="flex items-center justify-between">
                <div class="flex-grow">
                  <div class="font-medium">
                    <router-link :to="`/stock/${item.ticker}`" class="text-brand-600 hover:text-brand-700">
                      {{ item.ticker }}
                    </router-link>
                  </div>
                  <div class="text-xs text-slate-500 mt-1">
                    Added on {{ formatDate(item.added_at) }}
                  </div>
                </div>
                <button @click="toggleWatchlist(item.ticker)" class="btn-icon ml-4">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
                  </svg>
                </button>
              </div>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useStockStore } from '../stores/stock';
import { storeToRefs } from 'pinia';

const stockStore = useStockStore();
const { recommendations, watchlist } = storeToRefs(stockStore);

const activeTab = ref('picks');

const items = computed(() => recommendations.value.items);
const loading = computed(() => recommendations.value.loading);
const error = computed(() => recommendations.value.error);

const watchlistItems = computed(() => watchlist.value.items);
const watchlistLoading = computed(() => watchlist.value.loading);
const watchlistError = computed(() => watchlist.value.error);

function isWatched(ticker: string): boolean {
  return watchlistItems.value.some(i => i.ticker === ticker);
}

async function toggleWatchlist(ticker: string) {
  if (isWatched(ticker)) {
    await stockStore.removeFromWatchlist(ticker);
  } else {
    await stockStore.addToWatchlist(ticker);
  }
}

function deltaClass(v?: number | null) {
  if (v == null) return '';
  return v >= 0 ? 'text-success' : 'text-danger';
}

function money(v?: number | null): string {
  if (v == null) return '-';
  return `${v.toFixed(2)}`;
}

function percent(v?: number | null): string {
  if (v == null) return '-';
  return `${(v * 100).toFixed(1)}%`;
}

function formatDate(dateString: string): string {
  if (!dateString) return '';
  return new Date(dateString).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric'
  });
}

async function refresh() {
  if (activeTab.value === 'picks') {
    await stockStore.fetchRecommendations();
  } else {
    await stockStore.fetchWatchlist();
  }
}

onMounted(async () => {
  await Promise.all([
    stockStore.fetchRecommendations(),
    stockStore.fetchWatchlist(),
  ]);
});
</script>

<style scoped>
.btn-tab {
  @apply text-slate-600 font-semibold pb-2 border-b-2 border-transparent;
}
.btn-tab-active {
  @apply text-brand-600 border-brand-600;
}
.btn-icon {
  @apply p-1 rounded-full hover:bg-slate-100;
}
</style>
