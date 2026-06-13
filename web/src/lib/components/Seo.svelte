<script lang="ts">
  // Per-page document metadata: title, description, canonical, and Open
  // Graph / Twitter Card tags. One <Seo> per route; rendered server-side so
  // crawlers and link-preview bots get it in the initial HTML.
  let {
    title,
    description,
    canonical,
    ogType = 'website',
  }: {
    title: string;
    description?: string;
    canonical?: string;
    ogType?: string;
  } = $props();
</script>

<svelte:head>
  <title>{title}</title>
  {#if description}
    <meta name="description" content={description} />
  {/if}
  {#if canonical}
    <link rel="canonical" href={canonical} />
    <meta property="og:url" content={canonical} />
  {/if}
  <meta property="og:title" content={title} />
  {#if description}
    <meta property="og:description" content={description} />
  {/if}
  <meta property="og:type" content={ogType} />
  <meta property="og:site_name" content="freehire" />
  <meta name="twitter:card" content="summary" />
  <meta name="twitter:title" content={title} />
  {#if description}
    <meta name="twitter:description" content={description} />
  {/if}
</svelte:head>
