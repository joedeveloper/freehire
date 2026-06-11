<script lang="ts">
  import type { FacetOption } from '$lib/facets';
  import { cn } from '$lib/utils';

  // A wrap of toggle pills for one facet. Selected pills fill with the primary
  // color; in exclude mode they take the muted destructive treatment to signal
  // "filter these out". Stateless — selection and toggle come from the store.
  let {
    options,
    selected,
    exclude = false,
    onToggle,
  }: {
    options: FacetOption[];
    selected: string[];
    exclude?: boolean;
    onToggle: (value: string) => void;
  } = $props();
</script>

<div class="flex flex-wrap gap-2">
  {#each options as opt (opt.value)}
    {@const active = selected.includes(opt.value)}
    <button
      type="button"
      onclick={() => onToggle(opt.value)}
      class={cn(
        'rounded-full border px-3 py-1.5 text-sm font-medium transition-colors active:translate-y-px',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50',
        !active && 'border-transparent bg-secondary text-secondary-foreground hover:bg-accent',
        active && !exclude && 'border-transparent bg-primary text-primary-foreground',
        active && exclude && 'border-destructive/30 bg-destructive/15 text-destructive line-through',
      )}
    >
      {opt.label}
    </button>
  {/each}
</div>
