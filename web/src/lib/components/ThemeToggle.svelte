<script lang="ts">
  import { onMount } from 'svelte';
  import { Sun, Moon } from '@lucide/svelte';
  import { themeStore } from '$lib/theme.svelte';

  // The server can't know the persisted theme (it lives in localStorage), so
  // render the light icon until mounted to match SSR exactly, then reflect the
  // real state. The page colours themselves are already correct pre-paint via
  // the no-FOUC inline script in app.html.
  let mounted = $state(false);
  onMount(() => (mounted = true));
  const isDark = $derived(mounted && themeStore.isDark);
</script>

<button
  type="button"
  onclick={() => themeStore.toggle()}
  title={isDark ? 'Dark · click for light' : 'Light · click for dark'}
  aria-label="Toggle theme"
  class="inline-flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
>
  {#if isDark}
    <Moon class="size-4" />
  {:else}
    <Sun class="size-4" />
  {/if}
</button>
