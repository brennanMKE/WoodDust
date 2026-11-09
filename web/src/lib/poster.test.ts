import { describe, expect, it } from 'vitest';
import { cleanShop, plaqueFontSize, slugFile, SHOP_MAX } from './poster';

describe('cleanShop', () => {
  it('trims, collapses whitespace and caps at 40 characters like the server', () => {
    expect(cleanShop('  Brennan’s\tShop \n')).toBe('Brennan’s Shop');
    expect(cleanShop('A B')).toBe('A B');
    expect([...cleanShop('x'.repeat(60))].length).toBe(SHOP_MAX);
    expect(cleanShop('   ')).toBe('');
  });
});

describe('plaque', () => {
  it('shrinks long names', () => {
    expect(plaqueFontSize('Short')).toBe(26);
    expect(plaqueFontSize('x'.repeat(40))).toBeLessThan(26);
    expect(plaqueFontSize('x'.repeat(40))).toBeGreaterThanOrEqual(14);
  });
  it('names the PNG after the poster and shop', () => {
    expect(slugFile('shave', 'Brennan’s Shop!')).toBe('dust-almanac-shave-brennan-s-shop.png');
    expect(slugFile('sunbeam', null)).toBe('dust-almanac-sunbeam.png');
  });
});
