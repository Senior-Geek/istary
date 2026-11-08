// Display helpers for the location frame (Castle-0). They only name and
// arrange what Go sent: where the player is (view.location) and the ways
// out of it (view.exits, each open or closed with Go's reason). Which ways
// exist, and whether one is open, are Go's; nothing here decides a move.

// Every place Go can report, with its title and the place its Back leads
// to (the original's L)EAVE: the Castle, from everywhere in town).
export const places = {
  castle: { title: 'The Castle', parent: null },
  tavern: { title: "Gilgamesh's Tavern", parent: 'castle' },
  inn: { title: "The Adventurer's Inn", parent: 'castle' },
  trading: { title: "Boltac's Trading Post", parent: 'castle' },
  temple: { title: 'The Temple of Cant', parent: 'castle' },
  edge: { title: 'The Edge of Town', parent: 'castle' },
  training: { title: 'The Training Grounds', parent: 'castle' },
  settings: { title: 'Settings', parent: 'castle' },
  maze: { title: 'The Maze', parent: null },
}

export function title(location) {
  return places[location]?.title ?? location ?? ''
}

// back is the exit to the location's parent, as Go lists it ({ dest, open,
// reason, label }), or null where there is none (the Castle, the maze).
export function back(view) {
  const parent = places[view.location]?.parent
  const exit = view.exits?.find((e) => e.dest === parent)
  return exit ? { ...exit, label: title(parent) } : null
}

// destinations are the other exits, in Go's order, labelled.
export function destinations(view) {
  const b = back(view)
  return (view.exits ?? []).filter((e) => e.dest !== b?.dest).map((e) => ({ ...e, label: title(e.dest) }))
}
