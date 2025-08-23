import { defineConfig, loadEnv } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  // Prefer docker-compose provided env for API base, fallback to localhost:8080
  const apiTarget = env.VITE_API_BASE || 'http://localhost:8080';

  return {
    plugins: [vue()],
    server: {
      host: true,
      port: Number(env.FRONTEND_PORT || 5173),
      proxy: {
        '/api': {
          target: apiTarget,
          changeOrigin: true
        }
      }
    },
    preview: {
      port: 3000
    }
  };
});
