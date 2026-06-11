<script lang="ts">
  // A free-text token (chip) input for open-vocabulary facets like skills and
  // countries. Enter adds the draft; Backspace on an empty field removes the last
  // chip; the × removes a specific one. Stateless except for the in-progress draft.
  let {
    tokens,
    onAdd,
    onRemove,
    placeholder,
  }: { tokens: string[]; onAdd: (value: string) => void; onRemove: (value: string) => void; placeholder?: string } = $props();

  let draft = $state('');

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && draft.trim()) {
      e.preventDefault();
      onAdd(draft);
      draft = '';
    } else if (e.key === 'Backspace' && draft === '') {
      const last = tokens.at(-1);
      if (last !== undefined) onRemove(last);
    }
  }
</script>

<div class="flex flex-wrap items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1.5 focus-within:ring-2 focus-within:ring-ring">
  {#each tokens as token (token)}
    <span class="inline-flex items-center gap-1 rounded bg-accent px-1.5 py-0.5 text-xs">
      {token}
      <button type="button" class="text-muted-foreground hover:text-foreground" onclick={() => onRemove(token)} aria-label={`Remove ${token}`}>×</button>
    </span>
  {/each}
  <input
    bind:value={draft}
    onkeydown={onKeydown}
    {placeholder}
    class="min-w-[6rem] flex-1 bg-transparent text-sm outline-none"
  />
</div>
