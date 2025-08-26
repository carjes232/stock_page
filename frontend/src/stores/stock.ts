import { defineStore } from 'pinia';
import axios from 'axios';

export type StockItem = {
  id: string;
  ticker: string;
  company: string;
  brokerage: string;
  action: string;
  rating_from: string;
  rating_to: string;
  target_from?: number | null;
  target_to?: number | null;
  price_target_delta?: number | null;
  current_price?: number | null;
  percent_upside?: number | null;
  eps?: number | null;
  growth?: number | null;
  intrinsic_value?: number | null;
  intrinsic_value_2?: number | null;
  updated_at: string;
};

export type RecItem = {
  ticker: string;
  company: string;
  brokerage: string;
  rating_from: string;
  rating_to: string;
  target_from?: number | null;
  target_to?: number | null;
  price_target_delta?: number | null;
  current_price?: number | null;
  percent_upside?: number | null;
  eps?: number | null;
  growth?: number | null;
  intrinsic_value?: number | null;
  intrinsic_value_2?: number | null;
  score: number;
  reasons: string[];
  updated_at: string;
};

export type WatchlistItem = {
  ticker: string;
  notes: string | null;
  added_at: string;
}

export const useStockStore = defineStore('stock', {
  state: () => ({
    stocks: {
      items: [] as StockItem[],
      total: 0,
      loading: false,
      error: null as string | null,
    },
    recommendations: {
      items: [] as RecItem[],
      loading: false,
      error: null as string | null,
    },
    watchlist: {
      items: [] as WatchlistItem[],
      loading: false,
      error: null as string | null,
    },
    detail: {
      item: null as StockItem | null,
      loading: false,
      error: null as string | null,
    },
  }),
  actions: {
    async fetchStocks(params: { search: string; sort: string; order: 'ASC' | 'DESC'; page: number; pageSize: number; enrich?: boolean }) {
      this.stocks.loading = true;
      this.stocks.error = null;
      try {
        const search = (params.search || '').trim();
        const field = (params.sort || '').trim() || 'updated_at';
        const order = params.order || 'DESC';
        const page = params.page || 1;
        const limit = params.pageSize || 20;
        const enrich = params.enrich === true;

        let url = '/api/stocks';
        const query: Record<string, any> = { page, limit };
        if (enrich) query.enrich = 'true';

        if (search) {
          url = '/api/stocks/search';
          query.q = search;
        } else if (field !== 'updated_at' || order !== 'DESC') {
          url = '/api/stocks/sort';
          query.field = field;
          query.order = order;
        }

        const { data } = await axios.get(url, { params: query, timeout: enrich ? 15000 : 5000 });
        this.stocks.items = data.items ?? [];
        this.stocks.total = data.total ?? this.stocks.items.length;
      } catch (e: any) {
        this.stocks.error = e?.message || 'Failed to load';
      } finally {
        this.stocks.loading = false;
      }
    },
    async fetchRecommendations() {
      this.recommendations.loading = true;
      this.recommendations.error = null;
      try {
        // Bound the wait so UI doesn't spin forever on slow upstreams
        const { data } = await axios.get('/api/recommendations', { timeout: 5000 });
        this.recommendations.items = data.items ?? [];
      } catch (e: any) {
        this.recommendations.error = e?.message || 'Failed to load';
      } finally {
        this.recommendations.loading = false;
      }
    },
    async fetchStockDetail(ticker: string) {
      this.detail.loading = true;
      this.detail.error = null;
      try {
        // First try the stocks endpoint (for stocks with recommendations)
        const { data } = await axios.get(`/api/stocks/${ticker}`);
        this.detail.item = data;
      } catch (e: any) {
        try {
          // If stocks endpoint fails (e.g., ETFs not in stocks table), try quotes endpoint
          const quoteResponse = await axios.get(`/api/quotes/${ticker}`);
          const quoteData = quoteResponse.data;
          
          // Transform quote data to match StockItem interface
          this.detail.item = {
            id: ticker,
            ticker: ticker,
            company: quoteData.name || ticker,
            brokerage: 'N/A',
            action: 'N/A',
            rating_from: 'N/A',
            rating_to: 'N/A',
            target_from: null,
            target_to: null,
            price_target_delta: null,
            current_price: quoteData.current_price || null,
            percent_upside: null,
            eps: quoteData.eps || null,
            growth: null,
            intrinsic_value: null,
            intrinsic_value_2: null,
            updated_at: new Date().toISOString()
          };
        } catch (quoteError: any) {
          console.error(`Failed to fetch detail for ${ticker}`, quoteError);
          this.detail.error = `Stock not found in our database. This could be an ETF or a ticker not covered in our recommendations.`;
        }
      } finally {
        this.detail.loading = false;
      }
    },
    async fetchWatchlist() {
      this.watchlist.loading = true;
      this.watchlist.error = null;
      try {
        const { data } = await axios.get('/api/watchlist');
        this.watchlist.items = data.items ?? [];
      } catch (e: any) {
        this.watchlist.error = e?.message || 'Failed to load';
      } finally {
        this.watchlist.loading = false;
      }
    },
    async addToWatchlist(ticker: string) {
      try {
        await axios.post('/api/watchlist', { ticker });
        await this.fetchWatchlist(); // Refresh watchlist
      } catch (e: any) {
        console.error('Failed to add to watchlist', e);
      }
    },
    async removeFromWatchlist(ticker: string) {
      try {
        await axios.delete(`/api/watchlist/${ticker}`);
        await this.fetchWatchlist(); // Refresh watchlist
      } catch (e: any) {
        console.error('Failed to remove from watchlist', e);
      }
    },
  },
});
