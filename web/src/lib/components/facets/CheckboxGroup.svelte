<script lang="ts">
  import type { FacetOption } from '$lib/facets';

  // A vertical list of checkboxes for one enum facet. Stateless: the selected
  // set and the toggle handler come from the owning store via props.
  let {
    options,
    selected,
    onToggle,
  }: { options: FacetOption[]; selected: string[]; onToggle: (value: string) => void } = $props();
</script>

<div class="flex flex-col gap-1.5">
  {#each options as opt (opt.value)}
    <label class="flex cursor-pointer items-center gap-2 text-sm">
      <input
        type="checkbox"
        class="size-4 rounded border-border"
        checked={selected.includes(opt.value)}
        onchange={() => onToggle(opt.value)}
      />
      <span>{opt.label}</span>
    </label>
  {/each}
</div>
