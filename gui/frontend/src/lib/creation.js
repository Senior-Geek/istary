// Display helpers for character creation at the Training Grounds (UX-3B).
// They reshape Go's CreationView (view.creation) and decide no rule: what
// each step allows, the class requirements, every eligibility and its
// reason, and which points may move are Go's.
//
// The bonus points and the class are one screen: the class table is shown
// as points are spent, and once none are left the classes Go allows are
// chosen from it (Session.PickClass). The "class" step, reached only by
// the older two-step route (FinishStats), shows the same table, its
// classes chosen with Session.ChooseClass.

// stage names where creation is, for the page: confirming (the game's
// "DO YOU WANT TO CREATE IT?"), the creation steps, and the class choice
// within the points.
export function stage(view) {
  if (view?.screen === 'confirmCreate') return 'confirming'
  const c = view?.creation
  if (view?.screen !== 'creation' || !c) return ''
  if (c.step === 'stats') return c.points > 0 ? 'points' : 'choose'
  return c.step // name, password, race, alignment, class, confirm
}

// choosing says whether classes are chosen from the table now.
export function choosing(view) {
  const s = stage(view)
  return s === 'choose' || s === 'class'
}

// classRows is the class table: each class, in Go's order (its number for
// PickClass and ChooseClass), its minimums and shortfalls, its alignments,
// and Go's status and reason. A row is pickable when classes are chosen
// now and Go allows it (classOk).
export function classRows(view) {
  const c = view?.creation
  if (!c?.guide) return []
  const pick = choosing(view)
  return c.guide.map((row, i) => ({
    ...row,
    index: i,
    eligible: !!c.classOk?.[i],
    pickable: pick && !!c.classOk?.[i],
  }))
}

// instruction is the next decision, in words, for the Actions.
export function instruction(view) {
  const c = view?.creation
  switch (stage(view)) {
    case 'confirming':
      return 'Create this character?'
    case 'name':
      return 'Name the character.'
    case 'password':
      return 'Choose a password, or leave both empty for none.'
    case 'race':
      return 'Choose a race.'
    case 'alignment':
      return 'Choose an alignment.'
    case 'points':
      return `Spend the bonus points: ${c.points} left.`
    case 'choose':
    case 'class':
      return c.classOk?.some(Boolean)
        ? 'Choose a highlighted class.'
        : 'No class is open to these statistics: move points to qualify for one.'
    case 'confirm':
      return 'Keep this character? Hit points, gold and age are rolled when it is kept.'
  }
  return ''
}
