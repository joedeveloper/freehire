<script lang="ts">
  import type { FilterStore } from '$lib/filters.svelte';
  import { FACETS } from '$lib/facets';
  import FacetSection from './facets/FacetSection.svelte';

  // The panel is pure presentation over the store: it iterates the facet
  // registry and renders each section, plus the two special controls (visa,
  // min salary) that aren't multi-value facets.
  let { store }: { store: FilterStore } = $props();

  function onSalaryInput(e: Event) {
    const raw = (e.currentTarget as HTMLInputElement).value;
    store.setSalaryMin(raw === '' ? null : Number(raw));
  }
</script>

<div class="flex flex-col gap-4">
  <div class="flex items-center justify-between">
    <h2 class="text-base font-semibold tracking-tight">Filters</h2>
    {#if store.active > 0}
      <button type="button" class="text-xs text-muted-foreground transition-colors hover:text-foreground" onclick={() => store.clear()}>
        Reset all
      </button>
    {/if}
  </div>

  {#each FACETS as def (def.param)}
    <FacetSection {def} {store} />
  {/each}

  <div class="border-b border-border pb-4">
    <h3 class="mb-2 text-sm font-semibold tracking-tight">Visa</h3>
    <label class="flex cursor-pointer items-center gap-2 text-sm">
      <input
        type="checkbox"
        class="size-4 rounded border-border"
        checked={store.value.visa}
        onchange={(e) => store.setVisa(e.currentTarget.checked)}
      />
      <span>Visa sponsorship</span>
    </label>
  </div>

  <div>
    <h3 class="mb-2 text-sm font-semibold tracking-tight">Min salary</h3>
    <input
      type="number"
      inputmode="numeric"
      min="0"
      step="1000"
      placeholder="0"
      value={store.value.salaryMin ?? ''}
      oninput={onSalaryInput}
      class="h-9 w-full rounded-lg border border-input bg-transparent px-3 text-sm transition-colors placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 dark:bg-input/30"
    />
  </div>
</div>
