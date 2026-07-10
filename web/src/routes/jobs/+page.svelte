<script lang="ts">
  import { page } from '$app/state';
  import JobsView from '$lib/components/JobsView.svelte';
  import Seo from '$lib/components/Seo.svelte';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  const canonical = $derived(`${page.url.origin}/jobs`);
</script>

<Seo
  title="Jobs · freehire"
  description="Browse tech jobs aggregated from company career boards — filter by stack, seniority, location and remote policy."
  {canonical}
/>

<div class="mx-auto w-full max-w-6xl px-4 py-6">
  <!-- Primary heading kept for search engines and assistive tech, but visually
       hidden: the page's purpose is clear from the feed and onboarding banner, so
       the on-screen title was redundant. sr-only takes no layout space. -->
  <h1 class="sr-only">Tech jobs</h1>
  <JobsView initial={data.initial} />
</div>
