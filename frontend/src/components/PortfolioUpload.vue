<template>
  <div class="mb-6">
    <h3 class="text-lg font-semibold mb-2">{{ title }}</h3>
    <div
      class="border-2 border-dashed rounded-lg p-8 text-center transition-colors"
      :class="[
        isDragOver ? 'border-brand-500 bg-brand-50' : 'border-slate-300',
        portfolioUploading ? 'opacity-50 pointer-events-none' : ''
      ]"
      @dragover.prevent="handleDragOver"
      @dragenter.prevent="handleDragEnter"
      @dragleave.prevent="handleDragLeave"
      @drop.prevent="handleDrop"
    >
      <div v-if="portfolioUploading" class="flex items-center justify-center">
        <svg class="animate-spin h-8 w-8 text-brand-600 mr-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <span class="text-slate-600">{{ loadingText }}</span>
      </div>
      <div v-else>
        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-slate-400 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
        </svg>
        <p class="text-slate-600 mb-2">{{ uploadText }}</p>
        <input type="file" @change="handleFileSelect" accept="image/*" class="hidden" ref="fileInput" />
        <button @click="triggerFileInput" class="btn">
          Select Image
        </button>
        <p class="text-xs text-slate-500 mt-2">Supports JPG, PNG, and other image formats</p>
      </div>
    </div>
    <div v-if="portfolioUploadError" class="mt-2 text-danger text-sm">
      {{ portfolioUploadError }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const props = defineProps<{
  title: string;
  uploadText: string;
  loadingText: string;
  isDragOver: boolean;
  portfolioUploading: boolean;
  portfolioUploadError: string | null;
  isDemo?: boolean;
}>();

const emit = defineEmits<{
  dragOver: [event: DragEvent];
  dragEnter: [event: DragEvent];
  dragLeave: [event: DragEvent];
  drop: [event: DragEvent, isDemo: boolean];
  fileSelect: [event: Event, isDemo: boolean];
}>();

const fileInput = ref<HTMLInputElement | null>(null);

function handleDragOver(event: DragEvent) {
  emit('dragOver', event);
}

function handleDragEnter(event: DragEvent) {
  emit('dragEnter', event);
}

function handleDragLeave(event: DragEvent) {
  emit('dragLeave', event);
}

function handleDrop(event: DragEvent) {
  emit('drop', event, props.isDemo || false);
}

function handleFileSelect(event: Event) {
  emit('fileSelect', event, props.isDemo || false);
}

function triggerFileInput() {
  if (fileInput.value) {
    fileInput.value.click();
  }
}
</script>