<script lang="ts">
  import { onMount } from 'svelte';
  import { listCompanies } from '$lib/api';
  import { Paginator } from '$lib/paginated.svelte';
  import { router } from '$lib/router.svelte';
  import { Badge } from '$lib/ui';
  import States from './States.svelte';
  import LoadMore from './LoadMore.svelte';
  import CompanyLogo from './CompanyLogo.svelte';

  // Search lives in the URL (?q=) so it survives reload, sharing, and
  // back/forward — the same pattern as the jobs list, scaled down to a single
  // field (no FilterStore, which models job-only facets).
  let q = $state(router.query.get('q') ?? '');

  const makePaginator = () => new Paginator((limit, offset) => listCompanies(q, limit, offset));

  let companies = $state(makePaginator());
  let started = false;
  let timer: ReturnType<typeof setTimeout>;

  onMount(() => {
    companies.start();
    // A debounce timer left running after unmount would start a fetch for a
    // component that no longer exists.
    return () => clearTimeout(timer);
  });

  // Browser back/forward changes the URL query — pull it back into q. Reading
  // router.query (a derived over router.search) is what tracks this effect. The
  // guard makes our own setQuery write-back (below) a no-op, breaking the loop.
  $effect(() => {
    const urlQ = router.query.get('q') ?? '';
    if (urlQ !== q) q = urlQ;
  });

  // q changed: mirror it to the URL and re-query, debounced. Building params
  // reads q first, so the effect tracks it even on the initial run, which we
  // then skip — it is already loaded by onMount.
  $effect(() => {
    const params = new URLSearchParams();
    if (q) params.set('q', q);
    if (!started) {
      started = true;
      return;
    }
    router.setQuery(params);
    clearTimeout(timer);
    timer = setTimeout(() => {
      companies = makePaginator();
      companies.start();
    }, 300);
  });
</script>

<div class="mb-4">
  <input
    type="search"
    value={q}
    oninput={(e) => (q = e.currentTarget.value)}
    placeholder="Search companies…"
    aria-label="Search companies"
    class="h-9 w-full rounded-lg border border-input bg-transparent px-3 text-sm transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 dark:bg-input/30"
  />
</div>

{#if companies.status === 'loading'}
  <States state="loading" />
{:else if companies.status === 'error'}
  <States state="error" message="Failed to load companies." />
{:else if companies.items.length === 0}
  <States state="empty" message={q ? 'No matching companies.' : 'No companies yet.'} />
{:else}
  <p class="mb-3 text-sm text-muted-foreground" aria-live="polite">
    {companies.total.toLocaleString()} {companies.total === 1 ? 'company' : 'companies'}
  </p>
  <div class="flex flex-col gap-3">
    {#each companies.items as company (company.slug)}
      <a
        href={`/companies/${company.slug}`}
        class="flex items-center justify-between rounded-lg border border-border px-4 py-3 transition-colors hover:bg-accent"
      >
        <span class="flex min-w-0 items-center gap-2.5">
          <CompanyLogo name={company.name} size="size-6" />
          <span class="truncate font-medium">{company.name}</span>
        </span>
        <Badge variant="outline">{company.job_count} jobs</Badge>
      </a>
    {/each}
  </div>

  {#if companies.hasMore}
    <LoadMore
      loading={companies.loadingMore}
      error={companies.loadMoreError}
      onclick={() => companies.loadMore()}
    />
  {/if}
{/if}
