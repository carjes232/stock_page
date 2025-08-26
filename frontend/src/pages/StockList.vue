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
        <button
          @click="activeTab = 'demo'"
          :class="[
            'py-4 px-1 border-b-2 font-medium text-sm',
            activeTab === 'demo' 
              ? 'border-brand-500 text-brand-600' 
              : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
          ]"
        >
          My Portfolio Demo
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
      <WatchlistGrid
        :items="watchlistItems"
        :loading="watchlistLoading"
        :error="watchlistError"
        @go-detail="goDetail"
        @toggle-watchlist="toggleWatchlist"
      />
    </div>

    <div v-else-if="activeTab === 'portfolio'">
      <PortfolioSummary
        title="Portfolio"
        :portfolio-items="portfolioStore.portfolioItems"
        :total-investment="portfolioStore.totalInvestment"
        :total-pn-l="portfolioStore.totalPnL"
        :biggest-winner="portfolioStore.biggestWinner"
        :biggest-loser="portfolioStore.biggestLoser"
      />

      <div class="card rounded-xl p-6 mb-6">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-bold">My Investment Portfolio</h2>
          <button 
            v-if="portfolioStore.portfolioItems.length > 0" 
            @click="portfolioStore.showPortfolioForm = !portfolioStore.showPortfolioForm"
            class="btn btn-secondary"
          >
            {{ portfolioStore.showPortfolioForm ? 'Hide Form' : 'Show Form' }}
          </button>
        </div>
        <p class="text-slate-600 mb-4">Track your personal investments and see how they compare to our stock recommendations.</p>
        
        <div v-show="portfolioStore.showPortfolioForm || portfolioStore.portfolioItems.length === 0">
          <PortfolioUpload
            title="Upload Portfolio Image"
            upload-text="Drag and drop your portfolio image here, or"
            loading-text="Processing portfolio image..."
            :is-drag-over="portfolioStore.isDragOver"
            :portfolio-uploading="portfolioStore.portfolioUploading"
            :portfolio-upload-error="portfolioStore.portfolioUploadError"
            @drag-over="portfolioStore.handleDragOver"
            @drag-enter="portfolioStore.handleDragEnter"
            @drag-leave="portfolioStore.handleDragLeave"
            @drop="portfolioStore.handleDrop"
            @file-select="portfolioStore.handleFileSelect"
          />
          
          <PortfolioForm
            form-title="Or Add Manually"
            :button-text="portfolioStore.editingIndex !== null ? 'Update Investment' : 'Add Investment'"
            :portfolio-entry="portfolioStore.portfolioEntry"
            @submit="portfolioStore.addPortfolioItem"
          />
        </div>
      </div>
      
      <PortfolioGrid
        :portfolio-items="portfolioStore.portfolioItems"
        empty-message="No investments added yet. Start by adding your first investment above."
        @go-detail="goDetail"
        @add-more-shares="portfolioStore.addMoreShares"
        @sell-shares="portfolioStore.sellShares"
        @edit-item="portfolioStore.editPortfolioItem"
        @remove-item="portfolioStore.removePortfolioItem"
      />
    </div>

    <div v-else-if="activeTab === 'demo'">
      <PortfolioSummary
        title="Demo Portfolio"
        :portfolio-items="portfolioStore.demoPortfolioItems"
        :total-investment="portfolioStore.demoTotalInvestment"
        :total-pn-l="portfolioStore.demoPnL"
        :biggest-winner="portfolioStore.demoBiggestWinner"
        :biggest-loser="portfolioStore.demoBiggestLoser"
      />

      <div class="card rounded-xl p-6 mb-6">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-bold">Demo Portfolio</h2>
          <button 
            v-if="portfolioStore.demoPortfolioItems.length > 0" 
            @click="portfolioStore.showDemoPortfolioForm = !portfolioStore.showDemoPortfolioForm"
            class="btn btn-secondary"
          >
            {{ portfolioStore.showDemoPortfolioForm ? 'Hide Form' : 'Show Form' }}
          </button>
        </div>
        <p class="text-slate-600 mb-4">Practice with a demo portfolio to learn how our platform works without using real money.</p>
        
        <div v-show="portfolioStore.showDemoPortfolioForm || portfolioStore.demoPortfolioItems.length === 0">
          <PortfolioUpload
            title="Upload Demo Portfolio Image"
            upload-text="Drag and drop your demo portfolio image here, or"
            loading-text="Processing demo portfolio image..."
            :is-demo="true"
            :is-drag-over="portfolioStore.isDragOver"
            :portfolio-uploading="portfolioStore.portfolioUploading"
            :portfolio-upload-error="portfolioStore.portfolioUploadError"
            @drag-over="portfolioStore.handleDragOver"
            @drag-enter="portfolioStore.handleDragEnter"
            @drag-leave="portfolioStore.handleDragLeave"
            @drop="portfolioStore.handleDrop"
            @file-select="portfolioStore.handleFileSelect"
          />
          
          <PortfolioForm
            form-title="Or Add Demo Investment Manually"
            :button-text="portfolioStore.editingDemoIndex !== null ? 'Update Demo Investment' : 'Add Demo Investment'"
            :portfolio-entry="portfolioStore.portfolioEntry"
            @submit="portfolioStore.addDemoPortfolioItem"
          />
        </div>
      </div>
      
      <PortfolioGrid
        :portfolio-items="portfolioStore.demoPortfolioItems"
        empty-message="No demo investments added yet. Start by adding your first demo investment above."
        @go-detail="goDetail"
        @add-more-shares="portfolioStore.addMoreDemoShares"
        @sell-shares="portfolioStore.sellDemoShares"
        @edit-item="portfolioStore.editDemoPortfolioItem"
        @remove-item="portfolioStore.removeDemoPortfolioItem"
      />
    </div>

    <StockGrid
      :items="items"
      :loading="loading"
      :error="error"
      :enrich-list="enrichList"
      :watchlist-items="watchlistItems"
      @go-detail="goDetail"
      @toggle-watchlist="toggleWatchlist"
    />

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

    <AddSharesModal
      :show="portfolioStore.addSharesModal"
      :current-item="currentPortfolioItem"
      v-model:share-amount="portfolioStore.addSharesAmount"
      :is-sell="portfolioStore.isSellMode"
      @close="portfolioStore.addSharesModal = false"
      @confirm="portfolioStore.confirmAddShares"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useStockStore } from '../stores/stock';
import { usePortfolioStore } from '../stores/portfolio';
import { storeToRefs } from 'pinia';

// Components
import PortfolioSummary from '../components/PortfolioSummary.vue';
import PortfolioUpload from '../components/PortfolioUpload.vue';
import PortfolioForm from '../components/PortfolioForm.vue';
import PortfolioGrid from '../components/PortfolioGrid.vue';
import AddSharesModal from '../components/AddSharesModal.vue';
import StockGrid from '../components/StockGrid.vue';
import WatchlistGrid from '../components/WatchlistGrid.vue';

const router = useRouter();
const stockStore = useStockStore();
const portfolioStore = usePortfolioStore();
const { stocks, watchlist } = storeToRefs(stockStore);

const activeTab = ref('all');
const search = ref('');
const sort = ref('updated_at');
const order = ref<'ASC' | 'DESC'>('DESC');
const page = ref(1);
const pageSize = ref(20);
const enrichList = ref(false);

const items = computed(() => stocks.value.items);
const total = computed(() => stocks.value.total);
const loading = computed(() => stocks.value.loading);
const error = computed(() => stocks.value.error);

const watchlistItems = computed(() => watchlist.value.items);
const watchlistLoading = computed(() => watchlist.value.loading);
const watchlistError = computed(() => watchlist.value.error);

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

// Current item for add shares modal
const currentPortfolioItem = computed(() => {
  if (portfolioStore.addSharesIndex !== null) {
    const items = portfolioStore.isDemoModal ? portfolioStore.demoPortfolioItems : portfolioStore.portfolioItems;
    return items[portfolioStore.addSharesIndex] || null;
  }
  return null;
});

async function toggleWatchlist(ticker: string) {
  const isWatched = watchlistItems.value.some(i => i.ticker === ticker);
  if (isWatched) {
    await stockStore.removeFromWatchlist(ticker);
  } else {
    await stockStore.addToWatchlist(ticker);
  }
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
watch([search, sort, order, enrichList, activeTab], () => {
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
  
  // Load portfolios from localStorage
  portfolioStore.loadPortfolio();
  portfolioStore.loadDemoPortfolio();
});

// Watch for changes in the watchlist and reload if we're on the watchlist tab
watch(watchlistItems, () => {
  if (activeTab.value === 'watchlist') {
    // Force a reload of the watchlist items
    load();
  }
});
</script>

<style scoped>
.btn-icon {
  @apply p-1 rounded-full hover:bg-slate-100;
}
</style>
