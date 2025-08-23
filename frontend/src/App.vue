<template>
  <div class="min-h-screen bg-slate-50 text-slate-800">
    <header class="sticky top-0 z-40 w-full border-b bg-white/90 backdrop-blur-sm">
      <div class="container-app flex h-16 items-center">
        <RouterLink to="/" class="text-xl sm:text-2xl font-bold tracking-tight text-slate-900 mr-8">
          Stock Insights
        </RouterLink>
        <nav class="flex items-center space-x-6 text-sm font-medium">
          <RouterLink to="/" class="transition-colors hover:text-slate-900 text-slate-600">
            Home
          </RouterLink>
        </nav>
      </div>
    </header>

    <main class="container-app py-8">
      <RecommendationBanner />
      <div class="mt-6">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router';
import RecommendationBanner from './components/RecommendationBanner.vue';
</script>
