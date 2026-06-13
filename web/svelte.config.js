import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
export default {
  preprocess: vitePreprocess(),
  kit: {
    // The frontend ships as a long-lived Node server (see design D1/D2): nginx
    // fronts it and proxies /api + /health to the Go backend, keeping the SPA
    // and API same-origin for the SameSite=Lax auth cookie.
    adapter: adapter(),
  },
};
