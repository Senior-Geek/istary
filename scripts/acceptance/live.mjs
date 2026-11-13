#!/usr/bin/env node
// live.mjs drives the real window (docs/TESTING.md): `wails dev` in GUI_DIR,
// on a scratch library, read through headless Chrome over the DevTools
// protocol, with the bound Go methods behind it. It checks the town
// interface as a player meets it: the shared layout at every place, the
// embedded character sheet with its equipment lists, the illustrated places
// (the Castle and the Edge of Town: their pictures, Hide / Show panels, in
// windows from the smallest to a large display), and character creation at
// the Training Grounds.
//
//   node scripts/acceptance/live.mjs --gui GUI_DIR --library DIR --out DIR
//        [--size 980x692] [--suites layout,sheet,scenes,creation] [--wails PATH] [--chrome PATH]
//
// The library must have been written by TestWriteAcceptanceLibrary
// (gui/acceptance_seed_test.go). Everything it starts (wails dev, the app,
// Vite, Chrome) is in process groups of its own, stopped at the end, on an
// error and on a signal; it never stops anything else. Exit 0 only if every
// check passed.

import { spawn, execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { createServer } from 'node:net'
import { appendFileSync, createWriteStream, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const arg = (name, dflt) => { const i = process.argv.indexOf(`--${name}`); return i > 0 ? process.argv[i + 1] : dflt }
const GUI = arg('gui'), LIB = arg('library'), OUT = arg('out')
const [W, H] = arg('size', '980x692').split('x').map(Number)
const SUITES = arg('suites', 'layout,sheet,scenes,creation').split(',').filter(Boolean)
const WAILS = arg('wails', 'wails')
const CHROME = arg('chrome', '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome')
if (!GUI || !LIB || !OUT || !W || !H) { console.error('usage: live.mjs --gui GUI_DIR --library DIR --out DIR [--size WxH] [--suites ...]'); process.exit(2) }
mkdirSync(OUT, { recursive: true })

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const freePort = () => new Promise((r) => { const s = createServer().listen(0, '127.0.0.1', () => { const p = s.address().port; s.close(() => r(p)) }) })

// --- Processes of our own -----------------------------------------------------------

const groups = new Set()
function start(cmd, args, opts, logFile) {
  const p = spawn(cmd, args, { ...opts, detached: true, stdio: ['ignore', 'pipe', 'pipe'] })
  groups.add(p.pid)
  // Recorded, so the level can check afterwards that none is left.
  appendFileSync(join(OUT, 'groups'), `${p.pid} ${cmd}\n`)
  const log = createWriteStream(logFile)
  p.stdout.pipe(log); p.stderr.pipe(log)
  p.on('exit', () => groups.delete(p.pid))
  return p
}
const alive = (pgid) => { try { process.kill(-pgid, 0); return true } catch { return false } }
async function stopAll() {
  const gs = [...groups]
  for (const g of gs) { try { process.kill(-g, 'SIGTERM') } catch {} }
  for (let i = 0; i < 50 && gs.some(alive); i++) await sleep(100)
  for (const g of gs) { try { process.kill(-g, 'SIGKILL') } catch {} }
  for (let i = 0; i < 30 && gs.some(alive); i++) await sleep(100)
  return gs.filter(alive)
}
for (const [sig, code] of [['SIGINT', 130], ['SIGTERM', 143], ['SIGHUP', 129]]) {
  process.on(sig, async () => { console.log(`live: interrupted (${sig}), stopping its own processes`); await stopAll(); process.exit(code) })
}

// --- Checks ---------------------------------------------------------------------------

const passed = [], fails = [], problems = []
const check = (ok, what) => { (ok ? passed : fails).push(what); console.log(`${ok ? 'ok  ' : 'FAIL'} ${what}`); return ok }

async function main() {
  // wails dev on a port of its own, on the scratch library.
  const port = await freePort()
  const URL = `http://127.0.0.1:${port}/`
  const devLog = join(OUT, 'wails-dev.log')
  const dev = start(WAILS, ['dev', '-noreload', '-devserver', `127.0.0.1:${port}`], { cwd: GUI, env: { ...process.env, ISTARY_SAVE_DIR: LIB } }, devLog)
  let ready = false
  for (let i = 0; i < 240 && !ready; i++) {
    await sleep(500)
    const log = readFileSync(devLog, 'utf8')
    ready = log.includes(`navigate to: http://127.0.0.1:${port}`) || log.includes(`navigate to: http://localhost:${port}`)
    if (dev.exitCode !== null) break
  }
  if (!check(ready, `wails dev is serving the window on port ${port}`)) throw new Error(`wails dev did not start: see ${devLog}`)
  // The dev app opens a window of its own: hide it, found by its parent (ours).
  try {
    const pids = execFileSync('pgrep', ['-f', `${GUI}/build/bin/`], { encoding: 'utf8' }).trim().split('\n').filter(Boolean)
    for (const pid of pids) execFileSync('osascript', ['-e', `tell application "System Events" to set visible of (first process whose unix id is ${pid}) to false`], { stdio: 'ignore' })
  } catch {}

  // Headless Chrome, on a profile and port of its own.
  const dbg = await freePort()
  start(CHROME, ['--headless=new', '--disable-gpu', `--remote-debugging-port=${dbg}`, `--window-size=${W},${H}`, `--user-data-dir=${join(OUT, 'chrome-profile')}`, '--no-first-run', '--no-default-browser-check', '--use-mock-keychain', '--password-store=basic', 'about:blank'], {}, join(OUT, 'chrome.log'))
  let ws
  for (let i = 0; i < 100 && !ws; i++) {
    try { const p = (await (await fetch(`http://127.0.0.1:${dbg}/json`)).json()).find((t) => t.type === 'page'); if (p) ws = new WebSocket(p.webSocketDebuggerUrl) } catch {}
    if (!ws) await sleep(200)
  }
  if (!ws) throw new Error('Chrome did not start')
  if (ws.readyState !== 1) await new Promise((r, j) => { ws.onopen = r; ws.onerror = j })
  let id = 0
  const pending = new Map()
  ws.onmessage = (m) => {
    const d = JSON.parse(m.data)
    if (d.id && pending.has(d.id)) { pending.get(d.id)(d); pending.delete(d.id) }
    if (d.method === 'Runtime.exceptionThrown') problems.push((d.params.exceptionDetails.exception?.description ?? d.params.exceptionDetails.text).split('\n')[0])
  }
  // Every request has a time limit, so a page that stops answering is a
  // failure that says where, not a hang.
  const send = (method, params = {}, secs = 30) => new Promise((r, j) => {
    const i = ++id
    const timer = setTimeout(() => { pending.delete(i); j(new Error(`no answer in ${secs}s to ${method} ${JSON.stringify(params).slice(0, 120)}`)) }, secs * 1000)
    pending.set(i, (d) => { clearTimeout(timer); r(d) })
    ws.send(JSON.stringify({ id: i, method, params }))
  })
  const js = async (expr) => {
    const r = await send('Runtime.evaluate', { expression: expr, awaitPromise: true, returnByValue: true })
    if (r.result?.exceptionDetails) throw new Error('page: ' + JSON.stringify(r.result.exceptionDetails).slice(0, 300))
    return r.result?.result?.value
  }
  let shots = 0
  const shot = async (name) => { const r = await send('Page.captureScreenshot', { format: 'png' }); writeFileSync(join(OUT, `${String(++shots).padStart(2, '0')}-${name}.png`), Buffer.from(r.result.data, 'base64')) }
  const S = 'window.go.main.Session'
  const view = async () => JSON.parse(await js(`${S}.State().then((v) => JSON.stringify(v))`))
  const reload = async () => { await send('Page.navigate', { url: URL }); for (let i = 0; i < 40; i++) { await sleep(250); if (await js(`!!window.go?.main?.Session && !!document.querySelector('.frame, .games')`).catch(() => false)) break } await sleep(400) }
  const GAME = join(LIB, 'games', 'slot-1', 'roster1.json')
  const disk = () => JSON.parse(readFileSync(GAME, 'utf8'))
  const sha = (p) => createHash('sha256').update(readFileSync(p)).digest('hex')
  const routes = { castle: [], tavern: ['tavern'], trading: ['trading'], inn: ['inn'], temple: ['temple'], edge: ['edge'], training: ['edge', 'training'], settings: ['edge', 'settings'] }
  const goto = async (place) => { await js(`${S}.Navigate('castle')`); for (const s of routes[place]) await js(`${S}.Navigate('${s}')`); await reload() }
  const rect = (sel) => js(`(() => { const e = document.querySelector(${JSON.stringify(sel)}); if (!e) return null; const r = e.getBoundingClientRect(); return { x: r.x, y: r.y, w: r.width, h: r.height, r: r.right, b: r.bottom } })()`)
  const A = '[data-area=display]', B = '[data-area=actions]', C = '[data-area=information]', D = '[data-area=party]'
  const clickText = async (label, within = 'body') => {
    const ok = await js(`(() => { const b = [...document.querySelectorAll(${JSON.stringify(within + ' button')})].find((b) => b.innerText.trim() === ${JSON.stringify(label)}); if (!b || b.disabled) return false; b.click(); return true })()`)
    if (!ok) throw new Error(`no button ${label} in ${within}`)
    await sleep(700)
  }
  const clickIn = async (sel, txt, btn) => {
    const ok = await js(`(() => { const r = [...document.querySelectorAll(${JSON.stringify(sel)})].find((e) => e.innerText.includes(${JSON.stringify(txt)})); const b = r?.querySelector(${JSON.stringify(btn)}); if (!b || b.disabled) return false; b.click(); return true })()`)
    if (!ok) throw new Error(`no ${btn} in ${sel} with ${txt}`)
    await sleep(700)
  }
  const buttons = (within) => js(`[...document.querySelectorAll(${JSON.stringify(within + ' button')})].map((b) => b.innerText.trim())`)
  const sheetName = () => js(`document.querySelector('${A} .sheet .name')?.firstChild?.textContent.trim() ?? ''`)
  const toggle = () => js(`[...document.querySelectorAll('${B} button')].find((b) => /Character Sheet$/.test(b.innerText.trim()))?.innerText.trim() ?? ''`)
  const listNames = (which) => js(`[...document.querySelectorAll('${A} ul.${which} li .what')].map((e) => e.firstChild.textContent.trim())`)
  const typeInto = async (sel, text) => { await js(`(() => { const e = document.querySelector(${JSON.stringify(sel)}); e.focus(); e.value = ''; e.dispatchEvent(new Event('input', { bubbles: true })) })()`); await send('Input.insertText', { text }); await sleep(200) }
  const esc = async () => {
    await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 })
    await send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 })
    await sleep(500)
  }
  const pageStill = () => js(`document.scrollingElement.scrollHeight <= innerHeight + 1 && document.scrollingElement.scrollWidth <= innerWidth + 1 && (() => { const b = document.querySelector('.frame .body'); return b.scrollHeight <= b.clientHeight + 1 })()`)
  const slots = () => js(`document.querySelectorAll('.party-area .partybar button.slot').length`)
  const clickSlot = async (i) => { await js(`document.querySelectorAll('.party-area .partybar button.slot')[${i}].click()`); await sleep(600) }

  await send('Page.enable'); await send('Runtime.enable')
  await send('Emulation.setDeviceMetricsOverride', { width: W, height: H, deviceScaleFactor: 1, mobile: false })
  await reload()
  let v = await view()
  if (v.screen === 'games') { await js(`${S}.ContinueGame(1)`); await reload(); v = await view() }
  check(v.game === 1 && v.roster.length === 8 && v.party.length === 6, `the scratch game is open: ${v.roster?.length} characters, a party of ${v.party?.length}`)
  await js(`${S}.SetPartyVisible(true)`)

  const suites = {
    // The four areas at every place in town; the party bar hidden and shown.
    async layout() {
      for (const place of Object.keys(routes)) {
        await goto(place)
        const [a, b, c, d] = [await rect(A), await rect(B), await rect(C), await rect(D)]
        if (!check(a && b && c && d, `${place}: the four areas are there`)) continue
        check(Math.abs(a.y - b.y) < 1 && b.x >= a.r && Math.abs(a.h - b.h) < 1, `${place}: display upper left, actions upper right, the same height`)
        check(c.y >= a.b && c.y >= b.b && d.y >= c.b - 0.5 && d.b <= H + 0.5, `${place}: information below both, the party bar at the bottom, all in the window`)
        check(await pageStill(), `${place}: the page itself does not scroll`)
        const sc = await js(`(() => { const c = document.querySelector('${C}'); const big = c.scrollHeight > c.clientHeight + 2; const y = document.querySelector('${A}').getBoundingClientRect().top; c.scrollTop = c.scrollHeight; const moved = c.scrollTop > 0; const y2 = document.querySelector('${A}').getBoundingClientRect().top; const by = document.querySelector('${B}').getBoundingClientRect().top; c.scrollTop = 0; return { big, moved, still: Math.abs(y - y2) < 0.5 && Math.abs(by - y) < 0.5 } })()`)
        check(!sc.big || (sc.moved && sc.still), `${place}: the information scrolls on its own; the display and actions stay`)
        check(await js(`(() => { const e = document.querySelector('${B}'); return getComputedStyle(e).overflowY === 'auto' && [...e.querySelectorAll('button')].every((b) => { const r = b.getBoundingClientRect(); return r.width > 0 && r.right <= innerWidth + 0.5 }) })()`), `${place}: every action is reachable`)
        const t = await toggle()
        check(place === 'trading' ? t === '' : t === 'Character Sheet', `${place}: the sheet toggle ${place === 'trading' ? 'is not offered at Boltac’s' : 'reads Character Sheet'}`)
        await shot(place)
      }
      await goto('training')
      const withParty = await rect(C)
      await clickText('Hide party')
      const without = await rect(C)
      check(!(await rect(D)) && without.h > withParty.h + 40, `hiding the party gives the information its room (${Math.round(withParty.h)} → ${Math.round(without.h)} px)`)
      await clickText('Show party')
      check(!!(await rect(D)), 'Show party brings the bar back')
    },

    // The embedded sheet; Worn & Wielded and the Pack on every member; Remove and Use, saved.
    async sheet() {
      await goto('castle')
      const art = await rect(A)
      await clickSlot(0)
      check(await toggle() === 'Hide Character Sheet' && await js(`!!document.querySelector('${A} .sheet') && !document.querySelector('${A} figure.scene')`), 'a party member’s sheet takes the artwork’s place')
      check(await sheetName() === v.party[0].name, `it is that member’s sheet (${await sheetName()})`)
      const box = await rect(A)
      check(Math.abs(box.y - art.y) < 1 && Math.abs(box.h - art.h) < 1 && Math.abs(box.w - art.w) < 1, 'in exactly the artwork’s area')
      check(await js(`!document.querySelector('.floating, [role=dialog].panel')`), 'no window opens')
      check(await js(`(() => { const p = document.querySelector('${A} .sheet .pages'); const h = document.querySelector('${A} .sheet header'); const y = h.getBoundingClientRect().top; p.scrollTop = p.scrollHeight; const ok = Math.abs(h.getBoundingClientRect().top - y) < 0.5; p.scrollTop = 0; return ok })()`), 'the sheet scrolls inside, its header kept in view')
      const text = (await js(`document.querySelector('${A} .sheet').innerText`)).toUpperCase()
      for (const w of ['Hit points', 'Armour class', 'Experience', 'Gold', 'Age', 'Equipment', 'Worn & Wielded', 'Pack', 'Spells']) check(text.includes(w.toUpperCase()), `the sheet shows ${w}`)
      await shot('sheet')
      await clickSlot(1)
      check(await sheetName() === v.party[1].name, `clicking another member shows theirs (${await sheetName()})`)
      await clickText('Hide Character Sheet', B)
      check(await toggle() === 'Character Sheet' && await js(`!!document.querySelector('${A} figure.scene')`), 'Hide Character Sheet brings the artwork back')
      await clickText('Character Sheet', B)
      check(await sheetName() === v.party[1].name, 'Character Sheet shows the same member again')
      await esc()
      check(await toggle() === 'Character Sheet' && (await view()).location === 'castle', 'Escape hides the sheet first and stays at the Castle')

      v = await view()
      let trip = null
      for (let i = 0; i < await slots(); i++) {
        await clickSlot(i)
        const found = v.roster.find((c) => c.index === v.partyBar[i].index)
        const ch = { ...found, items: found.items ?? [] } // Go's empty list is null
        const worn = await listNames('worn'), pack = await listNames('pack')
        const want = { worn: ch.items.filter((it) => it.equipped).map((it) => it.name), pack: ch.items.filter((it) => !it.equipped).map((it) => it.name) }
        check(JSON.stringify(worn) === JSON.stringify(want.worn) && JSON.stringify(pack) === JSON.stringify(want.pack), `${ch.name}: Worn & Wielded ${worn.length}, Pack ${pack.length}, each item once`)
        check(await js(`document.querySelector('${A} .sheet .count')?.innerText`) === `${ch.items.length} / 8`, `${ch.name}: carrying ${ch.items.length} / 8`)
        check(await js(`[...document.querySelectorAll('${A} ul.pack li')].filter((li) => /SCROLL|POTION/.test(li.innerText)).every((li) => !li.querySelector('button'))`), `${ch.name}: no Use on what cannot be equipped`)
        const k = ch.items.findIndex((it) => it.equipped && !it.unequipRefusal)
        if (!trip && k >= 0) trip = { i, ch, item: ch.items[k] }
      }
      if (!check(!!trip, 'a member with something to take off')) return
      const before = JSON.stringify(disk().roster.find((c) => c.Name === trip.ch.name))
      await clickSlot(trip.i)
      await clickIn(`${A} ul.worn li`, trip.item.name, 'button')
      let after = (await view()).roster.find((c) => c.index === trip.ch.index)
      check(!after.items.find((it) => it.name === trip.item.name).equipped && (await listNames('pack')).includes(trip.item.name), `Remove: ${trip.item.name} moves to the Pack`)
      check(JSON.stringify(disk().roster.find((c) => c.Name === trip.ch.name)) !== before, 'and it is saved')
      await clickIn(`${A} ul.pack li`, trip.item.name, 'button')
      after = (await view()).roster.find((c) => c.index === trip.ch.index)
      check(after.items.find((it) => it.name === trip.item.name).equipped && after.ac === trip.ch.ac, `Use: ${trip.item.name} is worn again, AC ${after.ac}`)
      check(JSON.stringify(disk().roster.find((c) => c.Name === trip.ch.name)) === before, 'the character is saved exactly as before')
      await esc()
      await goto('inn')
      check(!(await buttons(B)).includes('Character sheet'), 'the Inn has no sheet button of its own')
      await clickText('Character Sheet', B)
      check(await sheetName() === v.party[0].name, `at the Inn the sheet is the guest’s (${await sheetName()})`)
      await esc()
      await goto('temple')
      await clickText('Character Sheet', B)
      check(!!(await sheetName()), `the Temple shows a sheet (${await sheetName()})`)
      await esc()
    },

    // The illustrated places (ART-2A): the picture fills the window behind
    // the panels, cropped, never stretched; every control in view and
    // unobscured; Hide / Show panels and Escape; Hide party; the ways between
    // the two; in windows larger and smaller than this one; nothing saved.
    async scenes() {
      const before = sha(GAME)
      const art = (place) => js(`(() => { const d = document.querySelector('.backdrop'); const i = d?.querySelector('img'); if (!i) return null; const r = d.getBoundingClientRect(); const s = getComputedStyle(i); return { place: d.dataset.place, loaded: i.complete && i.naturalWidth > 0, nw: i.naturalWidth, nh: i.naturalHeight, fit: s.objectFit, covers: r.x <= 0 && r.y <= 0 && r.right >= innerWidth && r.bottom >= innerHeight, ir: (() => { const q = i.getBoundingClientRect(); return { w: q.width, h: q.height } })() } })()`)
      const loaded = async () => { for (let i = 0; i < 40; i++) { if (await js(`!!document.querySelector('.backdrop img')?.complete`)) break; await sleep(100) } }
      // Every button of the frame can be brought into view and is the
      // topmost thing where it is drawn, and no label is cut off.
      const controls = () => js(`(() => { const bad = []; for (const b of document.querySelectorAll('.frame button')) { b.scrollIntoView({ block: 'nearest', inline: 'nearest' }); const r = b.getBoundingClientRect(); const x = r.x + r.width / 2, y = r.y + r.height / 2; const top = document.elementFromPoint(x, y); if (r.width === 0 || r.right > innerWidth + 0.5 || r.bottom > innerHeight + 0.5 || r.x < -0.5 || r.y < -0.5 || !(top === b || b.contains(top))) bad.push(b.innerText.trim() || b.title); else if (b.scrollWidth > b.clientWidth + 1) bad.push(b.innerText.trim() + ' (cut off)') } for (const e of document.querySelectorAll('.frame [data-area], .frame .body')) e.scrollTop = 0; return bad })()`)
      // A panel's background: near-opaque, so its text reads over the picture.
      const opaque = (sel) => js(`[...document.querySelectorAll(${JSON.stringify(sel)})].map((e) => { const m = getComputedStyle(e).backgroundColor.match(/[\\d.]+/g).map(Number); return m.length < 4 ? 1 : m[3] })`)
      // The picture is seen at these points of the window: everything above
      // it there draws nothing (no background, no blur).
      const seen = (pts) => js(`${JSON.stringify(pts)}.every(([x, y]) => { const l = document.elementsFromPoint(x, y); const i = l.findIndex((e) => e.matches('.backdrop img')); return i >= 0 && l.slice(0, i).every((e) => { const c = getComputedStyle(e); return c.backgroundColor === 'rgba(0, 0, 0, 0)' && c.backgroundImage === 'none' && [undefined, '', 'none'].includes(c.backdropFilter) }) })`)
      const sized = async (w, h) => { await send('Emulation.setDeviceMetricsOverride', { width: w, height: h, deviceScaleFactor: 1, mobile: false }); await sleep(500) }
      for (const place of ['castle', 'edge']) {
        await goto(place); await loaded()
        let a = await art(place)
        if (!check(a && a.place === place && a.loaded && a.nw === 1920 && a.nh === 1080, `${place}: its picture is loaded (${a?.nw}x${a?.nh})`)) continue
        check(a.covers && a.fit === 'cover', `${place}: it fills the window, cropped to cover it`)
        // Seen, not only there: through the display, the picture is the
        // topmost thing at its centre and its corners.
        const d = await rect(A)
        check(await seen([[0.5, 0.5], [0.05, 0.05], [0.95, 0.95]].map(([u, v]) => [d.x + d.w * u, d.y + d.h * v])), `${place}: and it is seen through the display`)
        check(await js(`(() => { const f = document.querySelector('${A} figure.scene.view'); return !!f && !f.querySelector('img') && getComputedStyle(f).backgroundColor === 'rgba(0, 0, 0, 0)' && !!f.getAttribute('aria-label') })()`), `${place}: the display is a described opening onto it`)
        const alphas = await opaque(`.frame .head, ${B} .actions, ${B} nav, ${C} > section, ${D}`)
        check(alphas.length >= 4 && alphas.every((x) => x >= 0.85), `${place}: the panels are near-opaque (${alphas.map((x) => x.toFixed(2)).join(' ')})`)
        let bad = await controls()
        check(bad.length === 0, `${place}: every control in view and uncovered${bad.length ? ': ' + bad.join(', ') : ''}`)
        check(await pageStill(), `${place}: the page itself does not scroll`)
        await shot(`${place}-panels`)

        await clickText('Hide panels')
        const only = await buttons('.frame')
        check(!(await rect(A)) && !(await rect(B)) && !(await rect(C)) && !(await rect(D)) && !(await rect('.frame .head')), `${place}: Hide panels leaves the picture alone`)
        check(only.length === 1 && /^Show panels/.test(only[0]) && await js(`document.activeElement?.innerText.startsWith('Show panels')`), `${place}: with one way back, Show panels, which has the focus`)
        a = await art(place)
        check(a.covers && (await controls()).length === 0 && await seen([[0.3, 0.5], [0.5, 0.9], [0.9, 0.6]].map(([u, v]) => [W * u, H * v])), `${place}: the picture is seen across the window; Show panels is in view`)
        await shot(`${place}-hidden`)
        await esc()
        check(!!(await rect(A)) && !!(await rect(B)) && (await view()).location === place, `${place}: Escape brings the panels back and stays here`)
        await clickText('Hide panels')
        await js(`[...document.querySelectorAll('.frame button')].find((b) => b.innerText.startsWith('Show panels')).click()`); await sleep(500)
        check(!!(await rect(B)) && await js(`document.activeElement?.innerText.trim() === 'Hide panels'`), `${place}: Show panels brings them back, the focus on Hide panels`)

        const withParty = [await rect(B), await rect(C)]
        await clickText('Hide party')
        const without = [await rect(B), await rect(C)]
        check(!(await rect(D)) && without[0].h > withParty[0].h + 40 && without[1].b > withParty[1].b + 40 && (await controls()).length === 0, `${place}: Hide party gives its room to the panels (actions ${Math.round(withParty[0].h)} → ${Math.round(without[0].h)} px high)`)
        await shot(`${place}-no-party`)
        await clickText('Show party')
        check(!!(await rect(D)), `${place}: Show party brings the bar back`)

        // Larger and smaller windows: the 14- and 16-inch MacBook Pro's
        // full screens, a large display, and the smallest window.
        for (const [w, h] of [[1512, 982], [1728, 1117], [2560, 1440], [760, 560]]) {
          await sized(w, h)
          a = await art(place); bad = await controls()
          const ratio = a.ir.w / a.ir.h
          check(a.covers && Math.abs(ratio - w / h) < 0.01 && bad.length === 0 && await pageStill(), `${place} at ${w}x${h}: the picture fills it, every control in view and uncovered${bad.length ? ': ' + bad.join(', ') : ''}`)
          if (w >= 1512 && w <= 1728) await shot(`${place}-${w}x${h}`)
        }
        await sized(W, H)
      }
      // The ways between them, through the buttons.
      await goto('castle')
      await clickText('The Edge of Town', B)
      check((await view()).location === 'edge' && (await art('edge'))?.place === 'edge', 'The Edge of Town: outside the walls, its own picture')
      await clickText('← Castle Esc', '.frame .head')
      check((await view()).location === 'castle' && (await art('castle'))?.place === 'castle', '← Castle: back in the courtyard, its own picture')
      await goto('tavern')
      check(!(await js(`!!document.querySelector('.backdrop')`)) && !!(await js(`document.querySelector('${A} figure.scene:not(.view)')`)) && !(await buttons('.frame .head')).includes('Hide panels'), 'a place without artwork keeps its themed panel, and no Hide panels')
      await goto('castle')
      await clickSlot(0)
      check(await js(`(() => { const s = document.querySelector('${A} .sheet'); return !!s && getComputedStyle(s).backgroundColor !== 'rgba(0, 0, 0, 0)' })()`), 'a character sheet stands over the picture in the display')
      await clickText('Hide panels')
      check(!(await rect(`${A} .sheet`)), 'Hide panels hides the sheet with the rest')
      await esc()
      check(!!(await rect(`${A} .sheet`)), 'and Show panels brings it back')
      await esc()
      check(sha(GAME) === before, 'the game is saved exactly as before: the pictures and panels change nothing')
    },

    // Creation: decisions in the Actions; points and class on one screen; Discard, then Keep.
    async creation() {
      await goto('training')
      const before = sha(GAME)
      const rosterSize = (await view()).roster.length
      const order = [1, 3, 0, 2, 4, 5]
      const spend = async () => {
        for (const i of order) for (let k = 0; k < 30; k++) {
          const c = (await view()).creation
          if (!c.points || !c.canAdd[i]) break
          await js(`document.querySelectorAll('${C} button.step.up')[${i}].click()`); await sleep(200)
        }
      }
      const begin = async () => {
        await typeInto('#newname', 'Live')
        await clickText('Create', B)
        await clickText('Yes', B)
        await clickText('Continue', B)
        await clickText('B) ELF', B)
        await clickText('B) NEUTRAL', B)
      }
      await typeInto('#newname', 'Live')
      await clickText('Create', B)
      check((await buttons(B)).includes('Yes') && !(await buttons(C)).includes('Yes'), 'confirming: Yes / No in the Actions')
      await clickText('No', B)
      await begin()
      let c = (await view()).creation
      check(c.step === 'stats' && (await js(`document.querySelector('${B} .next').innerText`)).startsWith('Spend the bonus points'), 'points: the Actions say to spend them')
      check(await js(`document.querySelectorAll('${C} table.guide tr.class-row').length`) === 8 && await js(`document.querySelectorAll('${C} tr.pickable').length`) === 0, 'the whole class table, nothing choosable yet')
      check(!(await buttons(C)).some((b) => /Done|choose class|Keep|Discard|Continue/.test(b)), 'no workflow buttons below')
      await spend()
      c = (await view()).creation
      check(c.points === 0 && c.step === 'stats', 'every point spent, on the same screen')
      check((await js(`document.querySelector('${B} .next').innerText`)) === 'Choose a highlighted class.', 'the Actions say: Choose a highlighted class.')
      const rows = await js(`[...document.querySelectorAll('${C} table.guide tr.class-row')].map((r) => ({ pick: r.classList.contains('pickable'), btn: !!r.querySelector('button.pick-class'), note: r.cells[r.cells.length - 1].innerText }))`)
      check(rows.every((r, i) => r.pick === c.classOk[i] && r.btn === r.pick), 'exactly the classes Go allows are choosable')
      check(rows.filter((r) => !r.pick).every((r) => r.note.length > 0), 'the others stay visible with their reasons')
      await shot('choose-class')
      await js(`document.querySelectorAll('${C} button.step.down')[1].click()`); await sleep(400)
      check((await view()).creation.points === 1 && await js(`document.querySelectorAll('${C} tr.pickable').length`) === 0, 'a point taken back: nothing choosable')
      await js(`document.querySelectorAll('${C} button.step.up')[1].click()`); await sleep(400)
      await clickIn(`${C} tr.pickable`, 'MAGE', 'button.pick-class')
      c = (await view()).creation
      check(c.step === 'confirm' && c.class === 'MAGE', 'choosing MAGE goes straight to the review')
      check((await buttons(B)).includes('Keep and save') && (await buttons(B)).includes('Discard'), 'review: Keep and save / Discard in the Actions')
      await clickText('Discard', B)
      v = await view()
      check(v.screen === 'roster' && v.location === 'training', `Discard: still at the Training Grounds (${v.screen}, ${v.location})`)
      check(v.roster.length === rosterSize, `Discard: nothing kept (roster ${v.roster.length}, was ${rosterSize})`)
      check(sha(GAME) === before, 'Discard: nothing saved')
      await begin()
      await spend()
      const stats = (await view()).creation.stats
      await clickIn(`${C} tr.pickable`, 'MAGE', 'button.pick-class')
      await clickText('Keep and save', B)
      v = await view()
      const made = v.roster.find((ch) => ch.name === 'LIVE')
      check(made && made.class === 'MAGE' && JSON.stringify(made.stats) === JSON.stringify(stats) && made.hp > 0, 'Keep and save: a MAGE with the stats chosen')
      check(disk().roster.some((ch) => ch.Name === 'LIVE'), 'and it is saved')
      await js(`[...document.querySelectorAll('${C} tr.pick')].find((r) => r.innerText.includes('LIVE')).click()`); await sleep(600)
      check(await sheetName() === 'LIVE' && (await buttons(`${A} .sheet`)).includes('Rename'), 'its roster row shows its sheet, with Rename')
    },
  }

  for (const name of SUITES) {
    if (!suites[name]) { check(false, `no live suite ${name}`); continue }
    const t = Date.now()
    try { await suites[name]() } catch (e) { check(false, `${name}: ${e.message}`) }
    console.log(`live: suite ${name} at ${W}x${H} in ${((Date.now() - t) / 1000).toFixed(0)}s`)
  }
  const pageErrors = problems.filter((p) => !/reading 'nodes'/.test(p))
  check(pageErrors.length === 0, `no errors in the page${pageErrors.length ? ': ' + pageErrors.slice(0, 3).join(' | ') : ''}`)
}

let crashed = null
try { await main() } catch (e) { crashed = e; check(false, `error: ${e.message}`) }
const left = await stopAll()
check(left.length === 0, `its own processes are stopped${left.length ? ' (still running: ' + left.join(' ') + ')' : ''}`)
console.log(`live: ${fails.length ? 'FAIL' : 'PASS'} at ${W}x${H}: ${passed.length} checks passed, ${fails.length} failed${crashed ? ' (stopped by an error)' : ''}`)
process.exit(fails.length ? 1 : 0)
