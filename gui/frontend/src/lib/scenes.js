// The artwork of each place (Castle-0). Each entry names its image, the
// point of the picture that must stay in view when it is cropped (CSS
// object-position), its description for screen readers, and the emblem of
// the themed panel shown while there is no image, or if it fails to load.
//
// No artwork is shipped yet: every src is null. To add one, put the
// 1920x1080 WebP in src/assets/scenes/<place>.webp (masters, 2560x1440,
// are kept outside Git), import it here and set src:
//
//   import castle from '../assets/scenes/castle.webp'
//   castle: { src: castle, ... }
//
// Keep what matters inside the middle 60% of the width and out of the top
// and bottom fifth: narrow windows show the picture as a 21:9 banner.
// Nothing here is game state, and the picture never changes what the
// player can do.

// Emblems are text, forced to text (not emoji) presentation.
const glyph = (c) => c + '︎'

export const scenes = {
  castle: { src: null, focus: '50% 40%', alt: 'The castle market', emblem: glyph('♜') },
  tavern: { src: null, focus: '50% 50%', alt: "Gilgamesh's Tavern", emblem: glyph('⚜') },
  inn: { src: null, focus: '50% 50%', alt: "The Adventurer's Inn", emblem: glyph('☾') },
  trading: { src: null, focus: '50% 50%', alt: "Boltac's Trading Post", emblem: glyph('⚖') },
  temple: { src: null, focus: '50% 45%', alt: 'The Temple of Cant', emblem: glyph('✠') },
  edge: { src: null, focus: '50% 50%', alt: 'The Edge of Town, at the entrance to the maze', emblem: glyph('⚑') },
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
