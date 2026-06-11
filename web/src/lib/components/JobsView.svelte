<script lang="ts">
  import { onMount } from 'svelte';
  import { searchJobs } from '$lib/api';
  import { Paginator } from '$lib/paginated.svelte';
  import { FilterStore, filtersToParams } from '$lib/filters.svelte';
  import { router } from '$lib/router.svelte';
  import FiltersPanel from './FiltersPanel.svelte';
  import States from './States.svelte';
  import JobRow from './JobRow.svelte';
  import LoadMore from './LoadMore.svelte';

  // Filters live in the URL; the whole page is now driven by the search endpoint
  // (an empty query with no facets just browses everything). Results share the
  // Job wire shape, so JobRow renders them unchanged.
  const filters = new FilterStore();

  const makePaginator = () =>
    new Paginator((limit, offset) => searchJobs(filtersToParams(filters.value), limit, offset));

  let jobs = $state(makePaginator());
  let drawerOpen = $state(false);

  onMount(() => jobs.start());

  // Browser back/forward changes the URL query — pull it back into the filters.
  $effect(() => {
    router.search; // track
    filters.syncFromUrl();
  });

  // Re-run the search when any filter changes, debounced. The first effect run is
  // the initial mount, already loaded by onMount, so skip it.
  let started = false;
  let timer: ReturnType<typeof setTimeout>;
  $effect(() => {
    filtersToParams(filters.value).toString(); // track every filter field
    if (!started) {
      started = true;
      return;
    }
    clearTimeout(timer);
    timer = setTimeout(() => {
      jobs = makePaginator();
      jobs.start();
    }, 300);
  });
</script>

<div class="flex gap-6">
  <aside class="hidden w-72 shrink-0 md:block">
    <div class="sticky top-6 max-h-[calc(100vh-5rem)] overflow-y-auto rounded-xl border border-border bg-card p-4">
      <FiltersPanel store={filters} />
    </div>
  </aside>

  <div class="min-w-0 flex-1">
    <div class="mb-4 flex items-center gap-2">
      <input
        type="search"
        value={filters.value.q}
        oninput={(e) => filters.setQuery(e.currentTarget.value)}
        placeholder="Search jobs…"
        aria-label="Search jobs"
        class="h-9 min-w-0 flex-1 rounded-lg border border-input bg-transparent px-3 text-sm transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 dark:bg-input/30"
      />
      <button
        type="button"
        class="h-9 shrink-0 rounded-lg border border-border bg-secondary px-3 text-sm font-medium text-secondary-foreground transition-colors hover:bg-accent md:hidden"
        onclick={() => (drawerOpen = true)}
      >
        Filters{#if filters.active > 0}&nbsp;({filters.active}){/if}
      </button>
    </div>

    {#if jobs.status === 'loading'}
      <States state="loading" />
    {:else if jobs.status === 'error'}
      <States state="error" message="Failed to load jobs." />
    {:else if jobs.items.length === 0}
      <States state="empty" message="No matching jobs." />
    {:else}
      <div class="flex flex-col gap-3">
        {#each jobs.items as job (job.public_slug)}
          <JobRow {job} />
        {/each}
      </div>

      {#if jobs.hasMore}
        <LoadMore loading={jobs.loadingMore} error={jobs.loadMoreError} onclick={() => jobs.loadMore()} />
      {/if}
    {/if}
  </div>
</div>

{#if drawerOpen}
  <div class="fixed inset-0 z-40 md:hidden">
    <button class="absolute inset-0 bg-black/40" aria-label="Close filters" onclick={() => (drawerOpen = false)}></button>
    <div class="absolute left-0 top-0 flex h-full w-80 max-w-[85%] flex-col overflow-y-auto bg-background p-4 shadow-xl">
      <div class="mb-3 flex items-center justify-between">
        <span class="text-sm font-semibold tracking-tight">Filters</span>
        <button type="button" class="text-sm text-muted-foreground hover:text-foreground" onclick={() => (drawerOpen = false)}>Done</button>
      </div>
      <FiltersPanel store={filters} />
    </div>
  </div>
{/if}
