import { describe, expect, it } from 'vitest';
import { OneTimeSecret, keyState, maskedKey, parseKeyList, toListEntry } from './keys';

const created = {
  name: 'ci',
  prefix: 'Ab12Cd34',
  scopes: ['stats:read'],
  created_at: '2026-11-01T00:00:00Z',
  key: 'da_Ab12Cd34_secretsecretsecret',
};

describe('key display once', () => {
  it('shows a new key until dismissed, then never again', () => {
    const s = new OneTimeSecret();
    expect(s.visible).toBe(false);
    expect(s.reveal(created.key, created.prefix)).toBe(true);
    expect(s.value).toBe(created.key);
    s.dismiss();
    expect(s.visible).toBe(false);
    expect(s.value).toBeNull();
    expect(s.prefix).toBeNull();
    expect(JSON.stringify(s)).not.toContain('secret');
  });

  it('refuses an empty secret', () => {
    expect(new OneTimeSecret().reveal('', 'x')).toBe(false);
  });

  it('never puts the secret in the key list', () => {
    const entry = toListEntry(created);
    expect(entry).not.toHaveProperty('key');
    expect(JSON.stringify(entry)).not.toContain('secret');
    const list = parseKeyList({ keys: [created] });
    expect(list[0]).not.toHaveProperty('key');
    expect(parseKeyList([created])[0]).not.toHaveProperty('key');
    expect(parseKeyList({ api_keys: [created] })).toHaveLength(1);
    expect(parseKeyList(null)).toEqual([]);
  });

  it('masks keys in the list with the public prefix only', () => {
    expect(maskedKey('Ab12Cd34')).toMatch(/^da_Ab12Cd34_•+$/);
  });
});

describe('keyState', () => {
  const now = Date.parse('2026-11-08T00:00:00Z');
  it('reports active, expired and revoked keys', () => {
    expect(keyState(created, now)).toBe('active');
    expect(keyState({ ...created, expires_at: '2026-11-01T00:00:00Z' }, now)).toBe('expired');
    expect(keyState({ ...created, revoked_at: '2026-11-02T00:00:00Z' }, now)).toBe('revoked');
  });
});
