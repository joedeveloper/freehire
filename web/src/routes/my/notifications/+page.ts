import { redirect } from '@sveltejs/kit';

// Notifications were merged into the saved-searches page (one tab); a subscription is
// always tied to a saved search, so alerts are managed there per-row.
export function load() {
  redirect(308, '/my/searches');
}
