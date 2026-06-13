<script lang="ts">
  import ApiKeysView from '$lib/components/ApiKeysView.svelte';
</script>

<svelte:head>
  <title>API keys — freehire</title>
  <!-- Personal page: keep it out of search results. -->
  <meta name="robots" content="noindex" />
</svelte:head>

<div class="mx-auto w-full max-w-3xl px-4 py-6">
  <ApiKeysView />
</div>
