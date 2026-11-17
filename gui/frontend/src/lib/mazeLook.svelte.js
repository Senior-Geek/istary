// The maze picture's style and zoom (M-ART-1A.2), chosen on the dungeon
// screen while Enhanced is reviewed. It lasts while the window is open and
// is never saved: no Session call, no gui-settings.json. Settings and the
// dungeon header's own controls replace it in M-ART-1A.3.

import { defaultLook } from './mazeStyle.js'

export const look = $state({ ...defaultLook })
