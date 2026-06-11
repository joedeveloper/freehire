// Option lists for the enum facets in the filters panel. They mirror the
// controlled vocabularies the backend enrichment validates (internal/enrich),
// curated and ordered for display. The open-vocabulary facets (skills, countries)
// are free token inputs, not listed here.

export interface FacetOption {
  value: string;
  label: string;
}

export const SENIORITY_OPTIONS: FacetOption[] = [
  { value: 'intern', label: 'Intern' },
  { value: 'junior', label: 'Junior' },
  { value: 'middle', label: 'Middle' },
  { value: 'senior', label: 'Senior' },
  { value: 'lead', label: 'Lead' },
  { value: 'principal', label: 'Principal' },
  { value: 'c_level', label: 'C-level' },
];

export const WORK_MODE_OPTIONS: FacetOption[] = [
  { value: 'remote', label: 'Remote' },
  { value: 'hybrid', label: 'Hybrid' },
  { value: 'onsite', label: 'On-site' },
];

export const EMPLOYMENT_OPTIONS: FacetOption[] = [
  { value: 'full_time', label: 'Full-time' },
  { value: 'part_time', label: 'Part-time' },
  { value: 'contract', label: 'Contract' },
  { value: 'internship', label: 'Internship' },
];

export const COMPANY_SIZE_OPTIONS: FacetOption[] = [
  { value: '1-10', label: '1–10' },
  { value: '11-50', label: '11–50' },
  { value: '51-200', label: '51–200' },
  { value: '201-500', label: '201–500' },
  { value: '501-1000', label: '501–1000' },
  { value: '1000+', label: '1000+' },
];

export const CATEGORY_OPTIONS: FacetOption[] = [
  { value: 'backend', label: 'Backend' },
  { value: 'frontend', label: 'Frontend' },
  { value: 'fullstack', label: 'Fullstack' },
  { value: 'mobile', label: 'Mobile' },
  { value: 'devops', label: 'DevOps' },
  { value: 'data_engineering', label: 'Data Engineering' },
  { value: 'data_science', label: 'Data Science' },
  { value: 'ml_ai', label: 'ML / AI' },
  { value: 'qa', label: 'QA' },
  { value: 'security', label: 'Security' },
  { value: 'design', label: 'Design' },
  { value: 'product', label: 'Product' },
  { value: 'project_management', label: 'Project Management' },
  { value: 'management', label: 'Management' },
  { value: 'marketing', label: 'Marketing' },
  { value: 'sales', label: 'Sales' },
  { value: 'support', label: 'Support' },
  { value: 'other', label: 'Other' },
];
