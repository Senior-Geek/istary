// The perspective scene (M-ART-1A.1): what the party sees ahead, as a list
// of surfaces relative to the party, for the projection (mazeProjection.js)
// and any style that paints it (MazeView.svelte).
//
// It is made from Go's view alone, and from only four of its fields:
// view.dungeon.phase, .corridor, .dark and .spells. The corridor is the
// engine's Sight (engine/sight.go, through app.DungeonView): how far the
// party sees (two squares, five with light, three with QUICK plot), where a
// dark square or the first wall or door ahead stops it, and what a hidden
// door looks like, are all decided there. Nothing here decides them again,
// and nothing here reads the maps (view.minimap, the full map), the
// position or anything else: what was seen before is not seen now.
//
// The scene:
//   depths    how many squares are seen (0 is the party's own)
//   lit       the party has magical light (for a style's shading only)
//   end       what ends the view: { kind, depth }
//               'wall' or 'door'  the last square's front (depth: that square)
//               'dark'            a dark square (depth: where it would be)
//               'limit'           the sight runs out; what lies on is unknown
//               'none'            not in the maze: nothing is seen
//   surfaces  every surface seen, nearest square first, each
//               { id, depth, lane, face, seen }
//             id     stable: 'd<depth>.<place>', e.g. 'd2.front', 'd1.leftAhead'
//             lane   -1 left, 0 the corridor, +1 right
//             face   'side' (a left or right wall), 'front' (facing the
//                    party), 'floor' or 'ceiling'
//             seen   'wall' or 'door' for walls; 'open' for floors and ceilings
//   openings  each open side: { id, depth, lane, ahead } where ahead is what
//             Go says is in front of that side square ('wall', 'door' or
//             'open'); nothing further down a side passage is known.
//
// A square's floor and ceiling are implied by the square being seen (and an
// open side's, by the square beside it being seen through the opening).
// They add no information; a style may leave them unpainted.

const SIDES = [['left', -1], ['right', 1]]
const solid = (w) => w === 'wall' || w === 'door'

export function mazeScene(dungeon) {
  const inMaze = dungeon?.phase === 'maze'
  const corridor = inMaze ? dungeon.corridor ?? [] : []
  const dark = inMaze && !!dungeon.dark
  const spells = dungeon?.spells ?? []
  const lit = spells.includes('LIGHT')
  const surfaces = [], openings = []
  const surface = (depth, place, lane, face, seen) => surfaces.push({ id: `d${depth}.${place}`, depth, lane, face, seen })

  corridor.forEach((sq, d) => {
    surface(d, 'floor', 0, 'floor', 'open')
    surface(d, 'ceiling', 0, 'ceiling', 'open')
    for (const [side, lane] of SIDES) {
      const w = sq[side]
      if (solid(w)) {
        surface(d, side, lane, 'side', w)
        continue
      }
      // An open side: the square beside is seen, and the wall in front of
      // it if Go says there is one. Its own far side is not seen.
      const ahead = sq[side + 'Ahead']
      openings.push({ id: `d${d}.${side}`, depth: d, lane, ahead: solid(ahead) ? ahead : 'open' })
      surface(d, side + 'Floor', lane, 'floor', 'open')
      surface(d, side + 'Ceiling', lane, 'ceiling', 'open')
      if (solid(ahead)) surface(d, side + 'Ahead', lane, 'front', ahead)
    }
    if (solid(sq.front)) surface(d, 'front', 0, 'front', sq.front)
  })

  const n = corridor.length
  let end = { kind: 'none', depth: 0 }
  if (dark) end = { kind: 'dark', depth: n }
  else if (n && solid(corridor[n - 1].front)) end = { kind: corridor[n - 1].front, depth: n - 1 }
  else if (n) end = { kind: 'limit', depth: n }
  return { depths: n, lit, end, surfaces, openings }
}
