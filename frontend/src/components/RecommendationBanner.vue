<template>
  <section class="mb-8">
    <div class="card">
      <div class="p-5 border-b border-slate-200">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-bold text-slate-900">Today's Top Picks</h2>
          <button
            class="btn-ghost text-sm flex items-center gap-1"
            @click="refresh"
            :disabled="loading"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {{ loading ? 'Refreshing…' : 'Refresh' }}
          </button>
        </div>
      </div>
      
      <div class="p-5">
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
              <div>
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
                </div>
                <div class="text-xs text-slate-500 mt-1" v-if="rec.reasons?.length">
                  Reasons: {{ rec.reasons.join(', ') }}
                </div>
              </div>
              <div class="text-right ml-4">
                <div class="text-sm font-semibold text-slate-900">Score: {{ rec.score.toFixed(2) }}</div>
                <div class="text-xs text-slate-500">{{ formatDate(rec.updated_at) }}</div>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useStockStore } from '../stores/stock';
import { storeToRefs } from 'pinia';

const stockStore = useStockStore();
const { recommendations } = storeToRefs(stockStore);

const items = computed(() => recommendations.value.items);
const loading = computed(() => recommendations.value.loading);
const error = computed(() => recommendations.value.error);

function deltaClass(v?: number | null) {
  if (v == null) return '';
  return v >= 0 ? 'text-success' : 'text-danger';
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric'
  });
}

async function refresh() {
  await stockStore.fetchRecommendations();
}

onMounted(refresh);
</script>
