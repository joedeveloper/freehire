<script lang="ts">
  import { onMount } from 'svelte';
  import { listJobs } from '$lib/api';
  import { Paginator } from '$lib/paginated.svelte';
  import States from './States.svelte';
  import JobRow from './JobRow.svelte';
  import LoadMore from './LoadMore.svelte';

  const jobs = new Paginator(listJobs);
  onMount(() => jobs.start());
</script>

{#if jobs.status === 'loading'}
  <States state="loading" />
{:else if jobs.status === 'error'}
  <States state="error" message="Failed to load jobs." />
{:else if jobs.items.length === 0}
  <States state="empty" message="No jobs yet." />
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
