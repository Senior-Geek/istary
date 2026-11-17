// How a style paints the projection's primitives (mazeProjection.js). A
// style chooses only the class each kind of primitive is painted with, or
// null to leave it unpainted; it never adds, moves or removes geometry.
//
// Classic (M-ART-1A.1): the Wizardry wireframe. Walls are outlined and
// filled with the background, so nearer walls hide what lies behind them;
// doors are outlined in gold; the void where the view ends is the
// background; floors and ceilings are not drawn.
export const classic = { wall: 'wall', door: 'door', void: 'void', floor: null, ceiling: null }

// painted is the primitives a style draws, in painting order, each with its
// paint class.
export const painted = (primitives, style = classic) =>
  primitives.filter((p) => style[p.kind]).map((p) => ({ ...p, paint: style[p.kind] }))
