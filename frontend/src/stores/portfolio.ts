import { defineStore } from 'pinia';
import axios from 'axios';

export type PortfolioItem = {
  ticker: string;
  shares: number;
  avgPrice: number;
  currentPrice: number | null;
  pnl: number | undefined;
};

export type PortfolioEntry = {
  ticker: string;
  shares: number;
  avgPrice: number;
};

export const usePortfolioStore = defineStore('portfolio', {
  state: () => ({
    // Real Portfolio
    portfolioItems: [] as PortfolioItem[],
    showPortfolioForm: true,
    editingIndex: null as number | null,
    portfolioEntry: {
      ticker: '',
      shares: 0,
      avgPrice: 0
    } as PortfolioEntry,

    // Demo Portfolio
    demoPortfolioItems: [] as PortfolioItem[],
    showDemoPortfolioForm: true,
    editingDemoIndex: null as number | null,

    // Upload state
    portfolioUploading: false,
    portfolioUploadError: null as string | null,
    isDragOver: false,

    // Add shares modal
    addSharesModal: false,
    addSharesIndex: null as number | null,
    addSharesAmount: 0,
    isDemoModal: false,
    isSellMode: false,
  }),

  getters: {
    // Real Portfolio Summary
    totalPnL: (state) => {
      return state.portfolioItems
        .filter(item => item.pnl !== undefined)
        .reduce((sum, item) => sum + item.pnl!, 0);
    },

    totalInvestment: (state) => {
      return state.portfolioItems.reduce((sum, item) => sum + (item.shares * item.avgPrice), 0);
    },

    biggestWinner: (state) => {
      const validItems = state.portfolioItems.filter(item => item.pnl !== undefined && item.pnl > 0);
      if (validItems.length === 0) return null;
      return validItems.reduce((max, item) => item.pnl! > max.pnl! ? item : max);
    },

    biggestLoser: (state) => {
      const validItems = state.portfolioItems.filter(item => item.pnl !== undefined && item.pnl < 0);
      if (validItems.length === 0) return null;
      return validItems.reduce((min, item) => item.pnl! < min.pnl! ? item : min);
    },

    // Demo Portfolio Summary
    demoPnL: (state) => {
      return state.demoPortfolioItems
        .filter(item => item.pnl !== undefined)
        .reduce((sum, item) => sum + item.pnl!, 0);
    },

    demoTotalInvestment: (state) => {
      return state.demoPortfolioItems.reduce((sum, item) => sum + (item.shares * item.avgPrice), 0);
    },

    demoBiggestWinner: (state) => {
      const validItems = state.demoPortfolioItems.filter(item => item.pnl !== undefined && item.pnl > 0);
      if (validItems.length === 0) return null;
      return validItems.reduce((max, item) => item.pnl! > max.pnl! ? item : max);
    },

    demoBiggestLoser: (state) => {
      const validItems = state.demoPortfolioItems.filter(item => item.pnl !== undefined && item.pnl < 0);
      if (validItems.length === 0) return null;
      return validItems.reduce((min, item) => item.pnl! < min.pnl! ? item : min);
    },
  },

  actions: {
    // Storage methods
    savePortfolio() {
      localStorage.setItem('stockPortfolio', JSON.stringify(this.portfolioItems));
    },

    saveDemoPortfolio() {
      localStorage.setItem('stockDemoPortfolio', JSON.stringify(this.demoPortfolioItems));
    },

    loadPortfolio() {
      const savedPortfolio = localStorage.getItem('stockPortfolio');
      if (savedPortfolio) {
        try {
          this.portfolioItems = JSON.parse(savedPortfolio);
          if (this.portfolioItems.length > 0) {
            this.showPortfolioForm = false;
          }
          this.updatePortfolioPrices();
        } catch (e) {
          console.error('Failed to load portfolio', e);
        }
      }
    },

    loadDemoPortfolio() {
      const savedDemoPortfolio = localStorage.getItem('stockDemoPortfolio');
      if (savedDemoPortfolio) {
        try {
          this.demoPortfolioItems = JSON.parse(savedDemoPortfolio);
          if (this.demoPortfolioItems.length > 0) {
            this.showDemoPortfolioForm = false;
          }
          this.updateDemoPortfolioPrices();
        } catch (e) {
          console.error('Failed to load demo portfolio', e);
        }
      }
    },

    // Real Portfolio methods
    addPortfolioItem() {
      if (!this.portfolioEntry.ticker || this.portfolioEntry.shares <= 0 || this.portfolioEntry.avgPrice <= 0) {
        return;
      }
      
      const itemData: PortfolioItem = {
        ticker: this.portfolioEntry.ticker.toUpperCase(),
        shares: parseFloat(this.portfolioEntry.shares.toString()),
        avgPrice: parseFloat(this.portfolioEntry.avgPrice.toString()),
        currentPrice: null,
        pnl: undefined
      };
      
      if (this.editingIndex !== null) {
        this.portfolioItems[this.editingIndex] = itemData;
        this.editingIndex = null;
      } else {
        this.portfolioItems.push(itemData);
      }
      
      this.savePortfolio();
      
      this.portfolioEntry = {
        ticker: '',
        shares: 0,
        avgPrice: 0
      };
      
      this.updatePortfolioPrices();
    },

    editPortfolioItem(index: number) {
      const item = this.portfolioItems[index];
      this.portfolioEntry = {
        ticker: item.ticker,
        shares: item.shares,
        avgPrice: item.avgPrice
      };
      
      this.editingIndex = index;
      this.showPortfolioForm = true;
      
      const formElement = document.querySelector('.border-t.border-slate-200.pt-6');
      if (formElement) {
        formElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    },

    removePortfolioItem(index: number) {
      this.portfolioItems.splice(index, 1);
      this.savePortfolio();
    },

    addMoreShares(index: number) {
      this.addSharesIndex = index;
      this.addSharesAmount = 0;
      this.isDemoModal = false;
      this.isSellMode = false;
      this.addSharesModal = true;
    },

    // Demo Portfolio methods
    addDemoPortfolioItem() {
      if (!this.portfolioEntry.ticker || this.portfolioEntry.shares <= 0 || this.portfolioEntry.avgPrice <= 0) {
        return;
      }
      
      const itemData: PortfolioItem = {
        ticker: this.portfolioEntry.ticker.toUpperCase(),
        shares: parseFloat(this.portfolioEntry.shares.toString()),
        avgPrice: parseFloat(this.portfolioEntry.avgPrice.toString()),
        currentPrice: null,
        pnl: undefined
      };
      
      if (this.editingDemoIndex !== null) {
        this.demoPortfolioItems[this.editingDemoIndex] = itemData;
        this.editingDemoIndex = null;
      } else {
        this.demoPortfolioItems.push(itemData);
      }
      
      this.saveDemoPortfolio();
      
      this.portfolioEntry = {
        ticker: '',
        shares: 0,
        avgPrice: 0
      };
      
      this.updateDemoPortfolioPrices();
    },

    editDemoPortfolioItem(index: number) {
      const item = this.demoPortfolioItems[index];
      this.portfolioEntry = {
        ticker: item.ticker,
        shares: item.shares,
        avgPrice: item.avgPrice
      };
      
      this.editingDemoIndex = index;
      this.showDemoPortfolioForm = true;
      
      const formElement = document.querySelector('.border-t.border-slate-200.pt-6');
      if (formElement) {
        formElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    },

    removeDemoPortfolioItem(index: number) {
      this.demoPortfolioItems.splice(index, 1);
      this.saveDemoPortfolio();
    },

    addMoreDemoShares(index: number) {
      this.addSharesIndex = index;
      this.addSharesAmount = 0;
      this.isDemoModal = true;
      this.isSellMode = false;
      this.addSharesModal = true;
    },

    sellShares(index: number) {
      this.addSharesIndex = index;
      this.addSharesAmount = 0;
      this.isDemoModal = false;
      this.isSellMode = true;
      this.addSharesModal = true;
    },

    sellDemoShares(index: number) {
      this.addSharesIndex = index;
      this.addSharesAmount = 0;
      this.isDemoModal = true;
      this.isSellMode = true;
      this.addSharesModal = true;
    },

    confirmAddShares() {
      if (this.addSharesIndex !== null && this.addSharesAmount > 0) {
        const items = this.isDemoModal ? this.demoPortfolioItems : this.portfolioItems;
        const item = items[this.addSharesIndex];
        
        if (!item.currentPrice || item.currentPrice <= 0) {
          alert('Current price not available for this stock. Please try again later.');
          return;
        }

        if (this.isSellMode) {
          // Selling shares: reduce quantity, average price unchanged
          if (this.addSharesAmount > item.shares) {
            alert('Cannot sell more shares than you own.');
            return;
          }
          item.shares = parseFloat((item.shares - this.addSharesAmount).toFixed(6));
          // If shares drop to zero, reset P&L
          if (item.shares === 0) {
            item.pnl = 0;
          } else {
            item.pnl = (item.currentPrice - item.avgPrice) * item.shares;
          }
        } else {
          // Buying more: recalc weighted avg price
          const oldInvestment = item.shares * item.avgPrice;
          const newInvestment = this.addSharesAmount * item.currentPrice;
          const totalShares = item.shares + this.addSharesAmount;
          const newAvgPrice = (oldInvestment + newInvestment) / totalShares;
          
          item.shares = parseFloat(totalShares.toFixed(6));
          item.avgPrice = newAvgPrice;
          item.pnl = (item.currentPrice - item.avgPrice) * item.shares;
        }

        if (this.isDemoModal) {
          this.saveDemoPortfolio();
        } else {
          this.savePortfolio();
        }
        
        this.addSharesModal = false;
        this.addSharesIndex = null;
        this.addSharesAmount = 0;
        this.isDemoModal = false;
        this.isSellMode = false;
      }
    },

    // Price update methods
    async updatePortfolioPrices() {
      for (const item of this.portfolioItems) {
        await this.updateItemPrice(item);
      }
      this.savePortfolio();
    },

    async updateDemoPortfolioPrices() {
      for (const item of this.demoPortfolioItems) {
        await this.updateItemPrice(item);
      }
      this.saveDemoPortfolio();
    },

    async updateItemPrice(item: PortfolioItem) {
      try {
        const response = await axios.get(`/api/stocks/${item.ticker}`);
        const data = response.data;
        if (data.current_price) {
          item.currentPrice = data.current_price;
          item.pnl = (data.current_price - item.avgPrice) * item.shares;
        } else {
          item.currentPrice = null;
          item.pnl = undefined;
          console.warn(`Price data not available for ${item.ticker}`);
        }
      } catch (e) {
        try {
          const quoteResponse = await axios.get(`/api/quotes/${item.ticker}`);
          const quoteData = quoteResponse.data;
          if (quoteData.current_price) {
            item.currentPrice = quoteData.current_price;
            item.pnl = (quoteData.current_price - item.avgPrice) * item.shares;
          } else {
            item.currentPrice = null;
            item.pnl = undefined;
            console.warn(`Quote data not available for ${item.ticker}`);
          }
        } catch (quoteError) {
          console.error(`Failed to fetch quote for ${item.ticker}`, quoteError);
          item.currentPrice = null;
          item.pnl = undefined;
        }
      }
    },

    // Upload methods
    async uploadPortfolioImage(file: File, isDemo = false) {
      this.portfolioUploading = true;
      this.portfolioUploadError = null;

      const formData = new FormData();
      formData.append('image', file);

      try {
        const response = await axios.post('/api/portfolio/upload', formData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
        });

        if (response.status === 200) {
          if (isDemo) {
            await this.fetchBackendDemoPortfolio();
          } else {
            await this.fetchBackendPortfolio();
          }
        }
      } catch (error: any) {
        console.error('Portfolio upload failed:', error);
        this.portfolioUploadError = error.response?.data?.error || 'Failed to upload portfolio image. Please try again.';
      } finally {
        this.portfolioUploading = false;
      }
    },

    async fetchBackendPortfolio() {
      try {
        const response = await axios.get('/api/portfolio');
        const backendPortfolio = response.data.items || [];
        
        const existingTickers = new Set(this.portfolioItems.map(item => item.ticker));
        
        for (const backendItem of backendPortfolio) {
          if (!existingTickers.has(backendItem.ticker)) {
            this.portfolioItems.push({
              ticker: backendItem.ticker,
              shares: backendItem.position,
              avgPrice: backendItem.average_price,
              currentPrice: null,
              pnl: undefined
            });
          }
        }
        
        this.savePortfolio();
        this.updatePortfolioPrices();
      } catch (error) {
        console.error('Failed to fetch backend portfolio:', error);
      }
    },

    async fetchBackendDemoPortfolio() {
      try {
        const response = await axios.get('/api/portfolio');
        const backendPortfolio = response.data.items || [];
        
        const existingTickers = new Set(this.demoPortfolioItems.map(item => item.ticker));
        
        for (const backendItem of backendPortfolio) {
          if (!existingTickers.has(backendItem.ticker)) {
            this.demoPortfolioItems.push({
              ticker: backendItem.ticker,
              shares: backendItem.position,
              avgPrice: backendItem.average_price,
              currentPrice: null,
              pnl: undefined
            });
          }
        }
        
        this.saveDemoPortfolio();
        this.updateDemoPortfolioPrices();
      } catch (error) {
        console.error('Failed to fetch backend demo portfolio:', error);
      }
    },

    // Drag & drop methods
    setDragOver(value: boolean) {
      this.isDragOver = value;
    },

    handleDragOver(event: DragEvent) {
      event.preventDefault();
      this.isDragOver = true;
    },

    handleDragEnter(event: DragEvent) {
      event.preventDefault();
      this.isDragOver = true;
    },

    handleDragLeave(event: DragEvent) {
      event.preventDefault();
      const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
      const x = event.clientX;
      const y = event.clientY;
      
      if (x < rect.left || x > rect.right || y < rect.top || y > rect.bottom) {
        this.isDragOver = false;
      }
    },

    handleDrop(event: DragEvent, isDemo = false) {
      event.preventDefault();
      this.isDragOver = false;
      
      const files = event.dataTransfer?.files;
      if (files && files.length > 0) {
        const file = files[0];
        if (file.type.startsWith('image/')) {
          this.uploadPortfolioImage(file, isDemo);
        } else {
          this.portfolioUploadError = 'Please select an image file (JPG, PNG, etc.)';
        }
      }
    },

    handleFileSelect(event: Event, isDemo = false) {
      const target = event.target as HTMLInputElement;
      const file = target.files?.[0];
      if (file) {
        this.uploadPortfolioImage(file, isDemo);
      }
      // Reset file input
      target.value = '';
    },
  },
});
