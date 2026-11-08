// Admin API client (docs/contract.md §7). Session auth: cookies go with
// credentials: 'same-origin', and the browser adds Origin to unsafe requests,
// which Go checks for CSRF. A 401 sends the owner to Portiko's login.
import { ApiError } from '../lib/api';
import { loginURL } from '../lib/nav.svelte';

export async function adminRequest<T>(
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE',
  path: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  const res = await fetch(path, {
    method,
    credentials: 'same-origin',
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  });
  if (res.status === 401) {
    location.assign(loginURL(location.pathname + location.search));
    throw new ApiError(401, 'unauthenticated', 'Signing you in again…');
  }
  if (!res.ok) {
    let code = 'http_error';
    let message = `Request failed (${res.status})`;
    try {
      const b = await res.json();
      if (b?.error) {
        code = b.error.code ?? code;
        message = b.error.message ?? message;
      }
    } catch {
      /* not JSON */
    }
    if (res.status === 403 && code === 'http_error') message = 'You don’t have permission to do that.';
    throw new ApiError(res.status, code, message);
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export const errorText = (err: unknown) => (err instanceof Error ? err.message : 'Something went wrong.');

// --- Shapes (contract §6-§7). Go is the reference; fields beyond these are shown as-is. ---

export interface Citation {
  doi?: string;
  pmid?: string;
  url?: string;
  text?: string;
}

export interface Submission {
  id: number;
  reference: string;
  kind: string;
  status: string;
  target_slug?: string | null;
  species_name?: string | null;
  botanical_name?: string | null;
  hazard_type?: string | null;
  details?: string;
  citations?: Citation[];
  email?: string | null;
  ip_hash?: string | null;
  warnings?: unknown;
  created_at: string;
  updated_at?: string;
}

export interface SubmissionEvent {
  at: string;
  actor?: string;
  type?: string;
  action?: string;
  from_status?: string | null;
  to_status?: string | null;
  status?: string | null;
  note?: string | { text: string; visible_to_submitter?: boolean } | null;
  text?: string;
  visible_to_submitter?: boolean;
}

export interface SubmissionPage {
  submissions: Submission[];
  next_cursor: string | null;
}

export function parseSubmissionPage(body: any): SubmissionPage {
  return {
    submissions: body?.submissions ?? body?.items ?? (Array.isArray(body) ? body : []),
    next_cursor: body?.next_cursor ?? body?.cursor ?? null,
  };
}

export function parseSubmissionDetail(body: any): { submission: Submission; events: SubmissionEvent[] } {
  const submission = (body?.submission ?? body) as Submission;
  return { submission, events: body?.events ?? (submission as any)?.events ?? [] };
}

export interface Stats {
  open: number;
  by_status: Record<string, number>;
  by_kind: Record<string, number>;
  newest_at?: string | null;
  new_last_24h: number;
  spam_last_24h: number;
}

export const STATUSES = ['new', 'triaging', 'needs_info', 'accepted', 'rejected', 'exported', 'merged', 'spam'] as const;
