// Tiny client router over the History API. Owns the URL <-> route mapping; views
// read `router.route` and never parse the URL themselves. `navigate()` pushes a
// new path; same-origin anchor clicks and back/forward are wired in App.svelte.

export type Route =
  | { name: 'jobs' }
  | { name: 'job'; slug: string }
  | { name: 'companies' }
  | { name: 'company'; slug: string }
  | { name: 'notfound' };

function parse(path: string): Route {
  if (path === '/' || path === '') return { name: 'jobs' };
  if (path === '/companies') return { name: 'companies' };

  const jobSlug = path.match(/^\/jobs\/(.+)$/)?.[1];
  if (jobSlug) return { name: 'job', slug: decodeURIComponent(jobSlug) };

  const companySlug = path.match(/^\/companies\/(.+)$/)?.[1];
  if (companySlug) return { name: 'company', slug: decodeURIComponent(companySlug) };

  return { name: 'notfound' };
}

class Router {
  path = $state(window.location.pathname);
  // Parsed once per path change and shared across consumers, rather than
  // re-running the regexes on every getter read.
  route = $derived<Route>(parse(this.path));

  /** Push a new path and re-render. No-op if already there. */
  navigate(to: string) {
    if (to === this.path) return;
    window.history.pushState({}, '', to);
    this.path = to;
  }

  /** Sync to the current URL after browser back/forward. */
  syncFromLocation() {
    this.path = window.location.pathname;
  }
}

export const router = new Router();
