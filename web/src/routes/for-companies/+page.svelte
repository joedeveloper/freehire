<script lang="ts">
  import { page } from '$app/state';
  import ForCompaniesView from '$lib/components/ForCompaniesView.svelte';
  import Seo from '$lib/components/Seo.svelte';

  const canonical = $derived(`${page.url.origin}/for-companies`);
</script>

<Seo
  title="List your job board — freehire for companies"
  description="Get your company's whole ATS board indexed by freehire, the free, open-source IT job aggregator. Add a supported board with one line, or contribute an adapter — we crawl it regularly and close roles you take down."
  {canonical}
/>

<div class="mx-auto w-full max-w-6xl px-4 py-6">
  <ForCompaniesView />
</div>
