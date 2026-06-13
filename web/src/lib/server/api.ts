import { env } from '$env/dynamic/private';
import { createApi } from '$lib/api';

/** Build an API client for use inside a SvelteKit server `load` or endpoint.
 *  Pass the event's `fetch`, and the incoming request's `cookie` header for any
 *  authenticated call (e.g. `/me`, `/my/*`).
 *
 *  In production set `API_INTERNAL_URL` to the Go service origin (e.g.
 *  `http://app:8080`) so server-to-server calls reach the backend directly — a
 *  relative `/api` would hit this Node server, which has no such route. In dev
 *  it defaults to relative URLs, which the Vite proxy forwards to the backend.
 *
 *  With an absolute base URL, `event.fetch` does not forward the request's
 *  cookies, so the session cookie is forwarded explicitly when provided. Public
 *  reads can omit it. */
export function serverApi(fetchImpl: typeof fetch, cookie?: string | null) {
  return createApi(fetchImpl, env.API_INTERNAL_URL ?? '', cookie ? { cookie } : {});
}
