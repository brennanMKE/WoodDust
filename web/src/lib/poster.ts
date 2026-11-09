// Shop posters (docs/posters.md): fill the hand-made SVG template's slots in
// the browser, and draw the same sheet onto a canvas for the PNG export.
// The template uses presentation attributes only, so the filled SVG can be
// inlined under the strict CSP. Pure functions here; DOM access is limited
// to DOMParser/XMLSerializer and the canvas.

import type { PosterLink, PosterView } from './types';

export const SHEET_W = 1024;
export const SHEET_H = 1536;
export const SHOP_MAX = 40;
// Control characters and the Unicode line/paragraph separators.
const CONTROL = new RegExp('[\\u0000-\\u001f\\u007f\\u2028\\u2029]', 'g');
const BLANK_PLAQUE = '________________________';

export interface PosterFill {
  /** Art URL, or null to leave the placeholder block showing. */
  art: string | null;
  qrSVG: string;
  /** null: no plaque. '': a plaque with a blank line to write on. */
  plaque: string | null;
}

export function svgDataURI(svg: string): string {
  return 'data:image/svg+xml;base64,' + btoa(unescape(encodeURIComponent(svg)));
}

/** Trim and cap a shop name the way the server does. */
export function cleanShop(s: string): string {
  const one = s.replace(CONTROL, ' ').replace(/\s+/g, ' ').trim();
  return [...one].slice(0, SHOP_MAX).join('').trim();
}

/** Plaque name size: shrink long names so they stay on the plaque. */
export function plaqueFontSize(name: string): number {
  const n = [...name].length;
  return n <= 22 ? 26 : Math.max(15, Math.floor((26 * 22) / n));
}

/** Parse a template and fill its slots. Returns the <svg> element. */
export function fillTemplate(template: string, fill: PosterFill, label: string): SVGSVGElement | null {
  if (!template) return null;
  const doc = new DOMParser().parseFromString(template, 'image/svg+xml');
  const svg = doc.documentElement as unknown as SVGSVGElement;
  if (svg.nodeName !== 'svg' || doc.getElementsByTagName('parsererror').length) return null;
  const slot = (name: string) => svg.querySelector(`[data-slot="${name}"]`);

  const art = slot('art');
  if (fill.art) art?.setAttribute('href', fill.art);
  else art?.remove();
  slot('qr')?.setAttribute('href', svgDataURI(fill.qrSVG));

  const plaque = slot('plaque');
  if (fill.plaque !== null && plaque) {
    plaque.removeAttribute('display');
    const name = slot('plaque-name');
    if (name) {
      const text = fill.plaque || BLANK_PLAQUE;
      name.textContent = text;
      name.setAttribute('font-size', String(fill.plaque ? plaqueFontSize(text) : 26));
    }
  }
  svg.removeAttribute('width');
  svg.removeAttribute('height');
  svg.setAttribute('role', 'img');
  svg.setAttribute('aria-label', label);
  return svg;
}

export function fillTemplateString(template: string, fill: PosterFill, label: string): string {
  const svg = fillTemplate(template, fill, label);
  return svg ? new XMLSerializer().serializeToString(svg) : '';
}

export function posterLabel(p: PosterView, plaque: string | null): string {
  const base = `Poster: “${p.slogan}” with a QR code that opens ${p.destination}.`;
  return plaque ? `${base} Posted by ${plaque}.` : base;
}

// ---- canvas export ------------------------------------------------------------

function loadImage(src: string): Promise<HTMLImageElement | null> {
  return new Promise((resolve) => {
    const img = new Image();
    img.decoding = 'async';
    img.onload = () => resolve(img);
    img.onerror = () => resolve(null);
    img.src = src;
  });
}

const num = (el: Element, a: string, d = 0) => {
  const v = parseFloat(el.getAttribute(a) ?? '');
  return Number.isFinite(v) ? v : d;
};

function hidden(el: Element): boolean {
  for (let e: Element | null = el; e; e = e.parentElement) if (e.getAttribute('display') === 'none') return true;
  return false;
}

function roundRect(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
  ctx.beginPath();
  if (r > 0 && 'roundRect' in ctx) ctx.roundRect(x, y, w, h, r);
  else ctx.rect(x, y, w, h);
}

/**
 * Draw a filled poster SVG onto a canvas, element by element: the subset the
 * templates use (rect, image, text with tspans). Fonts are the page's own
 * (web/public/fonts), so they must be loaded first. Patterns (the missing-art
 * placeholder hatching) are drawn as their base colour.
 */
export async function renderToCanvas(svg: SVGSVGElement, scale = 2): Promise<HTMLCanvasElement> {
  const canvas = document.createElement('canvas');
  canvas.width = SHEET_W * scale;
  canvas.height = SHEET_H * scale;
  const ctx = canvas.getContext('2d');
  if (!ctx) throw new Error('This browser can’t draw the poster.');
  await Promise.all([
    document.fonts.load(`96px "Alfa Slab One"`),
    document.fonts.load(`600 26px "Source Sans 3"`),
    document.fonts.load(`700 22px "Source Sans 3"`),
  ]).catch(() => undefined);
  ctx.scale(scale, scale);

  const els = Array.from(svg.querySelectorAll('rect, image, text')).filter(
    (el) => !hidden(el) && !el.closest('defs, pattern') && el.parentElement?.nodeName !== 'text',
  );
  for (const el of els) {
    if (el.nodeName === 'rect') {
      const fill = el.getAttribute('fill') ?? '#000';
      const x = num(el, 'x'),
        y = num(el, 'y'),
        w = num(el, 'width', SHEET_W),
        h = num(el, 'height', SHEET_H),
        r = num(el, 'rx');
      roundRect(ctx, x, y, w, h, r);
      if (fill !== 'none' && !fill.startsWith('url(')) {
        ctx.fillStyle = fill;
        ctx.fill();
      }
      const stroke = el.getAttribute('stroke');
      if (stroke && stroke !== 'none' && !el.closest('[data-slot="art-placeholder"]')) {
        ctx.strokeStyle = stroke;
        ctx.lineWidth = num(el, 'stroke-width', 1);
        ctx.stroke();
      }
    } else if (el.nodeName === 'image') {
      const href = el.getAttribute('href');
      if (!href) continue;
      const img = await loadImage(href);
      if (!img) {
        if (el.getAttribute('data-slot') === 'qr') throw new Error('The QR code didn’t load.');
        continue; // missing art: the placeholder underneath stays
      }
      const x = num(el, 'x'),
        y = num(el, 'y'),
        w = num(el, 'width'),
        h = num(el, 'height');
      if (el.getAttribute('data-slot') === 'qr') ctx.imageSmoothingEnabled = false;
      // Art is 2:3 like the sheet, so "slice" and "meet" both fill the box.
      ctx.drawImage(img, x, y, w, h);
      ctx.imageSmoothingEnabled = true;
    } else {
      drawText(ctx, el);
    }
  }
  return canvas;
}

function drawText(ctx: CanvasRenderingContext2D, el: Element) {
  const family = el.getAttribute('font-family') ?? 'serif';
  const size = num(el, 'font-size', 16);
  const weight = el.getAttribute('font-weight') ?? '400';
  ctx.font = `${weight} ${size}px ${family}`;
  ctx.fillStyle = el.getAttribute('fill') ?? '#000';
  const anchor = el.getAttribute('text-anchor');
  ctx.textAlign = anchor === 'middle' ? 'center' : anchor === 'end' ? 'right' : 'left';
  ctx.textBaseline = 'alphabetic';
  const spacing = el.getAttribute('letter-spacing');
  if ('letterSpacing' in ctx) (ctx as CanvasRenderingContext2D & { letterSpacing: string }).letterSpacing = spacing ? `${spacing}px` : '0px';
  const spans = Array.from(el.querySelectorAll('tspan'));
  if (spans.length) {
    for (const t of spans) ctx.fillText(t.textContent ?? '', num(t, 'x', num(el, 'x')), num(t, 'y', num(el, 'y')));
  } else {
    ctx.fillText(el.textContent ?? '', num(el, 'x'), num(el, 'y'));
  }
}

export function canvasToPNG(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    try {
      canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('Couldn’t make the PNG.'))), 'image/png');
    } catch {
      reject(new Error('This browser won’t export the picture. Use “Print or save as PDF” instead.'));
    }
  });
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 10_000);
}

export function slugFile(id: string, plaque: string | null): string {
  const shop = (plaque ?? '')
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
    .slice(0, 30);
  return `dust-almanac-${id}${shop ? '-' + shop : ''}.png`;
}

// ---- the plaque's own link ---------------------------------------------------------

export interface LinkRequest {
  poster: string;
  shop?: string;
  form_token: string;
  website: string;
}

export type LinkResult = { link: PosterLink; notice: string };

/** POST /api/v1/posters/links. Never throws for ShortLinks trouble: the
 * server falls back to the stock link, and so does this on 429. */
export async function requestLink(req: LinkRequest, stock: PosterLink): Promise<LinkResult> {
  const res = await fetch('/api/v1/posters/links', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify(req),
  });
  if (res.status === 429) {
    return { link: stock, notice: 'Lots of posters made today, so this one uses the standard QR code. It works just the same.' };
  }
  const body = await res.json().catch(() => null);
  if (res.status === 422 && body?.error?.fields?.form_token) throw new FormTokenExpired();
  if (!res.ok || !body?.short_url || !body?.qr_svg) {
    return { link: stock, notice: 'We couldn’t make a separate code just now, so this poster uses the standard one. It works just the same.' };
  }
  const link = body as PosterLink;
  return {
    link,
    notice: link.fallback ? 'This poster uses the standard QR code for now. It works just the same.' : '',
  };
}

export class FormTokenExpired extends Error {
  constructor() {
    super('form token expired');
  }
}
