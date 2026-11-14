// The artwork of each place (Castle-0, ART-2A). Each entry names its
// image, the point of the picture that must stay in view when it is cropped
// (CSS object-position), its description for screen readers, and the emblem
// of the themed panel shown while there is no image, or if it fails to load.
//
// A place with an image is illustrated (ART-2A): the picture fills the
// window behind the location frame, cropped to cover it around its focus,
// never stretched, and the frame's panels stand over it (LocationFrame).
// A place without one keeps the themed panel in the frame's display area.
//
// To add a place's artwork, put the 1920x1080 WebP in
// src/assets/scenes/<place>.webp (the masters are kept outside Git; ART-2A's
// in images/Artwork Samples/, untracked) and set its src as castle's is:
// new URL(..., import.meta.url) is an asset to Vite and an ordinary URL to
// the Node tests. The picture is shown whole only with the panels hidden;
// keep what matters away from the right edge (the actions) and the bottom
// (the information and the party bar), and inside the middle 70% of the
// width, which narrow windows keep. Nothing here is game state, and the
// picture never changes what the player can do.

// Emblems are text, forced to text (not emoji) presentation.
const glyph = (c) => c + '︎'

export const scenes = {
  castle: {
    src: new URL('../assets/scenes/castle.webp', import.meta.url).href,
    focus: '45% 50%',
    alt: "The castle courtyard: Gilgamesh's Tavern, the Adventurer's Inn, Boltac's Trading Post and the Temple of Cant around a fountain, the Keep above them, and the gate out to the Training Grounds and the maze",
    emblem: glyph('♜'),
  },
  tavern: { src: null, focus: '50% 50%', alt: "Gilgamesh's Tavern", emblem: glyph('⚜') },
  inn: { src: null, focus: '50% 50%', alt: "The Adventurer's Inn", emblem: glyph('☾') },
  trading: { src: null, focus: '50% 50%', alt: "Boltac's Trading Post", emblem: glyph('⚖') },
  temple: { src: null, focus: '50% 45%', alt: 'The Temple of Cant', emblem: glyph('✠') },
  edge: {
    src: new URL('../assets/scenes/edge.webp', import.meta.url).href,
    focus: '50% 50%',
    alt: 'Outside the castle walls: the gatehouse, the Training Grounds on one side and the guarded entrance to the maze on the other',
    emblem: glyph('⚑'),
  },
  training: {
    src: null,
    focus: '50% 55%',
    alt: 'The Training Grounds: a tournament field outside the castle walls',
    emblem: glyph('⚔'),
  },
  settings: { src: null, focus: '50% 50%', alt: 'Settings', emblem: glyph('⚙') },
  maze: { src: null, focus: '50% 50%', alt: 'The maze', emblem: glyph('▦') },
}

// fallback is the scene of a place with no entry.
export const fallback = { src: null, focus: '50% 50%', alt: '', emblem: glyph('✦') }

export function sceneFor(location) {
  return scenes[location] ?? fallback
}

// illustrated says whether a place has artwork to fill the window with.
export function illustrated(location) {
  return !!scenes[location]?.src
}

// PANELS is the context key of the illustrated places' window state, which
// App keeps and LocationFrame offers (ART-2A, ART-2B, ART-2C).
//   hidden, hide(), show()   the panels hidden to look at the picture; they
//                            come back at every other place
//   look                     { panels, story, artwork }: the appearance
//                            chosen in Settings (see looks), kept by Go in
//                            the window's settings, not in any game
//   story.open, story.rect   the story box open (until it is closed, and
//                            then until it is opened again), and where the
//                            player left it ({ x, y, w, h }, or null); the
//                            window's alone, for the session
export const PANELS = Symbol('panels')

// The illustrated places' appearances (ART-2C), chosen in Settings, each
// list's default first (as Go's, settings.go). The panels' and the story
// box's change only colours (style.css, [data-appearance]); the picture
// is shown whole, subdued (darker and less saturated: the panels are
// unchanged) or not at all.
export const looks = {
  panels: [
    { id: 'slate', label: 'Slate' },
    { id: 'dark', label: 'Dark' },
    { id: 'light', label: 'Light' },
  ],
  story: [
    { id: 'parchment', label: 'Parchment' },
    { id: 'dark', label: 'Dark' },
    { id: 'slate', label: 'Slate' },
  ],
  artwork: [
    { id: 'full', label: 'Full' },
    { id: 'subdued', label: 'Subdued' },
    { id: 'hidden', label: 'Hidden' },
  ],
}

// lookOf is the appearance in Go's settings view, each part its default
// if it is missing or unknown.
export function lookOf(settings) {
  const pick = (k) => (looks[k].some((o) => o.id === settings?.[k]) ? settings[k] : looks[k][0].id)
  return { panels: pick('panels'), story: pick('story'), artwork: pick('artwork') }
}
