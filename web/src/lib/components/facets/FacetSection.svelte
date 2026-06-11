<script lang="ts">
  import type { Snippet } from 'svelte';

  // A titled block in the filters panel. Keeps section chrome (label, spacing,
  // divider) in one place so every facet looks the same.
  let { title, children }: { title: string; children: Snippet } = $props();
</script>

<div class="border-b border-border pb-4">
  <h3 class="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">{title}</h3>
  {@render children()}
</div>
