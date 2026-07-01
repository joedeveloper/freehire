import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

// Swipe mode is personal: both actions (save and dismiss) are per-user, and the
// deck endpoint is authenticated. A signed-out visitor has no deck to show, so
// guard server-side and bounce home with ?auth=required — the TopBar pops the
// sign-in dialog (same pattern as /my/jobs). After signing in the visitor
// re-enters swipe mode from /jobs.
export const load: PageServerLoad = async ({ parent }) => {
  const { user } = await parent();
  if (!user) redirect(302, '/?auth=required');
};
