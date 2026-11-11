// The rescue's progress on the Inspect screen (R-3), from Go's
// app.InspectView: the party's size and room, who else is here, and how
// many are known to lie elsewhere. Every number is Go's, read from the
// game as it stands; nothing here is remembered between views.

const plural = (n, one, many) => `${n} ${n === 1 ? one : many}`

// progress is Inspect's lines: party (size and room), here (who else is
// in the party's area, or that no one is), elsewhere ('' when none).
export function progress(inspect) {
  if (!inspect) return { party: '', here: '', elsewhere: '' }
  const found = inspect.found?.length ?? 0
  const party = `PARTY ${inspect.size} OF 6 · ROOM FOR ${inspect.room}`
  let here
  if (found === 0) here = inspect.picked ? 'NO ONE ELSE IS HERE.' : 'LOOKING... NO ONE IS HERE.'
  else if (inspect.picked) here = `${plural(found, 'MORE IS', 'MORE ARE')} HERE.`
  else here = `${plural(found, 'CHARACTER IS', 'CHARACTERS ARE')} HERE.`
  const elsewhere = inspect.elsewhere > 0
    ? `${plural(inspect.elsewhere, 'MORE LIES', 'MORE LIE')} ELSEWHERE (MAP, F10).`
    : ''
  return { party, here, elsewhere }
}
