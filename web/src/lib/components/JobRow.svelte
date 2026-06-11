<script lang="ts">
  import { MapPin } from '@lucide/svelte';
  import { workArrangement } from '$lib/enrichment';
  import type { Job } from '$lib/types';
  import { Badge } from '$lib/ui';
  import { formatDate } from '$lib/utils';

  // Single source of truth for how a job appears in any list (jobs list and
  // company detail). The whole row is a link to the job detail.
  let { job }: { job: Job } = $props();

  const posted = $derived(formatDate(job.posted_at));
  // Prefer the enriched work mode over the raw remote flag, so the card and the
  // detail page agree (see workArrangement).
  const arrangement = $derived(workArrangement(job));
</script>

<a
  href={`/jobs/${job.id}`}
  class="block rounded-lg border border-border px-4 py-3 transition-colors hover:bg-accent"
>
  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0">
      <p class="truncate font-medium">{job.title}</p>
      <p class="mt-0.5 truncate text-sm text-muted-foreground">
        {job.company || 'Unknown company'}
        {#if job.location}
          <span class="inline-flex items-center gap-1">
            · <MapPin class="size-3" />{job.location}
          </span>
        {/if}
      </p>
    </div>
    <div class="flex shrink-0 flex-col items-end gap-1">
      {#if arrangement}
        <Badge variant="secondary">{arrangement}</Badge>
      {/if}
      {#if posted}
        <span class="text-xs text-muted-foreground">{posted}</span>
      {/if}
    </div>
  </div>
  <div class="mt-2 flex items-center gap-2">
    <Badge variant="outline">{job.source}</Badge>
  </div>
</a>
