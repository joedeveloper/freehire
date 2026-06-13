import { serverApi } from '$lib/server/api';
import type { PageServerLoad } from './$types';

const LIMIT = 20;

// Server-render the first page of search results for the current URL filters, so
// the job rows are in the initial HTML. The URL query already carries the search
// params (q, facets, sort) in the shape the search API reads; `searchJobs` adds
// pagination. Filters/sort/"load more" then run client-side over the same client.
export const load: PageServerLoad = async ({ url, fetch }) => {
  const initial = await serverApi(fetch).searchJobs(url.searchParams, LIMIT, 0);
  return { initial };
};
