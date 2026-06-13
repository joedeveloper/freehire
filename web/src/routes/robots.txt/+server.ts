import type { RequestHandler } from './$types';

// A real robots file (not the SPA shell): allow crawling the public pages, keep
// personal pages out, and point at the sitemap.
export const GET: RequestHandler = ({ url }) => {
  const body = `User-agent: *
Allow: /
Disallow: /my/

Sitemap: ${url.origin}/sitemap.xml
`;
  return new Response(body, {
    headers: {
      'content-type': 'text/plain; charset=utf-8',
      'cache-control': 'public, max-age=86400',
    },
  });
};
