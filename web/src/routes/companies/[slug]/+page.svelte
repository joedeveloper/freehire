<script lang="ts">
  import { page } from '$app/state';
  import CompanyView from '$lib/components/CompanyView.svelte';
  import Seo from '$lib/components/Seo.svelte';
  import { jsonLdScript, organizationJsonLd } from '$lib/seo';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  const origin = $derived(page.url.origin);
  const canonical = $derived(`${origin}/companies/${data.slug}`);
  const description = $derived(`Open jobs at ${data.company.name}, aggregated by freehire.`);
  const jsonLd = $derived(jsonLdScript(organizationJsonLd(data.company, origin)));
</script>

<Seo title={`${data.company.name} · freehire`} {description} {canonical} />

<svelte:head>
  {@html jsonLd}
</svelte:head>

<div class="mx-auto w-full max-w-3xl px-4 py-6">
  <!-- Remount on slug change so the seeded paginator starts fresh per company. -->
  {#key data.slug}
    <CompanyView company={data.company} jobs={data.jobs} hasMore={data.hasMore} slug={data.slug} />
  {/key}
</div>
