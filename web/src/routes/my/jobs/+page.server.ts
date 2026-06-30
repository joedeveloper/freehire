import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

// /my/jobs is personal: a signed-out visitor has nothing to show, so guard it
// server-side rather than render an empty "sign in" state. The user is resolved
// once in the root layout load; reuse it via parent(). Auth is a layout-level
// dialog (no /login route), so we bounce home with ?auth=required and the TopBar
// pops the sign-in dialog (same pattern as the ?auth_error OAuth callback).
export const load: PageServerLoad = async ({ parent }) => {
  const { user } = await parent();
  if (!user) redirect(302, '/?auth=required');
};
