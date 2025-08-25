<template>
  <section class="space-y-6">
    <!-- Tab navigation -->
    <div class="border-b border-slate-200">
      <nav class="flex space-x-8">
        <button
          @click="activeTab = 'all'"
          :class="[
            'py-4 px-1 border-b-2 font-medium text-sm',
            activeTab === 'all' 
              ? 'border-brand-500 text-brand-600' 
              : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
          ]"
        >
          All Stocks
        </button>
        <button
          @click="activeTab = 'watchlist'"
          :class="[
            'py-4 px-1 border-b-2 font-medium text-sm',
            activeTab === 'watchlist' 
              ? 'border-brand-500 text-brand-600' 
              : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
          ]"
        >
          Watchlist
        </button>
        <button
          @click="activeTab = 'portfolio'"
          :class="[
            'py-4 px-1 border-b-2 font-medium text-sm',
            activeTab === 'portfolio' 
              ? 'border-brand-500 text-brand-600' 
              : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
          ]"
        >
          My Portfolio
        </button>
      </nav>
    </div>

    <!-- Tab content -->
    <div v-if="activeTab === 'all'">
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
        <div class="flex gap-3 items-center">
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
          <label class="inline-flex items-center gap-2 text-sm text-slate-600">
            <input type="checkbox" v-model="enrichList" />
            Show fundamentals
          </label>
        </div>
      </div>
    </div>

    <div v-else-if="activeTab === 'watchlist'">
      <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div class="relative w-full md:w-80">
          <input
            v-model="watchlistSearch"
            type="text"
            placeholder="Search watchlist..."
            class="input pl-10 w-full"
          />
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
        <div class="flex gap-3 items-center">
          <label class="inline-flex items-center gap-2 text-sm text-slate-600">
            <input type="checkbox" v-model="enrichList" />
            Show fundamentals
          </label>
        </div>
      </div>
    </div>

    <div v-else-if="activeTab === 'portfolio'">
      <div class="card rounded-xl p-6 mb-6">
        <h2 class="text-xl font-bold mb-4">My Investment Portfolio</h2>
        <p class="text-slate-600 mb-4">Track your personal investments and see how they compare to our stock recommendations.</p>
        
        <!-- Portfolio entry form -->
        <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1">Ticker</label>
            <input 
              v-model="portfolioEntry.ticker" 
              type="text" 
              placeholder="e.g. NVDA" 
              class="input w-full"
              @keyup.enter="addPortfolioItem"
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
              @keyup.enter="addPortfolioItem"
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
              @keyup.enter="addPortfolioItem"
            />
          </div>
          <div class="flex items-end">
            <button 
              @click="addPortfolioItem"
              class="btn w-full"
            >
              Add Investment
            </button>
          </div>
        </div>
      </div>
      
      <!-- Portfolio items -->
      <div v-if="portfolioItems.length > 0" class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
        <div
          v-for="(item, index) in portfolioItems"
          :key="index"
          class="card rounded-xl transition-all hover:shadow-lg hover:-translate-y-1 flex flex-col"
        >
          <div class="p-5 flex-grow">
            <div class="flex items-center justify-between mb-3">
              <div class="text-xl font-bold text-brand-600 hover:text-brand-700">{{ item.ticker }}</div>
              <button @click="removePortfolioItem(index)" class="text-slate-400 hover:text-danger">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
                </svg>
              </button>
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
        No investments added yet. Start by adding your first investment above.
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
        class="card rounded-xl transition-all hover:shadow-lg hover:-translate-y-1 flex flex-col"
      >
        <div class="p-5 flex-grow" @click="goDetail(s.ticker)">
          <div class="flex items-center mb-3">
            <div class="text-xl font-bold text-brand-600 hover:text-brand-700 mr-3">{{ s.ticker }}</div>
            <span class="badge">{{ s.action }}</span>
          </div>
          <div class="text-sm text-slate-700 truncate mb-1" :title="s.company">{{ s.company }}</div>
        </div>
        <div class="px-5 py-4 bg-slate-50 border-t border-slate-100 text-xs">
          <div class="flex items-center justify-between mb-2">
            <div class="flex items-center">
              <button @click.stop="toggleWatchlist(s.ticker)" class="btn-icon mr-2">
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

    <div v-if="activeTab === 'all'" class="flex flex-col sm:flex-row items-center justify-between gap-4">
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
    <div v-else class="text-sm text-slate-600">
      Showing {{ items.length }} watchlisted items
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useStockStore } from '../stores/stock';
import { storeToRefs } from 'pinia';
import axios from 'axios';

const router = useRouter();
const stockStore = useStockStore();
const { stocks, watchlist } = storeToRefs(stockStore);

const activeTab = ref('all');
const search = ref('');
const watchlistSearch = ref('');
const sort = ref('updated_at');
const order = ref<'ASC' | 'DESC'>('DESC');
const page = ref(1);
const pageSize = ref(20);
const enrichList = ref(false);

// Portfolio state
const portfolioEntry = ref({
  ticker: '',
  shares: 0,
  avgPrice: 0
});

const portfolioItems = ref<any[]>([]);

const items = computed(() => {
  if (activeTab.value === 'watchlist') {
    // Filter stocks that are in the watchlist
    const watchlistTickers = new Set(watchlistItems.value.map(item => item.ticker));
    let filtered = stocks.value.items.filter(stock => watchlistTickers.has(stock.ticker));
    
    // Apply search filter for watchlist
    if (watchlistSearch.value.trim()) {
      const searchTerm = watchlistSearch.value.toLowerCase();
      filtered = filtered.filter(stock => 
        stock.ticker.toLowerCase().includes(searchTerm) || 
        stock.company.toLowerCase().includes(searchTerm)
      );
    }
    
    return filtered;
  }
  
  return stocks.value.items;
});
const total = computed(() => stocks.value.total);
const loading = computed(() => stocks.value.loading);
const error = computed(() => stocks.value.error);

const watchlistItems = computed(() => watchlist.value.items);

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

function money(v?: number | null): string {
  if (v == null) return '-';
  return `${v.toFixed(2)}`;
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
  if (activeTab.value === 'all') {
    await stockStore.fetchStocks({
      search: search.value,
      sort: sort.value,
      order: order.value,
      page: page.value,
      pageSize: pageSize.value,
      enrich: enrichList.value
    });
  } else {
    // For watchlist, we'll fetch all stocks and filter on the client side
    await stockStore.fetchStocks({
      search: '',
      sort: 'updated_at',
      order: 'DESC',
      page: 1,
      pageSize: 1000, // Fetch more items for watchlist
      enrich: enrichList.value
    });
  }
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
watch([search, sort, order, enrichList, watchlistSearch, activeTab], () => {
  clearTimeout(t);
  t = setTimeout(() => {
    page.value = 1;
    load();
  }, 300);
});

onMounted(async () => {
  await Promise.all([
    load(),
    stockStore.fetchWatchlist(),
  ]);
  
  // Load portfolio from localStorage
  const savedPortfolio = localStorage.getItem('stockPortfolio');
  if (savedPortfolio) {
    try {
      portfolioItems.value = JSON.parse(savedPortfolio);
      // Update portfolio prices
      updatePortfolioPrices();
    } catch (e) {
      console.error('Failed to load portfolio', e);
    }
  }
});

// Watch for changes in the watchlist and reload if we're on the watchlist tab
watch(watchlistItems, () => {
  if (activeTab.value === 'watchlist') {
    // Force a reload of the watchlist items
    load();
  }
});

// Portfolio methods
function addPortfolioItem() {
  if (!portfolioEntry.value.ticker || portfolioEntry.value.shares <= 0 || portfolioEntry.value.avgPrice <= 0) {
    return;
  }
  
  const newItem = {
    ticker: portfolioEntry.value.ticker.toUpperCase(),
    shares: parseFloat(portfolioEntry.value.shares.toString()),
    avgPrice: parseFloat(portfolioEntry.value.avgPrice.toString()),
    currentPrice: null,
    pnl: undefined
  };
  
  portfolioItems.value.push(newItem);
  savePortfolio();
  
  // Reset form
  portfolioEntry.value = {
    ticker: '',
    shares: 0,
    avgPrice: 0
  };
  
  // Update prices
  updatePortfolioPrices();
}

function removePortfolioItem(index: number) {
  portfolioItems.value.splice(index, 1);
  savePortfolio();
}

function savePortfolio() {
  localStorage.setItem('stockPortfolio', JSON.stringify(portfolioItems.value));
}

async function updatePortfolioPrices() {
  // Update prices for all portfolio items
  for (const item of portfolioItems.value) {
    try {
      const response = await axios.get(`/api/stocks/${item.ticker}`);
      const data = response.data;
      if (data.current_price) {
        item.currentPrice = data.current_price;
        item.pnl = (data.current_price - item.avgPrice) * item.shares;
      }
    } catch (e) {
      console.error(`Failed to fetch price for ${item.ticker}`, e);
    }
  }
  
  savePortfolio();
}
</script>

<style scoped>
.btn-icon {
  @apply p-1 rounded-full hover:bg-slate-100;
}
</style>
