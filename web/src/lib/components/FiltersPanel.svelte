<script lang="ts">
  import type { FilterStore } from '$lib/filters.svelte';
  import {
    SENIORITY_OPTIONS,
    CATEGORY_OPTIONS,
    WORK_MODE_OPTIONS,
    EMPLOYMENT_OPTIONS,
    COMPANY_SIZE_OPTIONS,
  } from '$lib/facets';
  import FacetSection from './facets/FacetSection.svelte';
  import CheckboxGroup from './facets/CheckboxGroup.svelte';
  import TokenInput from './facets/TokenInput.svelte';

  // The panel is pure presentation over the store: every control reads the
  // store's current value and routes changes back through its methods.
  let { store }: { store: FilterStore } = $props();

  // Category has many options; collapse to a short head until expanded.
  let showAllCategories = $state(false);
  const visibleCategories = $derived(showAllCategories ? CATEGORY_OPTIONS : CATEGORY_OPTIONS.slice(0, 6));

  function onSalaryInput(e: Event) {
    const raw = (e.currentTarget as HTMLInputElement).value;
    store.setSalaryMin(raw === '' ? null : Number(raw));
  }
</script>

<div class="flex flex-col gap-4">
  <div class="flex items-center justify-between">
    <h2 class="text-sm font-semibold tracking-tight">Filters</h2>
    {#if store.active > 0}
      <button type="button" class="text-xs text-muted-foreground hover:text-foreground" onclick={() => store.clear()}>
        Clear all
      </button>
    {/if}
  </div>

  <FacetSection title="Seniority">
    <CheckboxGroup options={SENIORITY_OPTIONS} selected={store.value.seniority} onToggle={(v) => store.toggle('seniority', v)} />
  </FacetSection>

  <FacetSection title="Category">
    <CheckboxGroup options={visibleCategories} selected={store.value.category} onToggle={(v) => store.toggle('category', v)} />
    <button
      type="button"
      class="mt-1.5 text-xs text-muted-foreground hover:text-foreground"
      onclick={() => (showAllCategories = !showAllCategories)}
    >
      {showAllCategories ? 'Show less' : `Show all (${CATEGORY_OPTIONS.length})`}
    </button>
  </FacetSection>

  <FacetSection title="Work mode">
    <CheckboxGroup options={WORK_MODE_OPTIONS} selected={store.value.workMode} onToggle={(v) => store.toggle('workMode', v)} />
  </FacetSection>

  <FacetSection title="Employment">
    <CheckboxGroup options={EMPLOYMENT_OPTIONS} selected={store.value.employmentType} onToggle={(v) => store.toggle('employmentType', v)} />
  </FacetSection>

  <FacetSection title="Company size">
    <CheckboxGroup options={COMPANY_SIZE_OPTIONS} selected={store.value.companySize} onToggle={(v) => store.toggle('companySize', v)} />
  </FacetSection>

  <FacetSection title="Min salary">
    <input
      type="number"
      inputmode="numeric"
      min="0"
      step="1000"
      placeholder="e.g. 80000"
      value={store.value.salaryMin ?? ''}
      oninput={onSalaryInput}
      class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    />
  </FacetSection>

  <FacetSection title="Visa sponsorship">
    <label class="flex cursor-pointer items-center gap-2 text-sm">
      <input type="checkbox" class="size-4 rounded border-border" checked={store.value.visa} onchange={(e) => store.setVisa(e.currentTarget.checked)} />
      <span>Offers visa sponsorship</span>
    </label>
  </FacetSection>

  <FacetSection title="Skills">
    <TokenInput
      tokens={store.value.skills}
      onAdd={(v) => store.add('skills', v)}
      onRemove={(v) => store.remove('skills', v)}
      placeholder="e.g. go, react"
    />
  </FacetSection>

  <FacetSection title="Countries">
    <TokenInput
      tokens={store.value.countries}
      onAdd={(v) => store.add('countries', v)}
      onRemove={(v) => store.remove('countries', v)}
      placeholder="ISO code, e.g. DE"
    />
  </FacetSection>
</div>
