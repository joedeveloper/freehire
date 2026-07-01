<script lang="ts">
  import SwipeDeck from '$lib/components/SwipeDeck.svelte';
</script>

<svelte:head>
  <title>Swipe jobs — freehire</title>
  <!-- Personal, interactive triage view: keep it out of search results. -->
  <meta name="robots" content="noindex" />
</svelte:head>

<div class="mx-auto w-full max-w-lg px-4 py-6">
  <SwipeDeck />
</div>
