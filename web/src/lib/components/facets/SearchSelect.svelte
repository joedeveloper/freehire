<script lang="ts">
  import type { FacetOption } from '$lib/facets';
  import { cn } from '$lib/utils';

  // A searchable multi-select: a filter field over the options rendered as a
  // scrollable pill list. Selected options sort to the front. Used for the larger
  // enum facets (specialization, industry).
  let {
    options,
    selected,
    exclude = false,
    placeholder,
    onToggle,
  }: {
    options: FacetOption[];
    selected: string[];
    exclude?: boolean;
    placeholder?: string;
    onToggle: (value: string) => void;
  } = $props();

  let filter = $state('');

  const shown = $derived(
    options
      .filter((o) => o.label.toLowerCase().includes(filter.trim().toLowerCase()))
      .sort((a, b) => Number(selected.includes(b.value)) - Number(selected.includes(a.value))),
  );
</script>

<div class="flex flex-col gap-2">
  <input
    bind:value={filter}
    {placeholder}
    class="h-9 w-full rounded-lg border border-input bg-transparent px-3 text-sm transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 dark:bg-input/30"
  />
  <div class="flex max-h-44 flex-wrap gap-1.5 overflow-y-auto">
    {#each shown as opt (opt.value)}
      {@const active = selected.includes(opt.value)}
      <button
        type="button"
        onclick={() => onToggle(opt.value)}
        class={cn(
          'rounded-full border px-2.5 py-1 text-sm transition-colors active:translate-y-px',
          !active && 'border-transparent bg-secondary text-secondary-foreground hover:bg-accent',
          active && !exclude && 'border-transparent bg-primary text-primary-foreground',
          active && exclude && 'border-destructive/30 bg-destructive/15 text-destructive line-through',
        )}
      >
        {opt.label}
      </button>
    {/each}
    {#if shown.length === 0}
      <span class="px-1 py-1 text-xs text-muted-foreground">Nothing found</span>
    {/if}
  </div>
</div>
