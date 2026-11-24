// Color math: canonical representation is OKLab + alpha (unclipped).
export const lin = c => { const a = Math.abs(c); return a <= 0.04045 ? c / 12.92 : Math.sign(c) * Math.pow((a + 0.055) / 1.055, 2.4); };
export const gam = c => { const a = Math.abs(c); return a <= 0.0031308 ? c * 12.92 : Math.sign(c) * (1.055 * Math.pow(a, 1 / 2.4) - 0.055); };
export function oklabToLinSrgb(L, a, b) {
  const l_ = L + 0.3963377774 * a + 0.2158037573 * b, m_ = L - 0.1055613458 * a - 0.0638541728 * b, s_ = L - 0.0894841775 * a - 1.2914855480 * b;
  const l = l_ ** 3, m = m_ ** 3, s = s_ ** 3;
  return [4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s, -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s, -0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s];
}
export function linSrgbToOklab(r, g, b) {
  const l = Math.cbrt(0.4122214708 * r + 0.5363020883 * g + 0.0514459929 * b), m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b), s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
  return [0.2104542553 * l + 0.7936177850 * m - 0.0040720468 * s, 1.9779984951 * l - 2.4285922050 * m + 0.4505937099 * s, 0.0259040371 * l + 0.7827717662 * m - 0.8086757660 * s];
}
export const fromSrgb = (r, g, b, alpha = 1) => { const [L, a, bb] = linSrgbToOklab(lin(r), lin(g), lin(b)); return { L, a, b: bb, alpha }; };
export const toSrgb = c => oklabToLinSrgb(c.L, c.a, c.b).map(gam);
export const fromOklch = (L, C, H, alpha = 1) => ({ L, a: C * Math.cos(H * Math.PI / 180), b: C * Math.sin(H * Math.PI / 180), alpha });
export const toOklch = c => { const C = Math.hypot(c.a, c.b); let H = Math.atan2(c.b, c.a) * 180 / Math.PI; if (H < 0) H += 360; return [c.L, C, H]; };
const inGamut = rgb => rgb.every(v => v >= -2e-3 && v <= 1 + 2e-3);
const clip = rgb => rgb.map(v => Math.min(1, Math.max(0, v)));
const dEOK = (c1, c2) => Math.hypot(c1.L - c2.L, c1.a - c2.a, c1.b - c2.b);
// CSS Color 4 gamut mapping (binary search on OKLCh chroma, JND 0.02)
export function gamutMap(c) {
  const rgb = toSrgb(c); if (inGamut(rgb)) return clip(rgb);
  const [L, C, H] = toOklch(c); if (L >= 1) return [1, 1, 1]; if (L <= 0) return [0, 0, 0];
  let lo = 0, hi = C, minInGamut = true; const JND = 0.02, EPS = 0.0001;
  let cur = { ...c };
  const clipOk = x => { const cl = clip(toSrgb(x)); return fromSrgb(...cl); };
  let clipped = clipOk(cur);
  if (dEOK(clipped, cur) < JND) return clip(toSrgb(cur));
  while (hi - lo > EPS) {
    const ch = (lo + hi) / 2; cur = fromOklch(L, ch, H);
    if (minInGamut && inGamut(toSrgb(cur))) { lo = ch; continue; }
    clipped = clipOk(cur); const e = dEOK(clipped, cur);
    if (e < JND) { if (JND - e < EPS) break; minInGamut = false; lo = ch; } else hi = ch;
  }
  return clip(toSrgb(clipped));
}
const h2 = v => Math.round(Math.min(1, Math.max(0, v)) * 255).toString(16).padStart(2, '0');
export function hex(c, mode = 'clip') {
  const rgb = mode === 'map' ? gamutMap(c) : clip(toSrgb(c));
  const a = Math.round(c.alpha * 1000) / 1000;
  return '#' + rgb.map(h2).join('') + (a >= 1 ? '' : h2(a));
}
export const isOutOfGamut = c => !inGamut(toSrgb(c));
export function parseHex(s) {
  let h = s.slice(1); if (h.length === 3 || h.length === 4) h = [...h].map(x => x + x).join('');
  const n = i => parseInt(h.slice(i, i + 2), 16) / 255;
  return fromSrgb(n(0), n(2), n(4), h.length === 8 ? n(6) : 1);
}
// premultiplied mix. space: 'oklab' | 'srgb'. p1,p2 fractions (already normalized), mult = alpha multiplier
export function mix(space, c1, p1, c2, p2, mult = 1) {
  let v1, v2;
  if (space === 'srgb') { v1 = toSrgb(c1); v2 = toSrgb(c2); } else if (space === 'srgb-linear') { v1 = oklabToLinSrgb(c1.L, c1.a, c1.b); v2 = oklabToLinSrgb(c2.L, c2.a, c2.b); } else { v1 = [c1.L, c1.a, c1.b]; v2 = [c2.L, c2.a, c2.b]; }
  const a = c1.alpha * p1 + c2.alpha * p2;
  let out = a === 0 ? [0, 0, 0] : v1.map((x, i) => (x * c1.alpha * p1 + v2[i] * c2.alpha * p2) / a);
  let res;
  if (space === 'srgb') res = fromSrgb(...out, a); else if (space === 'srgb-linear') { const [L, aa, b] = linSrgbToOklab(...out); res = { L, a: aa, b, alpha: a }; } else res = { L: out[0], a: out[1], b: out[2], alpha: a };
  res.alpha *= mult; return res;
}
export const NAMED = { white: fromSrgb(1, 1, 1), black: fromSrgb(0, 0, 0), transparent: { L: 0, a: 0, b: 0, alpha: 0 } };
