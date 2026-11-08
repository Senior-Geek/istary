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

// back is the way back to the Castle (Go's view.return: { open, note }),
// labelled, or null where there is none (at the Castle, out of town). It
// abandons what is unfinished; note says what, if anything.
export function back(view) {
  return view.return ? { ...view.return, dest: 'castle', label: 'Castle' } : null
}

// destinations are the exits other than the way back, in Go's order,
// labelled.
export function destinations(view) {
  const home = places[view.location]?.parent
  return (view.exits ?? []).filter((e) => e.dest !== home).map((e) => ({ ...e, label: title(e.dest) }))
}
