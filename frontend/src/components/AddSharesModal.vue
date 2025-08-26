<template>
  <div v-if="show" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click="$emit('close')">
    <div class="bg-white rounded-xl p-6 w-full max-w-md mx-4" @click.stop>
      <h3 class="text-lg font-semibold mb-4">{{ isSell ? 'Sell Shares' : 'Add More Shares' }}</h3>
      <div v-if="currentItem">
        <p class="text-sm text-slate-600 mb-4">
          {{ isSell ? 'Selling shares of' : 'Adding shares to' }} <strong>{{ currentItem.ticker }}</strong> at current price of 
          <strong>${{ currentItem.currentPrice?.toFixed(2) || 'Loading...' }}</strong>
        </p>
        <div class="mb-4">
          <label class="block text-sm font-medium text-slate-700 mb-1">Number of Shares</label>
          <input 
            v-model="shareAmount" 
            type="number" 
            step="any" 
            placeholder="e.g. 10" 
            class="input w-full"
            @keyup.enter="$emit('confirm')"
          />
        </div>
        <div v-if="shareAmount > 0 && currentItem.currentPrice" class="text-sm text-slate-600 mb-4">
          <template v-if="!isSell">
            <p>Investment: ${{ (shareAmount * currentItem.currentPrice).toFixed(2) }}</p>
            <p>New total shares: {{ (currentItem.shares + parseFloat(shareAmount.toString())).toFixed(4) }}</p>
            <p>New avg price: ${{ (((currentItem.shares * currentItem.avgPrice) + (shareAmount * currentItem.currentPrice)) / (currentItem.shares + parseFloat(shareAmount.toString()))).toFixed(2) }}</p>
          </template>
          <template v-else>
            <p>Proceeds: ${{ (shareAmount * currentItem.currentPrice).toFixed(2) }}</p>
            <p>New total shares: {{ Math.max(0, currentItem.shares - parseFloat(shareAmount.toString())).toFixed(4) }}</p>
            <p class="text-slate-500">Avg price remains ${{ currentItem.avgPrice.toFixed(2) }}</p>
          </template>
        </div>
        <div class="flex gap-3">
          <button @click="$emit('close')" class="btn btn-secondary flex-1">Cancel</button>
          <button @click="$emit('confirm')" class="btn flex-1" :disabled="!shareAmount || shareAmount <= 0">
            {{ isSell ? 'Sell Shares' : 'Add Shares' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { PortfolioItem } from '../stores/portfolio';

const props = defineProps<{
  show: boolean;
  currentItem: PortfolioItem | null;
  shareAmount: number;
  isSell?: boolean;
}>();

const emit = defineEmits<{
  close: [];
  confirm: [];
  'update:shareAmount': [value: number];
}>();

const shareAmount = computed({
  get: () => props.shareAmount,
  set: (value) => emit('update:shareAmount', value)
});

const isSell = computed(() => !!props.isSell);
</script>
