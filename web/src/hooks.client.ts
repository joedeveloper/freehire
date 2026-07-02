import { env } from '$env/dynamic/public';
import * as Sentry from '@sentry/sveltekit';

// Error tracking is opt-in and env-gated: without PUBLIC_SENTRY_DSN Sentry stays
// uninitialized and the app runs unchanged (mirrors the backend integration).
// Errors-only — no performance tracing, no session replay; sendDefaultPii is off
// so URLs/inputs/headers are not shipped.
if (env.PUBLIC_SENTRY_DSN) {
  Sentry.init({
    dsn: env.PUBLIC_SENTRY_DSN,
    environment: env.PUBLIC_SENTRY_ENVIRONMENT || 'development',
    tracesSampleRate: 0,
    sendDefaultPii: false,
  });
}

// Reports uncaught client-side errors to Sentry; inert when init was skipped above.
export const handleError = Sentry.handleErrorWithSentry();
