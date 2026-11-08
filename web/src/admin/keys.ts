// API key helpers (docs/contract.md §7). The full key comes back once, from
// POST /api/v1/admin/api-keys, and must never be kept in the list or shown again.

export const SCOPES = [
  { id: 'submissions:read', label: 'Read submissions' },
  { id: 'submissions:write', label: 'Change submissions (status, notes)' },
  { id: 'stats:read', label: 'Read stats' },
] as const;

export type Scope = (typeof SCOPES)[number]['id'];

export interface ApiKey {
  name: string;
  prefix: string;
  scopes: string[];
  created_at: string;
  last_used_at?: string | null;
  expires_at?: string | null;
  revoked_at?: string | null;
}

/** Strip the secret from a create response so it can safely join the list. */
export function toListEntry(created: ApiKey & { key?: string }): ApiKey {
  const { key: _secret, ...rest } = created;
  return rest;
}

/** The list endpoint may answer `{ keys }`, `{ api_keys }` or a bare array. Secrets are dropped either way. */
export function parseKeyList(body: unknown): ApiKey[] {
  const raw = Array.isArray(body)
    ? body
    : ((body as { keys?: unknown[]; api_keys?: unknown[] } | null)?.keys ??
      (body as { api_keys?: unknown[] } | null)?.api_keys ??
      []);
  return (raw as (ApiKey & { key?: string })[]).map(toListEntry);
}

/** How a key is shown in the list: never the secret, only its public prefix. */
export function maskedKey(prefix: string): string {
  return `da_${prefix}_${'•'.repeat(8)}`;
}

export function keyState(k: ApiKey, now = Date.now()): 'active' | 'revoked' | 'expired' {
  if (k.revoked_at) return 'revoked';
  if (k.expires_at && Date.parse(k.expires_at) <= now) return 'expired';
  return 'active';
}

/**
 * Holds a newly created key for one showing. Once dismissed it is wiped and
 * nothing keeps a copy, so it can't be shown again.
 */
export class OneTimeSecret {
  #value: string | null = null;
  #prefix: string | null = null;

  reveal(secret: string, prefix: string): boolean {
    if (!secret) return false;
    this.#value = secret;
    this.#prefix = prefix;
    return true;
  }

  get value(): string | null {
    return this.#value;
  }

  get prefix(): string | null {
    return this.#prefix;
  }

  get visible(): boolean {
    return this.#value !== null;
  }

  dismiss(): void {
    this.#value = null;
    this.#prefix = null;
  }
}
