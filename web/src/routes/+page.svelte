<script lang="ts">
  import { page } from '$app/state';
  import HomeView from '$lib/components/HomeView.svelte';
  import Seo from '$lib/components/Seo.svelte';

  const canonical = $derived(`${page.url.origin}/`);
</script>

<Seo
  title="freehire — every tech job, in one clean feed"
  description="FreeHire aggregates tech jobs straight from company career boards, deduplicates them, and tags each with stack, seniority and location. Free and open source."
  {canonical}
/>

<div class="mx-auto w-full max-w-6xl px-4 py-6">
  <HomeView />
</div>
