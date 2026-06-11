// Job search filters: the model, its URL <-> state (de)serialization, and a
// reactive store that mirrors the filters into the URL query so they survive
// reloads, sharing, and back/forward. Param names match exactly what the search
// API (GET /api/v1/jobs/search) expects.

import { router } from './router.svelte';

export interface JobFilters {
  q: string;
  seniority: string[];
  category: string[];
  workMode: string[];
  employmentType: string[];
  companySize: string[];
  visa: boolean;
  salaryMin: number | null;
  skills: string[];
  countries: string[];
}

/** Multi-value (array) facet fields, used to type the toggle/add/remove helpers. */
export type MultiField = 'seniority' | 'category' | 'workMode' | 'employmentType' | 'companySize' | 'skills' | 'countries';

// Each array field maps to a query-param name the backend understands.
const PARAM: Record<MultiField, string> = {
  seniority: 'seniority',
  category: 'category',
  workMode: 'work_mode',
  employmentType: 'employment_type',
  companySize: 'company_size',
  skills: 'skills',
  countries: 'countries',
};

export function emptyFilters(): JobFilters {
  return {
    q: '',
    seniority: [],
    category: [],
    workMode: [],
    employmentType: [],
    companySize: [],
    visa: false,
    salaryMin: null,
    skills: [],
    countries: [],
  };
}

/** Serialize filters to URL query params (the same shape the search API reads). */
export function filtersToParams(f: JobFilters): URLSearchParams {
  const p = new URLSearchParams();
  if (f.q) p.set('q', f.q);
  for (const field of Object.keys(PARAM) as MultiField[]) {
    for (const v of f[field]) p.append(PARAM[field], v);
  }
  if (f.visa) p.set('visa_sponsorship', 'true');
  if (f.salaryMin != null) p.set('salary_min', String(f.salaryMin));
  return p;
}

/** Parse filters back from URL query params. Unknown/invalid values are dropped. */
export function filtersFromParams(p: URLSearchParams): JobFilters {
  const salary = Number(p.get('salary_min'));
  return {
    q: p.get('q') ?? '',
    seniority: p.getAll('seniority'),
    category: p.getAll('category'),
    workMode: p.getAll('work_mode'),
    employmentType: p.getAll('employment_type'),
    companySize: p.getAll('company_size'),
    visa: p.get('visa_sponsorship') === 'true',
    salaryMin: p.get('salary_min') && !Number.isNaN(salary) ? salary : null,
    skills: p.getAll('skills'),
    countries: p.getAll('countries'),
  };
}

/** Number of active facet constraints (the free-text q is not counted) — drives
 *  the mobile "Filters" badge. */
export function activeFilterCount(f: JobFilters): number {
  const fields: MultiField[] = ['seniority', 'category', 'workMode', 'employmentType', 'companySize', 'skills', 'countries'];
  const arrays = fields.reduce((n, field) => n + f[field].length, 0);
  return arrays + (f.visa ? 1 : 0) + (f.salaryMin != null ? 1 : 0);
}

/** Reactive filter state mirrored into the URL. Owned by the jobs view; mutations
 *  go through its methods so every change updates both the state and the URL. */
export class FilterStore {
  value = $state<JobFilters>(emptyFilters());

  constructor() {
    this.value = filtersFromParams(router.query);
  }

  get active(): number {
    return activeFilterCount(this.value);
  }

  #commit() {
    router.setQuery(filtersToParams(this.value));
  }

  setQuery(q: string) {
    this.value = { ...this.value, q };
    this.#commit();
  }

  setVisa(on: boolean) {
    this.value = { ...this.value, visa: on };
    this.#commit();
  }

  setSalaryMin(n: number | null) {
    this.value = { ...this.value, salaryMin: n };
    this.#commit();
  }

  /** Add the value to a facet if absent, remove it if present (checkbox groups). */
  toggle(field: MultiField, v: string) {
    const has = this.value[field].includes(v);
    this.value = { ...this.value, [field]: has ? this.value[field].filter((x) => x !== v) : [...this.value[field], v] };
    this.#commit();
  }

  /** Add a token to a facet (token inputs); no-op on blank or duplicate. */
  add(field: MultiField, raw: string) {
    const v = raw.trim();
    if (!v || this.value[field].includes(v)) return;
    this.value = { ...this.value, [field]: [...this.value[field], v] };
    this.#commit();
  }

  remove(field: MultiField, v: string) {
    this.value = { ...this.value, [field]: this.value[field].filter((x) => x !== v) };
    this.#commit();
  }

  clear() {
    this.value = emptyFilters();
    this.#commit();
  }

  /** Re-read filters from the URL (browser back/forward). No-op when already in
   *  sync, which also breaks the write-back loop after our own setQuery. */
  syncFromUrl() {
    if (router.query.toString() === filtersToParams(this.value).toString()) return;
    this.value = filtersFromParams(router.query);
  }
}
