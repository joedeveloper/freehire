import { defineConfig } from 'vite';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  // SvelteKit owns routing/SSR; it provides the $lib alias, so the manual alias
  // is gone. tailwindcss() must precede sveltekit().
  plugins: [tailwindcss(), sveltekit()],
  server: {
    port: 5173,
    strictPort: false,
    // Proxy the API so the browser (and server-side `load` via event.fetch) only
    // ever talks to this origin. That makes dev match the same-origin production
    // deployment, so the SameSite=Lax auth cookie is sent and no CORS is needed.
    // Target overridable via VITE_API_URL. /health is proxied too, mirroring the
    // prod nginx config (design D2) so dev and prod route identically.
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL ?? 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: process.env.VITE_API_URL ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
