<template>
  <section class="space-y-6">
    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
      <div class="relative w-full md:w-80">
        <input
          v-model="search"
          type="text"
          placeholder="Search by ticker, company, or brokerage…"
          class="input pl-10 w-full"
        />
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
      </div>
      <div class="flex gap-3">
        <select v-model="sort" class="select w-40">
          <option value="updated_at">Updated</option>
          <option value="ticker">Ticker</option>
          <option value="company">Company</option>
          <option value="brokerage">Brokerage</option>
          <option value="price_target_delta">Δ Price Target</option>
          <option value="rating_to">Rating To</option>
          <option value="rating_from">Rating From</option>
        </select>
        <select v-model="order" class="select w-24">
          <option value="DESC">Desc</option>
          <option value="ASC">Asc</option>
        </select>
      </div>
    </div>

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
        class="card rounded-xl cursor-pointer transition-all hover:shadow-lg hover:-translate-y-1"
        @click="goDetail(s.ticker)"
      >
        <div class="p-5">
          <div class="flex items-center justify-between mb-3">
            <div class="text-lg font-bold text-brand-600 hover:text-brand-700">{{ s.ticker }}</div>
            <span class="badge">{{ s.action }}</span>
          </div>
          <div class="text-sm text-slate-700 truncate mb-1" :title="s.company">{{ s.company }}</div>
          <div class="text-xs text-slate-500 truncate" :title="s.brokerage">{{ s.brokerage }}</div>
        </div>
        <div class="px-5 py-4 bg-slate-50 border-t border-slate-100 text-xs">
          <div class="flex justify-between items-center mb-2">
            <span class="text-slate-600">Rating</span>
            <div class="font-medium">{{ s.rating_from }} → {{ s.rating_to }}</div>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-slate-600">Target Δ</span>
            <div class="font-medium" :class="deltaClass(s.price_target_delta)">
              {{ money(s.price_target_delta) }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="flex flex-col sm:flex-row items-center justify-between gap-4">
      <div class="text-sm text-slate-600">
        Showing {{ startIndex }} to {{ endIndex }} of {{ total }} items
      </div>
      <div class="flex gap-2">
        <button 
          class="pagination-btn" 
          :disabled="page === 1" 
          @click="prev"
        >
          Previous
        </button>
        <button 
          v-for="pageNum in visiblePages" 
          :key="pageNum"
          class="pagination-btn"
          :class="{ 'active': pageNum === page }"
          @click="goToPage(pageNum)"
        >
          {{ pageNum }}
        </button>
        <button 
          class="pagination-btn" 
          :disabled="page === totalPages" 
          @click="next"
        >
          Next
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useStockStore } from '../stores/stock';
import { storeToRefs } from 'pinia';

const router = useRouter();
const stockStore = useStockStore();
const { stocks } = storeToRefs(stockStore);

const search = ref('');
const sort = ref('updated_at');
const order = ref<'ASC' | 'DESC'>('DESC');
const page = ref(1);
const pageSize = ref(20);

const items = computed(() => stocks.value.items);
const total = computed(() => stocks.value.total);
const loading = computed(() => stocks.value.loading);
const error = computed(() => stocks.value.error);

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)));

const startIndex = computed(() => (page.value - 1) * pageSize.value + 1);
const endIndex = computed(() => Math.min(page.value * pageSize.value, total.value));

const visiblePages = computed(() => {
  const pages = [];
  const maxVisible = 5;
  let start = Math.max(1, page.value - Math.floor(maxVisible / 2));
  let end = Math.min(totalPages.value, start + maxVisible - 1);

  if (end - start + 1 < maxVisible) {
    start = Math.max(1, end - maxVisible + 1);
  }

  for (let i = start; i <= end; i++) {
    pages.push(i);
  }

  return pages;
});

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
    month: 'short',
    day: 'numeric'
  });
}

async function load() {
  await stockStore.fetchStocks({
    search: search.value,
    sort: sort.value,
    order: order.value,
    page: page.value,
    pageSize: pageSize.value
  });
}

function goDetail(ticker: string) {
  router.push({ name: 'stock-detail', params: { ticker } });
}

function prev() {
  if (page.value > 1) {
    page.value -= 1;
    load();
  }
}

function next() {
  if (page.value < totalPages.value) {
    page.value += 1;
    load();
  }
}

function goToPage(pageNum: number) {
  page.value = pageNum;
  load();
}

let t: any;
watch([search, sort, order], () => {
  clearTimeout(t);
  t = setTimeout(() => {
    page.value = 1;
    load();
  }, 300);
});

onMounted(load);
</script>
