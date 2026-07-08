<script lang="ts">
  import { page } from '$app/state';
  import RecruitersView from '$lib/components/RecruitersView.svelte';
  import Seo from '$lib/components/Seo.svelte';

  const canonical = $derived(`${page.url.origin}/recruiters`);
</script>

<Seo
  title="Post a job — freehire for recruiters"
  description="Submit a tech job to freehire, the free, open-source IT job aggregator. Moderator-reviewed postings join one clean, searchable feed of developer roles."
  {canonical}
/>

<div class="mx-auto w-full max-w-6xl px-4 py-6">
  <RecruitersView />
</div>
