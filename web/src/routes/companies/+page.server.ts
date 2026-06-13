import { serverApi } from '$lib/server/api';
import type { PageServerLoad } from './$types';

const LIMIT = 20;

// Server-render the first page of companies for the current ?q, so the rows are
// in the initial HTML. The debounced search and "load more" then run client-side.
export const load: PageServerLoad = async ({ url, fetch }) => {
  const q = url.searchParams.get('q') ?? '';
  const initial = await serverApi(fetch).listCompanies(q, LIMIT, 0);
  return { initial };
};
