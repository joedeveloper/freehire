import { describe, it, expect } from 'vitest';
import { FILTER_COLLECTIONS, collectionBySlug, collectionSlugs } from './collections';
import { FACETS } from './facets';

describe('collectionBySlug', () => {
  it('resolves a company-membership collection to a collections facet param', () => {
    const c = collectionBySlug('yc');
    expect(c).toBeDefined();
    expect(c?.title).toBe('Y Combinator');
    expect(c?.params).toEqual({ collections: 'yc' });
  });

  it('resolves a filter collection to its own facet params', () => {
    const c = collectionBySlug('remote-worldwide');
    expect(c).toBeDefined();
    expect(c?.title).toBe('Remote Worldwide');
    expect(c?.params).toEqual({ work_mode: 'remote', regions: 'global' });
  });

  it('returns undefined for an unknown slug', () => {
    expect(collectionBySlug('does-not-exist')).toBeUndefined();
  });
});

describe('collectionSlugs', () => {
  it('lists every collection slug across both registries with no duplicates', () => {
    const slugs = collectionSlugs();
    expect(slugs).toContain('yc'); // company collection
    expect(slugs).toContain('remote-worldwide'); // filter collection
    expect(new Set(slugs).size).toBe(slugs.length);
  });
});

describe('FILTER_COLLECTIONS invariants', () => {
  it('every filter collection has non-empty params', () => {
    // params is the single source of a card's count and the landing page's scoped
    // feed; an empty map would render the bare /jobs feed under a collection URL.
    for (const c of FILTER_COLLECTIONS) {
      expect(Object.keys(c.params).length, `filter collection "${c.slug}"`).toBeGreaterThan(0);
    }
  });

  it('every filter collection pins only known job-search facet params', () => {
    // A mistyped param key (e.g. `skill` for `skills`, `catgory` for `category`) is
    // silently ignored by the search, so the landing page would render an unfiltered
    // feed. Guard it against the same facet-param set the filter UI drives.
    const known = new Set(FACETS.map((f) => f.param));
    for (const c of FILTER_COLLECTIONS) {
      for (const key of Object.keys(c.params)) {
        expect(known.has(key), `filter collection "${c.slug}" param "${key}"`).toBe(true);
      }
    }
  });
});
