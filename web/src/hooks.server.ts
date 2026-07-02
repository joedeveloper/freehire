import { env } from '$env/dynamic/public';
import { sequence } from '@sveltejs/kit/hooks';
import * as Sentry from '@sentry/sveltekit';

// Same opt-in, env-gated init as the client: no PUBLIC_SENTRY_DSN ⇒ no init, and
// SSR runs unchanged. Errors-only, PII off. The server reports to the same
// (frontend) Sentry project as the browser.
if (env.PUBLIC_SENTRY_DSN) {
  Sentry.init({
    dsn: env.PUBLIC_SENTRY_DSN,
    environment: env.PUBLIC_SENTRY_ENVIRONMENT || 'development',
    tracesSampleRate: 0,
    sendDefaultPii: false,
  });
}

// sentryHandle scopes each SSR request; it is a passthrough when init was skipped.
export const handle = sequence(Sentry.sentryHandle());

// Reports uncaught SSR errors to Sentry; inert when init was skipped above.
export const handleError = Sentry.handleErrorWithSentry();
