<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { resolve } from '$app/paths';
  import { api } from '$lib/api';
  import { cardTags, formatSalary } from '$lib/enrichment';
  import type { Job } from '$lib/types';
  import { Badge } from '$lib/ui';
  import CompanyLogo from './CompanyLogo.svelte';
  import States from './States.svelte';

  type Judgement = 'save' | 'dismiss';

  // Batch size and the "top up before it runs dry" threshold. COMMIT_PX is how
  // far a card must be dragged before the gesture counts as a decision.
  const LIMIT = 12;
  const PREFETCH_AT = 4;
  const COMMIT_PX = 100;

  // The deck honors the /jobs filters carried in the URL; unknown params (there
  // are none here beyond facets) are ignored by the endpoint. Captured once —
  // the filters are fixed for a swipe session.
  const facets = new URLSearchParams(page.url.search);

  // queue holds only UNJUDGED cards; the active one is queue[0]. seen dedups by
  // slug so a prefetch that races a just-sent save/dismiss can't re-add a card.
  let queue = $state<Job[]>([]);
  const seen = new Set<string>();
  // Single-step undo: the last judged card and how it was judged.
  let last = $state<{ job: Job; kind: Judgement } | null>(null);

  let loading = $state(false);
  let error = $state(false);
  let exhausted = $state(false);

  // Live drag offset of the active card (px); 0 when settled.
  let dragX = $state(0);
  let dragging = $state(false);
  let startX = 0;

  const current = $derived(queue[0] ?? null);
  const next = $derived(queue[1] ?? null);

  async function loadBatch() {
    if (loading || exhausted) return;
    loading = true;
    error = false;
    try {
      // offset = held-unjudged count: judged cards are already excluded
      // server-side, so this returns the page right after what we hold.
      const slice = await api.swipeDeck(facets, LIMIT, queue.length);
      const fresh = slice.items.filter((j) => !seen.has(j.public_slug));
      for (const j of fresh) seen.add(j.public_slug);
      queue = [...queue, ...fresh];
      // A page with no rows at all means the undecided set is drained.
      if (slice.items.length === 0) exhausted = true;
    } catch {
      error = true;
    } finally {
      loading = false;
    }
  }

  function prefetchIfLow() {
    if (queue.length <= PREFETCH_AT && !exhausted && !loading) void loadBatch();
  }

  // Judge the active card: drop it from the queue optimistically, remember it for
  // undo, and persist. A failed write restores the card at the front so the
  // decision isn't silently lost.
  function judge(kind: Judgement) {
    const job = queue[0];
    if (!job) return;
    dragX = 0;
    dragging = false;
    queue = queue.slice(1);
    last = { job, kind };
    const send = kind === 'save' ? api.saveJob(job.public_slug) : api.dismissJob(job.public_slug);
    send.catch(() => {
      queue = [job, ...queue];
      if (last?.job.public_slug === job.public_slug) last = null;
    });
    prefetchIfLow();
  }

  // Undo the last decision: clear its server mark and put the card back on top.
  function undo() {
    if (!last) return;
    const { job, kind } = last;
    last = null;
    queue = [job, ...queue];
    void (kind === 'save' ? api.unsaveJob(job.public_slug) : api.undismissJob(job.public_slug));
  }

  function openDetail() {
    if (!current) return;
    window.open(resolve('/jobs/[slug]', { slug: current.public_slug }), '_blank', 'noopener');
  }

  // --- Pointer drag on the active card (touch + mouse) ---------------------
  function onPointerDown(e: PointerEvent) {
    if (!current) return;
    dragging = true;
    startX = e.clientX;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  }
  function onPointerMove(e: PointerEvent) {
    if (dragging) dragX = e.clientX - startX;
  }
  function onPointerUp() {
    if (!dragging) return;
    if (dragX > COMMIT_PX) judge('save');
    else if (dragX < -COMMIT_PX) judge('dismiss');
    else {
      // Not far enough — spring back.
      dragX = 0;
      dragging = false;
    }
  }

  // --- Keyboard (desktop) --------------------------------------------------
  function onKeydown(e: KeyboardEvent) {
    const t = e.target as HTMLElement | null;
    if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA')) return;
    switch (e.key) {
      case 'ArrowRight':
        e.preventDefault();
        judge('save');
        break;
      case 'ArrowLeft':
        e.preventDefault();
        judge('dismiss');
        break;
      case 'ArrowUp':
        e.preventDefault();
        openDetail();
        break;
      case 'u':
      case 'U':
        undo();
        break;
    }
  }

  onMount(loadBatch);

  const salary = $derived(current?.enrichment ? formatSalary(current.enrichment) : null);
  const tags = $derived(current ? cardTags(current) : []);
  const skills = $derived(current?.skills?.slice(0, 6) ?? []);
</script>

<svelte:window onkeydown={onKeydown} />

<div class="flex flex-col items-center gap-4">
  <p class="text-sm text-muted-foreground">Swipe to triage — right to save, left to skip.</p>

  {#if current}
    <!-- Card stack: the next card peeks behind for depth. -->
    <div class="relative h-[26rem] w-full select-none">
      {#if next}
        <div
          class="absolute inset-0 scale-95 rounded-2xl border border-border bg-card opacity-60"
          aria-hidden="true"
        ></div>
      {/if}

      <div
        role="group"
        aria-label={`Job: ${current.title} at ${current.company || 'unknown company'}`}
        class="absolute inset-0 flex touch-none flex-col gap-3 rounded-2xl border border-border bg-card p-5 shadow-lg"
        class:transition-transform={!dragging}
        style={`transform: translateX(${dragX}px) rotate(${dragX / 24}deg);`}
        onpointerdown={onPointerDown}
        onpointermove={onPointerMove}
        onpointerup={onPointerUp}
        onpointercancel={onPointerUp}
      >
        <!-- Swipe-direction cues. -->
        {#if dragX > 20}
          <span class="absolute left-4 top-4 rounded-md border-2 border-emerald-500 px-2 py-0.5 text-sm font-bold text-emerald-500">SAVE</span>
        {:else if dragX < -20}
          <span class="absolute right-4 top-4 rounded-md border-2 border-rose-500 px-2 py-0.5 text-sm font-bold text-rose-500">SKIP</span>
        {/if}

        <div class="flex items-center gap-3">
          <CompanyLogo name={current.company} />
          <div class="min-w-0">
            <div class="truncate font-semibold">{current.company || 'Unknown company'}</div>
            {#if current.location}
              <div class="truncate text-sm text-muted-foreground">{current.location}</div>
            {/if}
          </div>
        </div>

        <h2 class="line-clamp-2 text-xl font-semibold tracking-tight">{current.title}</h2>

        {#if salary}
          <div class="text-lg font-bold tracking-tight text-emerald-600 dark:text-emerald-400">{salary}</div>
        {/if}

        <div class="flex flex-wrap gap-1.5">
          {#each tags as tag (tag)}
            <Badge variant="secondary">{tag}</Badge>
          {/each}
          {#each skills as skill (skill)}
            <Badge variant="secondary">{skill}</Badge>
          {/each}
        </div>

        {#if current.description}
          <p class="line-clamp-4 text-sm text-muted-foreground">{current.description}</p>
        {/if}

        <button
          type="button"
          class="mt-auto text-left text-sm font-medium text-foreground underline-offset-4 hover:underline"
          onclick={openDetail}
        >
          Open full details →
        </button>
      </div>
    </div>

    <!-- Action buttons (desktop + a tap alternative to swiping). -->
    <div class="flex items-center gap-5">
      <button
        type="button"
        aria-label="Skip (dismiss)"
        class="flex h-14 w-14 items-center justify-center rounded-full border border-border bg-card text-2xl text-rose-500 transition hover:bg-accent"
        onclick={() => judge('dismiss')}
      >
        ✗
      </button>
      <button
        type="button"
        aria-label="Undo last"
        class="flex h-10 w-10 items-center justify-center rounded-full border border-border bg-card text-base text-muted-foreground transition hover:bg-accent disabled:opacity-40"
        disabled={!last}
        onclick={undo}
      >
        ↩
      </button>
      <button
        type="button"
        aria-label="Save"
        class="flex h-14 w-14 items-center justify-center rounded-full border border-border bg-card text-2xl text-emerald-500 transition hover:bg-accent"
        onclick={() => judge('save')}
      >
        ♥
      </button>
    </div>

    <p class="text-xs text-muted-foreground">
      <kbd class="rounded border border-border px-1">←</kbd> skip ·
      <kbd class="rounded border border-border px-1">→</kbd> save ·
      <kbd class="rounded border border-border px-1">↑</kbd> open ·
      <kbd class="rounded border border-border px-1">U</kbd> undo
    </p>
  {:else if loading}
    <States state="loading" />
  {:else if error}
    <States state="error" message="Failed to load the deck." />
  {:else}
    <div class="flex flex-col items-center gap-3 py-16 text-center">
      <p class="text-lg font-semibold">You're all caught up</p>
      <p class="text-sm text-muted-foreground">No more jobs match these filters.</p>
      <a
        href={resolve('/jobs')}
        class="text-sm font-medium text-foreground underline underline-offset-4"
      >
        Back to all jobs →
      </a>
    </div>
  {/if}
</div>
