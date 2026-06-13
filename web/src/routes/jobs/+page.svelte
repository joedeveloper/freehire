<script lang="ts">
  import JobsView from '$lib/components/JobsView.svelte';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();
</script>

<svelte:head>
  <!-- Full metadata/canonical/OG land in the SEO tasks (5.1). -->
  <title>Jobs — freehire</title>
</svelte:head>

<div class="mx-auto w-full max-w-6xl px-4 py-6">
  <JobsView initial={data.initial} />
</div>
