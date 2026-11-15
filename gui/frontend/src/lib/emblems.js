// The places' emblems (ART-3B-0): one for each place, beside its title and
// on its themed panel while it has no picture. Each is the approved SVG in
// docs/design/art3/emblems/emblem-<file>.svg, as data (emblems.test.js
// checks the two are the same drawing): a 64x64 viewBox, outlines stroke 3
// and details stroke 2, in currentColor, so the appearance's gold colours
// it (Emblem.svelte). Drawn by Svelte as ordinary elements, never as HTML.
// A place without one (Settings) has none in its title.

export const emblems = {
  castle: { file: 'castle-tower', shapes: [
    { tag: 'path', d: 'M10 58 H54' },
    { tag: 'path', d: 'M18 58 V24 M46 24 V58' },
    { tag: 'path', d: 'M14 24 H50 V11 H43 V17 H37 V11 H27 V17 H21 V11 H14 Z' },
    { tag: 'path', d: 'M27 58 V48 A5 5 0 0 1 37 48 V58' },
    { tag: 'path', d: 'M32 30 V37', 'stroke-width': '2' },
  ] },
  edge: { file: 'edge-gatehouse', shapes: [
    { tag: 'path', d: 'M4 58 H60' },
    { tag: 'path', d: 'M6 58 V10 H10 V14 H14 V10 H18 V58' },
    { tag: 'path', d: 'M46 58 V10 H50 V14 H54 V10 H58 V58' },
    { tag: 'path', d: 'M18 25 V20 H23 V25 H29 V20 H35 V25 H41 V20 H46 V25' },
    { tag: 'path', d: 'M24 58 V42 A8 8 0 0 1 40 42 V58' },
    { tag: 'path', d: 'M28 36 V58 M32 34.5 V58 M36 36 V58 M24.5 45 H39.5 M24 51.5 H40', 'stroke-width': '2' },
  ] },
  tavern: { file: 'tavern-tankard', shapes: [
    { tag: 'path', d: 'M10 58 H50' },
    { tag: 'path', d: 'M18 21 H40 L38.5 56 H19.5 Z' },
    { tag: 'path', d: 'M18 21 Q15 14 21 13 Q23 8 29 10 Q33 7 37 10 Q43 10 40 16 Q41 19 40 21' },
    { tag: 'path', d: 'M40 27 C53 27 53 47 39 47' },
    { tag: 'path', d: 'M19 31 H39 M19.5 49 H38.5', 'stroke-width': '2' },
  ] },
  inn: { file: 'inn-bed', shapes: [
    { tag: 'path', d: 'M9 57 V22 A3 3 0 0 1 15 22 V57' },
    { tag: 'path', d: 'M51 57 V36 A3 3 0 0 1 57 36 V57' },
    { tag: 'path', d: 'M15 48 H51' },
    { tag: 'path', d: 'M15 39 H51' },
    { tag: 'path', d: 'M18 39 V34 Q18 31 21 31 H27 Q30 31 30 34 V39' },
    { tag: 'path', d: 'M33 39 V37 Q33 34 36 34 H51', 'stroke-width': '2' },
  ] },
  trading: { file: 'trading-balance-scale', shapes: [
    { tag: 'path', d: 'M24 58 L28 54 H36 L40 58 Z' },
    { tag: 'path', d: 'M32 54 V13' },
    { tag: 'circle', cx: '32', cy: '9.5', r: '2.5', fill: 'currentColor' },
    { tag: 'path', d: 'M12 16 H52' },
    { tag: 'path', d: 'M12 16 L2 40 M12 16 L22 40 M52 16 L42 40 M52 16 L62 40', 'stroke-width': '2' },
    { tag: 'path', d: 'M2 40 Q12 49 22 40 Z M42 40 Q52 49 62 40 Z' },
    { tag: 'circle', cx: '12', cy: '34', r: '5', fill: 'currentColor' },
  ] },
  temple: { file: 'temple-star-arch', shapes: [
    { tag: 'path', d: 'M10 58 H54' },
    { tag: 'path', d: 'M14 58 V32 A26 26 0 0 1 32 13.2 A26 26 0 0 1 50 32 V58' },
    { tag: 'path', d: 'M32.0 26.0 L33.1 36.7 L37.3 34.2 L34.8 38.4 L45.5 39.5 L34.8 40.6 L37.3 44.8 L33.1 42.3 L32.0 53.0 L30.9 42.3 L26.7 44.8 L29.2 40.6 L18.5 39.5 L29.2 38.4 L26.7 34.2 L30.9 36.7 Z', fill: 'currentColor', 'stroke-width': '1.5' },
  ] },
  training: { file: 'training-crossed-swords', shapes: [
    { tag: 'path', d: 'M21 47 L50 18 L54 10 L46 14 L17 43 Z' },
    { tag: 'path', d: 'M43 47 L14 18 L10 10 L18 14 L47 43 Z' },
    { tag: 'path', d: 'M13 39 L25 51 M51 39 L39 51' },
    { tag: 'path', d: 'M19 45 L13 51 M45 45 L51 51' },
    { tag: 'circle', cx: '11', cy: '53', r: '2.5' },
    { tag: 'circle', cx: '53', cy: '53', r: '2.5' },
  ] },
  maze: { file: 'maze-stone-archway', shapes: [
    { tag: 'path', d: 'M20 58 V40 L23 32 L28 27 L36 27 L41 32 L44 40 V58 Z', fill: '#120f0a', stroke: 'none' },
    { tag: 'path', d: 'M5 58 H59' },
    { tag: 'path', d: 'M8 58 L9 41 L11.5 29 L18.5 18 L28 11.5 L37 11 L46.5 17 L52.5 27.5 L55 40 L56 58' },
    { tag: 'path', d: 'M20 58 V40 L23 32 L28 27 L36 27 L41 32 L44 40 V58' },
    { tag: 'path', d: 'M20 47 L8.6 46 M21.5 35.5 L10.5 33.5 M25 29.5 L17 19.5 M29 27 L28 11.5 M35 27 L37 11 M39 29.5 L47.5 18.5 M42.5 35.5 L53.5 34 M44 47 L55.5 47.5', 'stroke-width': '2' },
    { tag: 'path', d: 'M23 54 H41 M25.5 50 H38.5 M27.5 46.5 H36.5 M29 43.5 H35', 'stroke-width': '2' },
  ] },

}

// emblemFor is the emblem of location, or null.
export function emblemFor(location) {
  return emblems[location] ?? null
}
