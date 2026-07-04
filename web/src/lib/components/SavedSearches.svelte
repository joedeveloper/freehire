<script lang="ts">
  import { Bell, Pencil, Plus, Trash2 } from '@lucide/svelte';
  import { ApiError } from '$lib/api';
  import { isAuthenticated } from '$lib/auth.svelte';
  import { openAuthDialog } from '$lib/auth-dialog.svelte';
  import { canonicalQuery, filtersToParams } from '$lib/filters';
  import type { StagedFilters } from '$lib/stagedFilters.svelte';
  import { savedSearches } from '$lib/savedSearches.svelte';
  import { notifications } from '$lib/notifications.svelte';
  import type { SavedSearch } from '$lib/types';
  import { Button, Input } from '$lib/ui';

  // Focus the text input a save/rename row reveals, so the caret is ready without a
  // second click — the row only mounts inside the {#if}, so the attachment runs
  // exactly when it appears.
  const focusInput = (node: Element) => void (node.querySelector('input') as HTMLInputElement | null)?.focus();

  // The "My filters" control: a list of saved sets. Click a name to apply it (seeds the
  // filters), rename it inline, or delete it; a save bar on top saves the current
  // filters under a name (and offers "Update <name>" once they drift from an applied
  // set). It is the first tab of the filter modal and drives the modal's *staged* copy —
  // applying/saving act on the staged edits and reach the live list only on the modal's
  // Show-results action. Board sharing lives on the /my/searches account page, not here.
  // Signed-out users see a sign-in prompt instead.
  let { store }: { store: StagedFilters } = $props();

  // The set the user is working from (selected or just saved). Distinct from the
  // derived `activeId` so we can offer "Update <name>" after the filters are edited
  // away from it.
  let baseId = $state<number | null>(null);
  let naming = $state(false);
  let name = $state('');
  // The set whose name is being edited inline (null = none), and its working value.
  let renamingId = $state<number | null>(null);
  let renameValue = $state('');
  let busy = $state(false);
  let error = $state<string | null>(null);

  // The current filters as a canonical query string (filtersToParams is already
  // canonical), and the saved set — if any — that exactly matches it.
  const current = $derived(filtersToParams(store.value).toString());
  const items = $derived(savedSearches.items);
  const activeId = $derived(items.find((s) => canonicalQuery(s.query) === current)?.id ?? null);
  const base = $derived(items.find((s) => s.id === baseId) ?? null);
  // The user started from a saved set and then changed the filters.
  const dirty = $derived(base != null && canonicalQuery(base.query) !== current);

  // Notification state for the currently-selected saved set: whether Telegram is
  // configured + linked, and the subscription (if any) on the active set.
  const telegram = $derived(notifications.telegram);
  const activeSub = $derived(activeId != null ? notifications.forSavedSearch(activeId) : undefined);
  let notifyBusy = $state(false);
  let notifyError = $state<string | null>(null);
  // Set after opening the connect link, so the UI can offer a "I've connected" recheck.
  let connecting = $state(false);

  // Load the list once the session is confirmed (boot-time /me may still be in
  // flight); the store no-ops on repeat calls and off the browser. On sign-out,
  // drop the cache so a different user signing in on the same tab can't see the
  // previous user's saved searches (the store is a per-user module singleton).
  $effect(() => {
    if (isAuthenticated()) {
      void savedSearches.ensureLoaded();
      void notifications.ensureLoaded();
    } else {
      savedSearches.reset();
      notifications.reset();
    }
  });

  // Toggle Telegram notifications for the active saved set. If the user hasn't
  // linked Telegram yet, open the deep link instead of subscribing — they connect
  // the bot, then flip the toggle.
  async function toggleNotify() {
    if (activeId == null || notifyBusy) return;
    notifyBusy = true;
    notifyError = null;
    try {
      if (!telegram.linked) {
        await connectTelegram();
        return;
      }
      if (activeSub) {
        await notifications.unsubscribe(activeSub.id);
      } else {
        await notifications.subscribe(activeId);
      }
    } catch (e) {
      notifyError = e instanceof ApiError ? e.message : 'Could not update notifications. Please try again.';
    } finally {
      notifyBusy = false;
    }
  }

  // Open the one-time deep link in a new tab so the user can tap Start in Telegram.
  async function connectTelegram() {
    const url = await notifications.link();
    window.open(url, '_blank', 'noopener');
    connecting = true;
  }

  // After the user reports they tapped Start, re-read the link status.
  async function recheckLink() {
    notifyBusy = true;
    notifyError = null;
    try {
      await notifications.refreshTelegram();
      if (notifications.telegram.linked) connecting = false;
    } catch {
      notifyError = 'Could not check the connection. Please try again.';
    } finally {
      notifyBusy = false;
    }
  }

  // Apply a saved set: seed the staged filters from it and remember it as the base
  // (so later edits surface an "Update <name>").
  function apply(set: SavedSearch) {
    error = null;
    naming = false;
    renamingId = null;
    store.apply(set.query);
    baseId = set.id;
  }

  function startSave() {
    error = null;
    renamingId = null;
    name = '';
    naming = true;
  }

  function onNameKey(e: KeyboardEvent) {
    if (e.key === 'Enter') void save();
    else if (e.key === 'Escape') naming = false;
  }

  function startRename(set: SavedSearch) {
    error = null;
    naming = false;
    renamingId = set.id;
    renameValue = set.name;
  }

  function cancelRename() {
    renamingId = null;
  }

  function onRenameKey(e: KeyboardEvent) {
    if (e.key === 'Enter') void commitRename();
    else if (e.key === 'Escape') cancelRename();
  }

  async function commitRename() {
    const set = items.find((s) => s.id === renamingId);
    const trimmed = renameValue.trim();
    if (!set || !trimmed || busy) return;
    if (trimmed === set.name) {
      renamingId = null;
      return;
    }
    busy = true;
    error = null;
    try {
      await savedSearches.update(set.id, { name: trimmed });
      renamingId = null;
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not rename. Please try again.';
    } finally {
      busy = false;
    }
  }

  async function save() {
    const trimmed = name.trim();
    if (!trimmed || busy) return;
    busy = true;
    error = null;
    try {
      const set = await savedSearches.create(trimmed, current);
      baseId = set.id;
      naming = false;
      name = '';
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not save. Please try again.';
    } finally {
      busy = false;
    }
  }

  async function update() {
    if (!base || busy) return;
    busy = true;
    error = null;
    try {
      await savedSearches.update(base.id, { query: current });
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not update. Please try again.';
    } finally {
      busy = false;
    }
  }

  async function remove(set: SavedSearch) {
    if (busy || !window.confirm(`Delete the saved filter “${set.name}”?`)) return;
    busy = true;
    error = null;
    try {
      await savedSearches.remove(set.id);
      if (baseId === set.id) baseId = null;
      if (renamingId === set.id) renamingId = null;
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Could not delete. Please try again.';
    } finally {
      busy = false;
    }
  }
</script>

<div class="flex flex-col gap-2">
  <h3 class="text-sm font-semibold tracking-tight">My filters</h3>

  {#if !isAuthenticated()}
    <button
      type="button"
      class="self-start text-sm font-medium text-primary underline-offset-4 hover:underline"
      onclick={() => openAuthDialog()}
    >
      Sign in to save filters
    </button>
  {:else}
    <!-- Save bar: the primary action. It becomes an Update/Save-as-new pair once the
         filters drift from an applied set, and an inline name input while naming. -->
    {#if naming}
      <div class="flex items-center gap-2" {@attach focusInput}>
        <Input bind:value={name} placeholder="Filter name" maxlength={100} class="min-w-0 flex-1" onkeydown={onNameKey} />
        <Button variant="primary" size="sm" onclick={save} disabled={!name.trim() || busy}>
          {busy ? 'Saving…' : 'Save'}
        </Button>
        <Button variant="ghost" size="sm" onclick={() => (naming = false)}>Cancel</Button>
      </div>
    {:else if dirty}
      <div class="flex flex-wrap items-center gap-2">
        <Button variant="primary" size="sm" onclick={update} disabled={busy}>Update “{base?.name}”</Button>
        <Button variant="secondary" size="sm" onclick={startSave} disabled={busy}>Save as new</Button>
      </div>
    {:else}
      <Button variant="secondary" size="sm" onclick={startSave} disabled={busy} class="w-full justify-center gap-1.5">
        <Plus class="size-4" aria-hidden="true" />
        Save current filters
      </Button>
    {/if}

    <!-- The saved sets as a list: click a name to apply, pencil to rename inline,
         trash to delete. The set matching the current filters is dotted + highlighted;
         a set you applied and then edited keeps a muted dot as the "base". -->
    {#if items.length > 0}
      <ul class="mt-1 flex flex-col gap-0.5">
        {#each items as set (set.id)}
          {#if renamingId === set.id}
            <li class="flex items-center gap-2 py-0.5" {@attach focusInput}>
              <Input bind:value={renameValue} maxlength={100} class="min-w-0 flex-1" onkeydown={onRenameKey} />
              <Button variant="primary" size="sm" onclick={commitRename} disabled={!renameValue.trim() || busy}>Save</Button>
              <Button variant="ghost" size="sm" onclick={cancelRename}>Cancel</Button>
            </li>
          {:else}
            {@const active = set.id === activeId}
            {@const isBase = dirty && set.id === baseId}
            <li class="group flex items-center gap-0.5">
              <button
                type="button"
                onclick={() => apply(set)}
                title="Apply this filter"
                class={['flex min-w-0 flex-1 items-center gap-2 rounded-lg px-2 py-2 text-left transition-colors hover:bg-accent', active && 'bg-accent']}
              >
                <span class={['size-1.5 shrink-0 rounded-full transition-colors', active ? 'bg-primary' : isBase ? 'bg-muted-foreground/50' : 'bg-transparent']}></span>
                <span class={['truncate text-sm', active && 'font-medium']}>{set.name}</span>
              </button>
              <button
                type="button"
                aria-label="Rename “{set.name}”"
                title="Rename"
                onclick={() => startRename(set)}
                class="flex size-8 shrink-0 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              >
                <Pencil class="size-4" />
              </button>
              <button
                type="button"
                aria-label="Delete “{set.name}”"
                title="Delete"
                onclick={() => remove(set)}
                class="flex size-8 shrink-0 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
              >
                <Trash2 class="size-4" />
              </button>
            </li>
          {/if}
        {/each}
      </ul>
    {/if}

    {#if error}
      <p class="text-sm text-destructive">{error}</p>
    {/if}

    <!-- Telegram notifications. The toggle needs a concrete saved set to target;
         when none is active, a one-line hint advertises the feature so it is
         discoverable from the filters panel (not only after selecting a set). -->
    {#if telegram.enabled && !naming && activeId == null}
      <p class="flex items-center gap-1.5 border-t border-border pt-2 text-xs text-muted-foreground">
        <Bell class="size-3.5" aria-hidden="true" />
        Save a filter to get its new jobs in Telegram.
      </p>
    {/if}
    {#if telegram.enabled && activeId != null && !naming}
      <div class="flex flex-col gap-1 border-t border-border pt-2">
        {#if telegram.linked}
          <label class="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              class="size-4 rounded border-input"
              checked={!!activeSub}
              disabled={notifyBusy}
              onchange={toggleNotify}
            />
            <Bell class="size-4" aria-hidden="true" />
            <span>Notify me on Telegram</span>
          </label>
        {:else if connecting}
          <p class="text-sm text-muted-foreground">Opened Telegram — tap “Start”, then:</p>
          <Button variant="secondary" size="sm" onclick={recheckLink} disabled={notifyBusy}>
            {notifyBusy ? 'Checking…' : 'I’ve connected'}
          </Button>
        {:else}
          <Button variant="secondary" size="sm" onclick={connectTelegram} disabled={notifyBusy}>
            <Bell class="mr-1 size-4" aria-hidden="true" />
            Connect Telegram for alerts
          </Button>
        {/if}
        {#if notifyError}
          <p class="text-sm text-destructive">{notifyError}</p>
        {/if}
      </div>
    {/if}
  {/if}
</div>
