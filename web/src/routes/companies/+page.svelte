<script lang="ts">
  import CompaniesView from '$lib/components/CompaniesView.svelte';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();
</script>

<svelte:head>
  <!-- Full metadata/canonical/OG land in the SEO tasks (5.1). -->
  <title>Companies — freehire</title>
</svelte:head>

<div class="mx-auto w-full max-w-3xl px-4 py-6">
  <CompaniesView initial={data.initial} />
</div>
