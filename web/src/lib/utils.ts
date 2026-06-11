import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

/** Merge Tailwind class lists, resolving conflicts (last wins). */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** Format an RFC3339 timestamp as a short local date; '' for null/invalid. */
export function formatDate(ts: string | null): string {
  if (!ts) return '';
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

const TIME_UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ['year', 31536000],
  ['month', 2592000],
  ['week', 604800],
  ['day', 86400],
  ['hour', 3600],
  ['minute', 60],
  ['second', 1],
];

/** Format an RFC3339 timestamp as a relative "N ago" label (e.g. "13 seconds
 *  ago", "2 days ago"); '' for null/invalid. How recently a job was posted is a
 *  key signal, so the list card shows it relative rather than as a bare date. */
export function timeAgo(ts: string | null): string {
  if (!ts) return '';
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return '';
  const seconds = Math.round((Date.now() - d.getTime()) / 1000);
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
  for (const [unit, span] of TIME_UNITS) {
    if (Math.abs(seconds) >= span || unit === 'second') {
      return rtf.format(-Math.round(seconds / span), unit);
    }
  }
  return '';
}
