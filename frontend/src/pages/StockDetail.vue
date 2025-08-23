<template>
  <section class="space-y-6">
    <div>
      <router-link to="/" class="text-sm text-brand-600 hover:text-brand-700 flex items-center gap-1">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
        </svg>
        Back to stocks
      </router-link>
    </div>

    <div v-if="loading" class="card rounded-xl flex items-center justify-center py-24">
      <div class="flex items-center text-slate-600">
        <svg class="animate-spin h-6 w-6 text-brand-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <span class="ml-3">Loading stock details…</span>
      </div>
    </div>
    <div v-else-if="error" class="card rounded-xl text-danger text-center py-24">
      {{ error }}
    </div>
    <div v-else-if="!item" class="card rounded-xl text-slate-600 text-center py-24">
      Stock not found
    </div>
    <div v-else>
      <div class="flex flex-col md:flex-row md:items-start gap-8">
        <div class="w-full md:w-1/3 lg:w-1/4">
          <div class="card rounded-xl p-6">
            <div class="flex items-center gap-4 mb-2">
              <div class="text-4xl font-bold text-slate-900">{{ item.ticker }}</div>
              <span class="badge-lg">{{ item.action }}</span>
            </div>
            <div class="text-lg text-slate-700 mb-6">{{ item.company }}</div>
            
            <div class="space-y-4 text-sm">
              <div class="flex justify-between items-center">
                <span class="text-slate-600">Brokerage</span>
                <span class="font-medium text-slate-900">{{ item.brokerage }}</span>
              </div>
              <div class="flex justify-between items-center">
                <span class="text-slate-600">Last Updated</span>
                <span class="font-medium text-slate-900 text-right">{{ formatDate(item.updated_at) }}</span>
              </div>
            </div>
          </div>
        </div>
        
        <div class="w-full md:w-2/3 lg:w-3/4">
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="card rounded-xl p-6">
              <h3 class="font-bold text-slate-800 mb-4">Rating Change</h3>
              <div class="flex items-center justify-center text-center">
                <div class="w-1/2">
                  <div class="text-xs text-slate-500 mb-1">From</div>
                  <div class="text-2xl font-bold text-slate-800">{{ item.rating_from }}</div>
                </div>
                <div class="w-auto px-2">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
                  </svg>
                </div>
                <div class="w-1/2">
                  <div class="text-xs text-slate-500 mb-1">To</div>
                  <div class="text-2xl font-bold text-slate-800">{{ item.rating_to }}</div>
                </div>
              </div>
            </div>
            
            <div class="card rounded-xl p-6">
              <h3 class="font-bold text-slate-800 mb-4">Target Price</h3>
              <div class="flex items-center justify-center text-center">
                <div class="w-1/2">
                  <div class="text-xs text-slate-500 mb-1">From</div>
                  <div class="text-2xl font-bold text-slate-800">{{ money(item.target_from) }}</div>
                </div>
                 <div class="w-auto px-2">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
                  </svg>
                </div>
                <div class="w-1/2">
                  <div class="text-xs text-slate-500 mb-1">To</div>
                  <div class="text-2xl font-bold text-slate-800">{{ money(item.target_to) }}</div>
                </div>
              </div>
            </div>
            
            <div class="card rounded-xl p-6 lg:col-span-2">
               <h3 class="font-bold text-slate-800 mb-4">Price Delta</h3>
               <div class="text-4xl font-bold text-center" :class="deltaClass(item.price_target_delta)">
                 {{ money(item.price_target_delta) }}
               </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useStockStore } from '../stores/stock';
import { storeToRefs } from 'pinia';

const route = useRoute();
const stockStore = useStockStore();
const { detail } = storeToRefs(stockStore);

const item = computed(() => detail.value.item);
const loading = computed(() => detail.value.loading);
const error = computed(() => detail.value.error);

function money(v?: number | null): string {
  if (v == null) return '-';
  return `$${v.toFixed(2)}`;
}
function deltaClass(v?: number | null) {
  if (v == null) return '';
  return v >= 0 ? 'text-success' : 'text-danger';
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
}

async function load() {
  const ticker = route.params.ticker as string;
  if (!ticker) return;
  await stockStore.fetchStockDetail(ticker);
}

onMounted(load);
watch(() => route.params.ticker, load);
</script>
