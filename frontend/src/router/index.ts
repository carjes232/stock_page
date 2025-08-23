import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';
import StockList from '../pages/StockList.vue';
import StockDetail from '../pages/StockDetail.vue';

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: StockList },
  { path: '/stock/:ticker', name: 'stock-detail', component: StockDetail, props: true }
];

const router = createRouter({
  history: createWebHistory(),
  routes
});

export default router;
