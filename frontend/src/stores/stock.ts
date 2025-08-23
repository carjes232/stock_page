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
    detail: {
      item: null as StockItem | null,
      loading: false,
      error: null as string | null,
    },
  }),
  actions: {
    async fetchStocks(params: { search: string; sort: string; order: 'ASC' | 'DESC'; page: number; pageSize: number }) {
      this.stocks.loading = true;
      this.stocks.error = null;
      try {
        const search = (params.search || '').trim();
        const field = (params.sort || '').trim() || 'updated_at';
        const order = params.order || 'DESC';
        const page = params.page || 1;
        const limit = params.pageSize || 20;

        let url = '/api/stocks';
        const query: Record<string, any> = { page, limit };

        if (search) {
          url = '/api/stocks/search';
          query.q = search;
        } else if (field !== 'updated_at' || order !== 'DESC') {
          url = '/api/stocks/sort';
          query.field = field;
          query.order = order;
        }

        const { data } = await axios.get(url, { params: query });
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
        const { data } = await axios.get('/api/recommendations');
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
        const { data } = await axios.get(`/api/stocks/${ticker}`);
        this.detail.item = data;
      } catch (e: any) {
        this.detail.error = e?.message || 'Failed to load';
      } finally {
        this.detail.loading = false;
      }
    },
  },
});
