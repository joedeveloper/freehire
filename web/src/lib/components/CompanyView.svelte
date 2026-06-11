<script lang="ts">
  import { getCompany } from '$lib/api';
  import type { Company, Job } from '$lib/types';
  import { Paginator } from '$lib/paginated.svelte';
  import States from './States.svelte';
  import JobRow from './JobRow.svelte';
  import LoadMore from './LoadMore.svelte';

  let { slug }: { slug: string } = $props();

  const LIMIT = 20;

  // The company rides along with the first page of jobs, so a per-slug Paginator
  // drives the jobs list and stashes the company as a side output. The endpoint
  // omits a total, so "more" is inferred from a full page.
  let company = $state.raw<Company | null>(null);
  let pager = $state.raw<Paginator<Job> | null>(null);

  $effect(() => {
    const current = slug;
    company = null;
    const p = new Paginator<Job>(async (limit, offset) => {
      const res = await getCompany(current, limit, offset);
      if (current === slug) company = res.company;
      return { items: res.jobs, hasMore: res.jobs.length === limit };
    }, LIMIT);
    pager = p;
    p.start();
  });
</script>

{#if !pager || pager.status === 'loading'}
  <States state="loading" rows={4} />
{:else if pager.status === 'error' || !company}
  <States state="error" message="Company not found." />
{:else}
  <h1 class="text-xl font-semibold tracking-tight">{company.name}</h1>

  <div class="mt-4">
    {#if pager.items.length === 0}
      <States state="empty" message="No jobs for this company yet." />
    {:else}
      <div class="flex flex-col gap-3">
        {#each pager.items as job (job.public_slug)}
          <JobRow {job} />
        {/each}
      </div>

      {#if pager.hasMore}
        <LoadMore
          loading={pager.loadingMore}
          error={pager.loadMoreError}
          onclick={() => pager?.loadMore()}
        />
      {/if}
    {/if}
  </div>
{/if}
