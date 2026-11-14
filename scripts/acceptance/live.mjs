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
//        [--size 980x692] [--suites layout,sheet,scenes,creation,combat,nav] [--wails PATH] [--chrome PATH]
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
  // The sheet's toggle, in the top bar (NAV-1).
  const toggle = () => js(`[...document.querySelectorAll('.frame .head button')].find((b) => /^(Show|Hide) sheet$/.test(b.innerText.trim()))?.innerText.trim() ?? ''`)
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

  // The illustrated places (ART-2A, ART-2B).
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
  // The mouse and keys, as a player would use them.
  const mouse = (type, x, y) => send('Input.dispatchMouseEvent', { type, x, y, button: 'left', buttons: type === 'mouseReleased' ? 0 : 1, clickCount: 1 })
  const dragBy = async (x, y, dx, dy) => {
    await mouse('mousePressed', x, y)
    for (let i = 1; i <= 6; i++) await mouse('mouseMoved', x + (dx * i) / 6, y + (dy * i) / 6)
    await mouse('mouseReleased', x + dx, y + dy); await sleep(300)
  }
  const keys = { ArrowLeft: 37, ArrowUp: 38, ArrowRight: 39, ArrowDown: 40 }
  const press = async (key, shift = false) => {
    for (const type of ['rawKeyDown', 'keyUp']) await send('Input.dispatchKeyEvent', { type, key, code: key, windowsVirtualKeyCode: keys[key], modifiers: shift ? 8 : 0 })
    await sleep(200)
  }
  const STORY = '.frame aside.story'
  // The appearance (ART-2C), set as Settings sets it, then shown at place.
  const LOOKS = { panels: ['slate', 'dark', 'light'], story: ['parchment', 'dark', 'slate'], artwork: ['full', 'subdued', 'hidden'] }
  const setLook = async (l, place) => { const v = JSON.parse(await js(`${S}.SetAppearance(${JSON.stringify(l.panels)}, ${JSON.stringify(l.story)}, ${JSON.stringify(l.artwork)}).then((v) => JSON.stringify(v))`)); if (v.error) throw new Error(v.error); await goto(place); await loaded() }
  // What the page shows of it.
  const shownLook = () => js(`(() => { const f = document.querySelector('.frame'), b = document.querySelector('.backdrop'), i = b?.querySelector('img'); return { panels: f?.dataset.appearance ?? null, story: document.querySelector('${STORY}')?.dataset.appearance ?? null, artwork: b?.dataset.artwork ?? null, img: !!i, filter: i ? getComputedStyle(i).filter : null } })()`)
  const SETTINGS_FILE = join(LIB, 'gui-settings.json')
  const fileLook = () => { try { const f = JSON.parse(readFileSync(SETTINGS_FILE, 'utf8')); return { panels: f.panels, story: f.story, artwork: f.artwork } } catch { return null } }
  // Where everything in the frame is: the same in every appearance.
  const layoutNow = () => js(`[...document.querySelectorAll('.frame .head, .frame [data-area], .frame button, ${STORY}, .frame p, .frame h2, .frame h3')].map((e) => { const r = e.getBoundingClientRect(); return [r.x, r.y, r.width, r.height].map(Math.round).join(',') }).join(' ')`)
  // The lowest contrast of the frame's text against what is behind it
  // (WCAG 2), with the picture under the panels taken at its darkest and
  // its lightest: every fill and opacity between the text and the picture
  // composited over black and over white. Disabled controls are exempt.
  // Large text (24px, or 18.66px bold) needs 3:1, the rest 4.5:1.
  const contrast = () => js(`(() => {
    const rgba = (s) => { const m = s.match(/[\\d.]+/g).map(Number); return [m[0], m[1], m[2], m.length > 3 ? m[3] : 1] }
    const over = (t, b) => [0, 1, 2].map((i) => t[i] * t[3] + b[i] * (1 - t[3])).concat(1)
    const lin = (c) => { c /= 255; return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4 }
    const lum = (c) => 0.2126 * lin(c[0]) + 0.7152 * lin(c[1]) + 0.0722 * lin(c[2])
    const ratio = (a, b) => { const [x, y] = [lum(a), lum(b)].sort((p, q) => q - p); return (x + 0.05) / (y + 0.05) }
    const worst = []
    for (const el of document.querySelectorAll('.frame *')) {
      if (![...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim())) continue
      const r = el.getBoundingClientRect(); if (!r.width || !r.height || el.closest('button:disabled, select:disabled, .backdrop')) continue
      const chain = []; let op = 1
      for (let e = el; e && e !== document.documentElement; e = e.parentElement) { const c = getComputedStyle(e); op *= Number(c.opacity); const f = rgba(c.backgroundColor); if (f[3] > 0) chain.push(f) }
      const cs = getComputedStyle(el); const fg = rgba(cs.color); fg[3] *= op
      const size = parseFloat(cs.fontSize), bold = Number(cs.fontWeight) >= 700
      const need = size >= 24 || (size >= 18.66 && bold) ? 3 : 4.5
      let min = Infinity
      for (const base of [[0, 0, 0, 1], [255, 255, 255, 1]]) { const bg = chain.reverse().reduce((acc, f) => over(f, acc), base); chain.reverse(); min = Math.min(min, ratio(over(fg, bg), bg)) }
      if (min < need) worst.push(el.tagName.toLowerCase() + '.' + [...el.classList].join('.') + ' "' + el.textContent.trim().slice(0, 24) + '" ' + min.toFixed(2))
    }
    return worst
  })()`)

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
        check(place === 'trading' ? t === '' : t === 'Show sheet', `${place}: the sheet toggle ${place === 'trading' ? 'is not offered at Boltac’s' : 'reads Show sheet, in the top bar'}`)
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
      check(await toggle() === 'Hide sheet' && await js(`!!document.querySelector('${A} .sheet') && !document.querySelector('${A} figure.scene')`), 'a party member’s sheet takes the artwork’s place')
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
      await clickText('Hide sheet', '.frame .head')
      check(await toggle() === 'Show sheet' && await js(`!!document.querySelector('${A} figure.scene')`), 'Hide sheet brings the artwork back')
      await clickText('Show sheet', '.frame .head')
      check(await sheetName() === v.party[1].name, 'Show sheet shows the same member again')
      await esc()
      check(await toggle() === 'Show sheet' && (await view()).location === 'castle', 'Escape hides the sheet first and stays at the Castle')

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
      check(!(await buttons(B)).some((b) => /sheet/i.test(b)), 'the Inn has no sheet button of its own, in its actions')
      await clickText('Show sheet', '.frame .head')
      check(await sheetName() === v.party[0].name, `at the Inn the sheet is the guest’s (${await sheetName()})`)
      await esc()
      await goto('temple')
      await clickText('Show sheet', '.frame .head')
      check(!!(await sheetName()), `the Temple shows a sheet (${await sheetName()})`)
      await esc()
    },

    // The illustrated places (ART-2A): the picture fills the window behind
    // the panels, cropped, never stretched; every control in view and
    // unobscured; Hide / Show panels and Escape; Hide party; the ways between
    // the two; in windows larger and smaller than this one; nothing saved.
    async scenes() {
      const before = sha(GAME)
      // A library that has never chosen: Slate panels, a Parchment story
      // box, the whole picture (ART-2C).
      await goto('castle'); await loaded()
      const first = await shownLook()
      check(fileLook()?.panels === undefined && first.panels === 'slate' && first.story === 'parchment' && first.artwork === 'full' && first.img && first.filter === 'none', `the defaults: Slate panels, a Parchment story box, the whole picture (${JSON.stringify(first)})`)
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
        // Its panels (NAV-1): the Castle's are the top bar, its
        // destinations and the party; the Edge's the top bar, its actions,
        // its information and the party.
        const panels = place === 'castle' ? ['.frame .head', `${B} nav`, D] : ['.frame .head', `${B} .actions`, `${C} > section`, D]
        const alphas = await opaque(panels.join(', '))
        check(alphas.length === panels.length && alphas.every((x) => x >= 0.85), `${place}: the panels are near-opaque (${alphas.map((x) => x.toFixed(2)).join(' ')})`)
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

      // The story box (ART-2B): open at first, in the lower left of the
      // display; moved and resized within the window, with the mouse and
      // the keys; closed, and not opened again by going elsewhere; the
      // narrative in it, and what the player needs to play still below.
      await goto('castle')
      let st = await rect(STORY), disp = await rect(A)
      check(!!st && st.x >= disp.x && st.x < disp.x + 40 && Math.abs(st.b - (disp.b - 12)) < 2 && st.y >= disp.y, `castle: the story box is open, in the lower left of the display (${st && Math.round(st.x)},${st && Math.round(st.y)} ${st && Math.round(st.w)}x${st && Math.round(st.h)})`)
      check(await js(`document.querySelector('${STORY} .text').innerText.includes("The party is formed at Gilgamesh's Tavern")`) && !(await js(`document.querySelector('${C}').innerText.includes('The party is formed')`)), 'castle: the description is in it, and no longer below')
      let bad = await controls()
      check(bad.length === 0, `castle: it covers no control${bad.length ? ': ' + bad.join(', ') : ''}`)
      let bar = await rect(`${STORY} .bar`)
      await dragBy(bar.x + 30, bar.y + bar.h / 2, 120, -80)
      let moved = await rect(STORY)
      check(Math.abs(moved.x - st.x - 120) < 2 && Math.abs(moved.y - st.y + 80) < 2 && Math.abs(moved.w - st.w) < 1, 'dragging its title moves it')
      bar = await rect(`${STORY} .bar`)
      await dragBy(bar.x + 30, bar.y + bar.h / 2, 5000, 5000)
      moved = await rect(STORY)
      check(Math.abs(moved.r - (W - 8)) < 1.5 && Math.abs(moved.b - (H - 8)) < 1.5, 'dragged past the window, it stops at its edge')
      bar = await rect(`${STORY} .bar`)
      await dragBy(bar.x + 30, bar.y + bar.h / 2, -5000, -5000)
      moved = await rect(STORY)
      check(Math.abs(moved.x - 8) < 1.5 && Math.abs(moved.y - 8) < 1.5, 'and at the other')
      let g = await rect(`${STORY} .grip`)
      await dragBy(g.x + 8, g.y + 8, 80, 60)
      let sz = await rect(STORY)
      check(Math.abs(sz.w - moved.w - 80) < 2 && Math.abs(sz.h - moved.h - 60) < 2 && Math.abs(sz.x - moved.x) < 1, `dragging its corner resizes it (${Math.round(moved.w)}x${Math.round(moved.h)} → ${Math.round(sz.w)}x${Math.round(sz.h)})`)
      g = await rect(`${STORY} .grip`)
      await dragBy(g.x + 8, g.y + 8, 5000, 5000)
      sz = await rect(STORY)
      check(Math.abs(sz.r - (W - 8)) < 1.5 && Math.abs(sz.b - (H - 8)) < 1.5, 'no larger than the window')
      g = await rect(`${STORY} .grip`)
      await dragBy(g.x + 8, g.y + 8, -5000, -5000)
      sz = await rect(STORY)
      check(Math.abs(sz.w - 220) < 1.5 && Math.abs(sz.h - 90) < 1.5 && await js(`(() => { const t = document.querySelector('${STORY} .text'); return getComputedStyle(t).overflowY === 'auto' && t.scrollHeight > t.clientHeight })()`), `no smaller than 220x90, its text scrolling (${Math.round(sz.w)}x${Math.round(sz.h)})`)
      await js(`document.querySelector('${STORY} .bar').focus()`)
      const k0 = await rect(STORY)
      await press('ArrowRight'); await press('ArrowDown')
      await press('ArrowRight', true); await press('ArrowDown', true)
      const k1 = await rect(STORY)
      check(Math.abs(k1.x - k0.x - 16) < 1 && Math.abs(k1.y - k0.y - 16) < 1 && Math.abs(k1.w - k0.w - 16) < 1 && Math.abs(k1.h - k0.h - 16) < 1, 'with its title focused, the arrow keys move it and Shift with them resizes it')
      // Left somewhere of the player's choosing, then closed.
      bar = await rect(`${STORY} .bar`)
      await dragBy(bar.x + 30, bar.y + bar.h / 2, 40, 200)
      const left = await rect(STORY)
      await js(`document.querySelector('${STORY} .close').click()`); await sleep(400)
      check(!(await rect(STORY)) && await js(`(() => { const b = [...document.querySelectorAll('.frame .head button')].find((b) => b.innerText.trim() === 'Story'); return !!b && b.getAttribute('aria-pressed') === 'false' && document.activeElement === b })()`), '× closes it; Story, not pressed, has the focus')
      await clickText('The Edge of Town', B)
      check((await view()).location === 'edge' && !(await rect(STORY)), 'going to the Edge of Town does not open it')
      const info = await js(`document.querySelector('${C}').innerText`)
      check(/Saving\.[\s\S]*saved after every/.test(info), 'the Edge’s saving note is below, the story box closed')
      await clickText('Story', '.frame .head')
      st = await rect(STORY)
      check(!!st && await js(`document.querySelector('${STORY} .text').innerText.trim()`) === 'The maze. The party sets out from here.', 'Story opens it, with the Edge’s narrative')
      check(Math.abs(st.x - left.x) < 1 && Math.abs(st.y - left.y) < 1 && Math.abs(st.w - left.w) < 1 && Math.abs(st.h - left.h) < 1, 'where the player left it')
      check(/Saving\./.test(await js(`document.querySelector('${C}').innerText`)), 'and the note stays below')
      await clickText('Hide panels')
      check(!(await rect(STORY)), 'Hide panels hides it')
      await esc()
      check(!!(await rect(STORY)), 'Show panels brings it back')
      await clickText('← Castle Esc', '.frame .head')
      st = await rect(STORY)
      check((await view()).location === 'castle' && !!st && Math.abs(st.x - left.x) < 1 && Math.abs(st.y - left.y) < 1 && Math.abs(st.w - left.w) < 1, 'open, it stays open at the Castle, in the same place')

      // Settings (ART-2C): the three choices, taking effect at once, kept
      // in the window's settings file and nowhere in the game; the story
      // box as it was, open and where it was left, through Settings.
      await clickText('The Edge of Town', B)
      await clickText('Settings', B)
      const radios = () => js(`Object.fromEntries(['panels', 'story', 'artwork'].map((k) => [k, [...document.querySelectorAll('input[name=look-' + k + ']')].map((r) => (r.checked ? '*' : '') + r.value).join(' ')]))`)
      let r0 = await radios()
      check(r0.panels === '*slate dark light' && r0.story === '*parchment dark slate' && r0.artwork === '*full subdued hidden', `Settings offers each part, the defaults chosen (${JSON.stringify(r0)})`)
      check(!(await js(`!!document.querySelector('[data-appearance]') || !!document.querySelector('.backdrop')`)), 'Settings itself, a place without artwork, is as it was')
      await shot('settings-appearance')
      const clickRadio = async (k, id) => {
        const p = await js(`(() => { const r = document.querySelector('input[name=look-${k}][value=${id}]').closest('label'); r.scrollIntoView({ block: 'nearest' }); const q = r.getBoundingClientRect(); return { x: q.x + 12, y: q.y + q.height / 2 } })()`)
        await mouse('mousePressed', p.x, p.y); await mouse('mouseReleased', p.x, p.y); await sleep(500)
      }
      await clickRadio('panels', 'light'); await clickRadio('story', 'dark'); await clickRadio('artwork', 'subdued')
      r0 = await radios()
      check(r0.panels.includes('*light') && r0.story.includes('*dark') && r0.artwork.includes('*subdued') && JSON.stringify(fileLook()) === JSON.stringify({ panels: 'light', story: 'dark', artwork: 'subdued' }), `clicked, each takes effect and is kept in gui-settings.json (${JSON.stringify(fileLook())})`)
      check(!(await js(`document.querySelector('input[name=look-panels]').closest('fieldset').disabled`)), 'and the choices stay open to change')
      await clickText('← Castle Esc', '.frame .head'); await loaded()
      let sl = await shownLook()
      st = await rect(STORY)
      check(sl.panels === 'light' && sl.story === 'dark' && sl.artwork === 'subdued' && sl.img && sl.filter !== 'none', `back at the Castle: Light panels, a Dark story box, the picture subdued (${sl.filter})`)
      check(!!st && Math.abs(st.x - left.x) < 1 && Math.abs(st.y - left.y) < 1, 'the story box open, where it was left')

      // Another game, and back: the same appearance (it is the window's).
      await js(`${S}.Navigate('edge')`)
      let gv = JSON.parse(await js(`${S}.ChangeGame().then((v) => JSON.stringify(v))`))
      if (gv.screen === 'games') gv = JSON.parse(await js(`${S}.NewGame(2).then((v) => JSON.stringify(v))`))
      if (gv.screen === 'games') gv = JSON.parse(await js(`${S}.ContinueGame(2).then((v) => JSON.stringify(v))`)) // a library used before
      await reload(); await loaded()
      sl = await shownLook()
      check(gv.game === 2 && gv.location === 'castle' && sl.panels === 'light' && sl.story === 'dark' && sl.artwork === 'subdued', `in another game (${gv.game}, ${gv.location}): the same appearance`)
      await js(`${S}.Navigate('edge')`)
      await js(`${S}.ChangeGame()`)
      gv = JSON.parse(await js(`${S}.ContinueGame(1).then((v) => JSON.stringify(v))`))
      await goto('castle'); await loaded()
      sl = await shownLook()
      check(gv.game === 1 && sl.panels === 'light' && sl.story === 'dark' && sl.artwork === 'subdued' && sha(GAME) === before, 'back in the first game: the same, and the game unchanged')

      // Every combination (ART-2C): the panels' 3 appearances, the story
      // box's 3 and the picture's 3, at both places. Only colours change, so
      // everything is where it is in the defaults (the picture hidden: less
      // Hide panels); the text reads on every panel and in the story box;
      // every control is in view and uncovered, and none disabled by it.
      const combos = []
      for (const panels of LOOKS.panels) for (const story of LOOKS.story) for (const artwork of LOOKS.artwork) combos.push({ panels, story, artwork })
      for (const place of ['castle', 'edge']) {
        await setLook({ panels: 'slate', story: 'parchment', artwork: 'full' }, place)
        const base = await layoutNow(), enabled = await js(`[...document.querySelectorAll('.frame button')].filter((b) => !b.disabled).length`)
        await setLook({ panels: 'slate', story: 'parchment', artwork: 'hidden' }, place)
        const baseHidden = await layoutNow()
        const issues = []
        for (const l of combos) {
          await setLook(l, place)
          const name = `${l.panels}/${l.story}/${l.artwork}`
          const sl = await shownLook()
          if (sl.panels !== l.panels || sl.story !== l.story || sl.artwork !== l.artwork) issues.push(`${name}: shows ${JSON.stringify(sl)}`)
          const pic = l.artwork === 'hidden' ? !sl.img : sl.img && (l.artwork === 'full' ? sl.filter === 'none' : /brightness/.test(sl.filter))
          if (!pic) issues.push(`${name}: the picture is not ${l.artwork} (${sl.img}, ${sl.filter})`)
          const head = await buttons('.frame .head')
          if (head.includes('Hide panels') === (l.artwork === 'hidden') || !head.includes('Story')) issues.push(`${name}: head ${head.join('|')}`)
          if (await layoutNow() !== (l.artwork === 'hidden' ? baseHidden : base)) issues.push(`${name}: the layout moved`)
          const al = await opaque(`.frame .head, ${B} .actions, ${B} nav, ${C} > section, ${D}, ${STORY}`)
          if (!al.every((x) => x >= 0.9 && x <= 0.95)) issues.push(`${name}: opacity ${al.join(' ')}`)
          const low = await contrast()
          if (low.length) issues.push(`${name}: contrast ${low.slice(0, 3).join('; ')}`)
          const bad = await controls()
          if (bad.length) issues.push(`${name}: covered ${bad.join(', ')}`)
          const on = await js(`[...document.querySelectorAll('.frame button')].filter((b) => !b.disabled).length`)
          if (on !== enabled - (l.artwork === 'hidden' ? 1 : 0)) issues.push(`${name}: ${on} controls enabled, ${enabled} in the defaults`)
          if (!(await pageStill())) issues.push(`${name}: the page scrolls`)
        }
        check(issues.length === 0, `${place}: all ${combos.length} combinations shown as chosen, the layout unchanged, 90–95% opaque, every text at WCAG AA, every control in view, uncovered and enabled${issues.length ? ': ' + issues.slice(0, 6).join(' | ') : ''}`)
      }
      // The picture hidden: the place's tint alone; the display describes
      // nothing; Escape still goes Back.
      await setLook({ panels: 'dark', story: 'slate', artwork: 'hidden' }, 'castle')
      check(await js(`(() => { const b = document.querySelector('.backdrop'); return !b.querySelector('img') && !b.querySelector('.emblem') && !!b.querySelector('.fallback.plain') && document.querySelector('${A} figure.scene.view').getAttribute('aria-hidden') === 'true' && !document.querySelector('${A} [role=img]') })()`), 'the picture hidden: a plain backdrop, and the display describes no picture')
      await clickText('The Edge of Town', B); await esc()
      check((await view()).location === 'castle', 'Escape still goes Back')
      await setLook({ panels: 'slate', story: 'parchment', artwork: 'full' }, 'castle')
      await goto('tavern')
      check(await js(`!document.querySelector('[data-appearance]') && !document.querySelector('.backdrop') && !!document.querySelector('${A} figure.scene:not(.view) .fallback .emblem')`), 'a place without artwork is as it was')

      check(sha(GAME) === before, 'the game is saved exactly as before: the pictures and panels change nothing')
    },

    // The appearances for review (ART-2C): the Castle and the Edge of Town
    // with the defaults (Slate panels, a Parchment story box) and a
    // contrasting combination (Light panels, a Dark story box, the picture
    // subdued), at the compact window, 1440x900 and the MacBook Pro's full
    // screens, the story box open and, at the first two, closed; and the
    // picture hidden, once. Screenshots only, named for what they show;
    // the checks are the scenes suite's. Leaves the defaults.
    async looks() {
      const shots = [{ panels: 'slate', story: 'parchment', artwork: 'full' }, { panels: 'light', story: 'dark', artwork: 'subdued' }]
      for (const [w, h] of [[980, 692], [1440, 900], [1512, 982], [1728, 1117]]) {
        await sized(w, h)
        for (const place of ['castle', 'edge']) {
          for (const l of shots) {
            await setLook(l, place)
            const name = `${place}-${l.panels}-${l.story}-${l.artwork}-${w}x${h}`
            await shot(`${name}-story`)
            if (w <= 1440) {
              await js(`document.querySelector('${STORY} .close').click()`); await sleep(300)
              await shot(`${name}-closed`)
              await clickText('Story', '.frame .head')
            }
          }
        }
      }
      await sized(1440, 900)
      for (const place of ['castle', 'edge']) { await setLook({ panels: 'dark', story: 'slate', artwork: 'hidden' }, place); await shot(`${place}-dark-slate-hidden-1440x900-story`) }
      await setLook({ panels: 'slate', story: 'parchment', artwork: 'full' }, 'castle')
      await sized(W, H)
      check(true, 'looks: the screenshots are taken')
    },

    // A restart (ART-2C; by hand, twice on one library): Istary opens with
    // the appearance gui-settings.json keeps, the defaults if it keeps none;
    // then another is chosen, for the next run to find.
    async restarted() {
      const kept = fileLook()
      await goto('castle'); await loaded()
      const sl = await shownLook()
      const want = kept?.panels ? kept : { panels: 'slate', story: 'parchment', artwork: 'full' }
      check(sl.panels === want.panels && sl.story === want.story && sl.artwork === want.artwork, `opened with ${kept?.panels ? 'the appearance kept' : 'the defaults, none kept'} (${JSON.stringify(want)}; shown ${JSON.stringify(sl)})`)
      const i = (LOOKS.panels.indexOf(want.panels) + 1) % 3
      const next = { panels: LOOKS.panels[i], story: LOOKS.story[(i + 1) % 3], artwork: LOOKS.artwork[i] }
      await setLook(next, 'castle')
      check(JSON.stringify(fileLook()) === JSON.stringify(next), `chosen for the next run: ${JSON.stringify(next)}`)
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

    // COMBAT-1: the spell picker with its descriptions and reasons, Party
    // Retreat and its outcome, and the character sheet's spells, in a
    // practice encounter (which writes nothing).
    combat: async () => {
      const before = sha(GAME)
      const C1 = '.combat .commands'
      const cv = async () => (await view()).combat
      // On to order entry, through the bound methods, then the page.
      const toOrders = async () => {
        for (let i = 0; i < 60; i++) {
          const c = await cv()
          if (!c || c.next === 'orders') break
          const m = { start: 'CombatStart', advance: 'CombatAdvance', 'end-round': 'CombatEndRound', confirm: 'CombatCommit' }[c.next]
          if (c.next === 'friendly') await js(`${S}.CombatFriendly(true)`)
          else if (m) await js(`${S}.${m}()`)
          else break
        }
        await reload()
        return cv()
      }
      const practice = async () => {
        await goto('edge')
        await js(`${S}.PracticeEncounter(false)`)
        await reload()
        return toOrders()
      }
      const slotOf = (c, cls) => c.party.find((m) => m.class.endsWith(cls.slice(0, 3)))?.slot // e.g. N-MAG
      // The combat screen has no frame: wait for its own controls.
      const waitFor = async (sel) => { for (let i = 0; i < 40; i++) { if (await js(`!!document.querySelector(${JSON.stringify(sel)})`)) return true; await sleep(250) } return false }
      const select = async (slot) => { await js(`${S}.CombatSelect(${slot})`); await reload(); await waitFor(`${C1} button.act`) }
      const openSpells = async (slot) => {
        await select(slot)
        await js(`[...document.querySelectorAll('${C1} button.act')].find((b) => /spell/i.test(b.innerText)).click()`)
        await waitFor(`${C1} .spellpick`); await sleep(200)
      }
      const info = () => js(`(() => { const e = document.querySelector('${C1} .spellinfo'); return e ? { name: e.querySelector('.shead b').innerText, text: e.innerText, status: e.querySelector('.status').innerText } : null })()`)
      const spellButtons = () => js(`[...document.querySelectorAll('${C1} .spelllist button.spell')].map((b) => ({ name: b.querySelector('.sname').innerText, off: b.classList.contains('off'), dis: b.disabled }))`)
      // Everything the picker shows is in the window and readable: the list
      // and the description side by side, nothing cut off or covered.
      const fits = () => js(`(() => {
        const pick = document.querySelector('${C1} .spellpick'), list = pick?.querySelector('.spelllist'), inf = pick?.querySelector('.spellinfo')
        if (!pick || !list || !inf) return 'no picker'
        const a = list.getBoundingClientRect(), b = inf.getBoundingClientRect(), vw = document.documentElement.clientWidth
        if (a.right > b.left + 0.5 && a.bottom > b.top + 0.5) return 'list and description overlap'
        if (b.right > vw + 0.5 || a.left < -0.5) return 'outside the window: ' + b.right + ' > ' + vw
        scrollTo(0, 0)
        const p = pick.getBoundingClientRect()
        if (p.bottom > innerHeight + 0.5) return 'below the window without scrolling: ' + Math.round(p.bottom) + ' > ' + innerHeight
        for (const e of [inf, ...inf.children, ...list.querySelectorAll('button')]) if (e.scrollWidth > e.clientWidth + 1) return 'cut off: ' + e.innerText.slice(0, 40)
        inf.scrollIntoView({ block: 'nearest' })
        const r = inf.getBoundingClientRect(), top = document.elementFromPoint(r.left + r.width / 2, r.top + Math.min(20, r.height / 2))
        if (!inf.contains(top)) return 'description covered by ' + (top?.className || top?.tagName)
        return ''
      })()`)

      let c = await practice()
      check(c?.next === 'orders' && c.retreat, `a practice encounter at order entry, retreat offered (${c?.next}, ${c?.retreat})`)
      const mage = slotOf(c, 'MAGE'), priest = slotOf(c, 'PRIEST')
      await openSpells(mage)
      let bs = await spellButtons(), i1 = await info()
      // The library's mage was made with HALITO and KATINO, and knows no
      // more after the library is loaded (COMBAT-1, #32).
      check(JSON.stringify(bs.map((b) => [b.name, b.off])) === JSON.stringify([['HALITO', false], ['KATINO', false]]),
        `the mage's known spells, as made, both castable (${JSON.stringify(bs)})`)
      check(i1?.name === bs[0].name && /damage|sleep/.test(i1.text) && /Affects:/.test(i1.text) && /Casts left at this level:\s*2 of 2/.test(i1.text) && /can cast/.test(i1.status),
        `the description panel: name, level, what it does, target, casts left, castable (${i1?.text.replace(/\n/g, ' | ')})`)
      await js(`document.querySelectorAll('${C1} .spelllist button.spell')[1].dispatchEvent(new MouseEvent('mouseenter'))`); await sleep(200)
      check((await info())?.name === bs[1].name, 'pointing at another spell describes it at once')
      await js(`document.querySelectorAll('${C1} .spelllist button.spell')[0].dispatchEvent(new FocusEvent('focus'))`); await sleep(200)
      check((await info())?.name === bs[0].name, 'focusing a spell (the keyboard) describes it at once')
      check(await fits() === '', `the picker fits the window (${await fits()})`)
      await shot('spell-picker')

      // The priest's CALFO: listed, described, not cast; clicking it orders nothing.
      await openSpells(priest)
      bs = await spellButtons()
      const calfo = bs.findIndex((b) => b.name === 'CALFO')
      await js(`document.querySelectorAll('${C1} .spelllist button.spell')[${calfo}].dispatchEvent(new FocusEvent('focus'))`); await sleep(200)
      check(/cannot cast CALFO/.test((await info())?.status ?? ''), 'focusing a spell that cannot be cast says why at once')
      await js(`document.querySelectorAll('${C1} .spelllist button.spell')[${calfo}].click()`); await sleep(400)
      const ic = await info()
      c = await cv()
      check(calfo >= 0 && bs[calfo].off && ic?.name === 'CALFO' && /cannot cast CALFO: for chests/.test(ic.status) && !c.orders?.some((o) => o.slot === priest),
        `a spell with no effect in combat is shown with its reason and not ordered (${ic?.status})`)
      check(await fits() === '', 'and the picker still fits')
      await shot('spell-unavailable')

      // The mage spends both casts (KATINO twice, everyone else parries),
      // then finds none left, with the reason.
      let spent = 0
      for (let round = 0; round < 6 && spent < 2; round++) {
        c = await cv()
        if (!c || c.next !== 'orders') { c = await practice(); spent = 0 }
        for (const m of c.party.filter((m) => m.canAct)) {
          const katino = m.slot === mage
          await js(`${S}.CombatSelect(${m.slot})`)
          const g = (await cv()).groups[0].number
          const r = katino ? `{ slot: ${m.slot}, action: 'spell', spell: 'KATINO', group: ${g}, ally: 0, item: -1 }` : `{ slot: ${m.slot}, action: 'parry', group: 0, spell: '', ally: 0, item: -1 }`
          await js(`${S}.CombatOrder(${r})`)
        }
        await js(`${S}.CombatFinishOrders()`)
        c = await toOrders()
        if (c?.next === 'orders') {
          await js(`${S}.CombatSelect(${mage})`)
          const k = (await cv()).spells?.find((sp) => sp.name === 'KATINO')
          spent = k ? k.max - k.left : spent
        }
      }
      c = await cv()
      check(c?.next === 'orders' && spent === 2, `the mage has spent both casts in one encounter (${spent})`)
      await select(mage)
      const menuText = await js(`[...document.querySelectorAll('${C1} button.act')].find((b) => /spell/i.test(b.innerText))?.innerText ?? ''`)
      check(/no spell can be cast now: see why/.test(menuText), `the Spell order says none can be cast now (${menuText.replace(/\n/g, ' ')})`)
      await openSpells(mage)
      bs = await spellButtons()
      const ie = await info()
      check(bs.length === 2 && bs.every((b) => b.off) && /cannot cast HALITO: no level 1 mage casts left \(0 of 2\); a stay at the Inn restores them\./.test(ie?.status ?? ''),
        `no casts left: every spell says why (${ie?.status})`)
      check(await fits() === '', 'the exhausted picker fits')
      await shot('spell-exhausted')
      await js(`[...document.querySelectorAll('${C1} button.small')].find((b) => b.innerText.trim() === 'Back').click()`); await sleep(300)

      // Party Retreat: one click, the review explains it, then the round.
      await js('scrollTo(0, 0)')
      const rb = await rect(`${C1} .retreat button`)
      check(rb && rb.b <= H && rb.r <= W, `Party retreat is in the window (${JSON.stringify(rb)})`)
      await clickText('Party retreat', C1)
      c = await cv()
      const note = await js(`document.querySelector('${C1} .runnote')?.innerText ?? ''`)
      check(c.next === 'confirm' && c.orders.filter((o) => o.action === 'run').length === 1 && /^The party attempts to retreat: \w+ will try to lead it away/.test(note),
        `the review shows one Run and explains the retreat (${note})`)
      await shot('retreat-review')
      await clickText('Fight! ↵', C1)
      let gloss = ''
      for (let i = 0; i < 80 && !gloss; i++) {
        gloss = await js(`[...document.querySelectorAll('.combat .messages p.gloss')].map((p) => p.innerText).join(' / ')`)
        if (/escapes|Escape failed/.test(gloss)) break
        gloss = ''
        const b = await js(`(() => { const b = document.querySelector('${C1} button.primary'); if (!b || b.disabled) return false; b.click(); return true })()`)
        if (!b) { if ((await view()).screen !== 'combat') break; await sleep(300) } else await sleep(350)
      }
      check(/^The party attempts to retreat! \/ (The party escapes!|Escape failed — the fight goes on\.)$/.test(gloss), `the attempt and its outcome are said (${gloss})`)
      await shot('retreat-outcome')
      for (let i = 0; i < 80 && (await view()).screen === 'combat'; i++) {
        c = await cv()
        if (c.next === 'orders') await js(`${S}.CombatRetreat()`)
        else if (c.next === 'confirm') await js(`${S}.CombatCommit()`)
        else if (c.next === 'leave') await js(`${S}.CombatLeave()`)
        else if (c.next === 'choice') await js(`${S}.CombatChooseBoon(1)`)
        else await js(`${S}.${{ advance: 'CombatAdvance', 'end-round': 'CombatEndRound', start: 'CombatStart' }[c.next] ?? 'State'}()`)
      }
      check((await view()).screen !== 'combat', 'the encounter is over')
      check(sha(GAME) === before, 'practice combat, spells and retreat saved nothing')

      // The character sheet: each known spell with Go's description.
      await goto('castle')
      const v = await view()
      const mi = v.partyBar.findIndex((p) => /-MAG$/.test(p.class))
      check(mi >= 0, `the mage is in the party bar (${mi})`)
      await clickSlot(mi)
      if (await toggle() === 'Show sheet') await clickText('Show sheet', '.frame .head')
      const rows = await js(`[...document.querySelectorAll('${A} .sheet ul.spells li')].map((li) => li.innerText.replaceAll(String.fromCharCode(10), ' '))`)
      check(rows.length === 2 && rows.every((r) => /^(HALITO|KATINO)\s+mage 1\s+\S/.test(r)), `the sheet lists the mage's spells with descriptions (${JSON.stringify(rows)})`)
      await shot('sheet-spells')
    },

    // NAV-1: a game opens at the Castle; the Castle's five destinations and
    // no Actions; the Edge's five actions, no Go to list and no practice;
    // the top bar's Castle (Esc) and toggles, on one row, never Settings;
    // Escape from every place back to the Castle; Escape in the maze and in
    // combat does nothing; practice is still there for the tests (its
    // Session methods), and none of it is saved.
    async nav() {
      const HEAD = '.frame .head'
      const headButtons = () => buttons(HEAD)
      const title = () => js(`document.querySelector('.frame .head .title')?.innerText.trim() ?? ''`)
      const loc = async () => (await view()).location
      // The top bar fits: every button whole and in the window, on one row,
      // and nothing scrolls sideways.
      const barFits = () => js(`(() => { const h = document.querySelector('${HEAD}'); const bs = [...h.querySelectorAll('button')]; const ys = new Set(bs.map((b) => Math.round(b.getBoundingClientRect().top))); return h.scrollWidth <= h.clientWidth + 1 && document.scrollingElement.scrollWidth <= innerWidth + 1 && ys.size <= 1 && bs.every((b) => { const r = b.getBoundingClientRect(); return r.width > 0 && r.right <= innerWidth + 0.5 && b.scrollWidth <= b.clientWidth + 1 }) })()`)

      // Select Game: the library's own selection; continuing opens the Castle.
      await goto('edge')
      await clickText('Select Game', B)
      check((await view()).screen === 'games' && await js(`!!document.querySelector('.games')`), 'Select Game opens the game selection')
      await shot('nav-select-game')
      await reload()
      await clickIn('.games .slot, .games article, .games section', 'Game 1', 'button')
      check(await loc() === 'castle' && await title() === 'The Castle', `Continue opens the game at the Castle (${await title()})`)

      // The Castle: the top bar, its five destinations and no Actions.
      const castle = await headButtons()
      check(JSON.stringify(castle) === JSON.stringify(['Hide party', 'Story', 'Show sheet', 'Hide panels']), `the Castle's top bar: the toggles, no way back from the Castle itself (${JSON.stringify(castle)})`)
      check(!castle.some((b) => /Settings/.test(b)), 'no Settings in the top bar')
      check(await barFits(), 'the top bar fits on one row')
      const dests = await js(`[...document.querySelectorAll('${B} nav[aria-label=Destinations] button')].map((b) => b.innerText.trim())`)
      check(JSON.stringify(dests) === JSON.stringify(["Gilgamesh's Tavern", "The Adventurer's Inn", "Boltac's Trading Post", 'The Temple of Cant', 'The Edge of Town']), `the Castle's five destinations (${JSON.stringify(dests)})`)
      check(await js(`!document.querySelector('${B} section.actions') && [...document.querySelectorAll('${B} button')].length === 5`), 'the Castle has no Actions, and nothing else to press')
      await shot('nav-castle')

      // Each Castle destination, and Escape back.
      for (const [label, place] of [["Gilgamesh's Tavern", 'tavern'], ["The Adventurer's Inn", 'inn'], ["Boltac's Trading Post", 'trading'], ['The Temple of Cant', 'temple'], ['The Edge of Town', 'edge']]) {
        await clickText(label, B)
        const at = await loc()
        const bar = await headButtons()
        check(at === place && bar[0] === '← Castle Esc' && !bar.some((b) => /Settings/.test(b)), `${label} is open from the Castle, Castle (Esc) first in its top bar (${at}, ${JSON.stringify(bar)})`)
        check(await barFits(), `${place}: the top bar fits on one row`)
        await esc()
        check(await loc() === 'castle', `Escape at ${label} returns to the Castle`)
      }

      // The Edge of Town: its five actions, no Go to list, no practice.
      await clickText('The Edge of Town', B)
      const edge = await buttons(`${B} section.actions`)
      check(JSON.stringify(edge) === JSON.stringify(['Enter the Maze', 'The Training Grounds', 'Select Game', 'Settings', 'Leave the Game']), `the Edge's five actions, in order (${JSON.stringify(edge)})`)
      check(await js(`!document.querySelector('${B} nav[aria-label=Destinations]') && [...document.querySelectorAll('${B} button')].length === 5`), 'no Go to list at the Edge')
      check(!/practice|guards a chest/i.test(await js(`document.querySelector('.frame').innerText`)) && await js(`!document.querySelector('.frame input[type=checkbox]')`), 'no practice controls at the Edge')
      const bar = await headButtons()
      check(JSON.stringify(bar) === JSON.stringify(['← Castle Esc', 'Hide party', 'Story', 'Show sheet', 'Hide panels']), `the Edge's top bar: Castle (Esc), party, story, sheet, panels (${JSON.stringify(bar)})`)
      await clickSlot(0)
      check(JSON.stringify(await headButtons()) === JSON.stringify(['← Castle Esc', 'Hide party', 'Story', 'Hide sheet', 'Hide panels']) && await barFits(), 'with a sheet shown, Hide sheet, and the bar still fits')
      await shot('nav-edge')
      await clickText('Hide sheet', HEAD)
      check(await js(`!!document.querySelector('${A} figure.scene')`), 'Hide sheet brings the picture back')
      check(await barFits(), 'the Edge’s top bar fits on one row')

      // The Training Grounds and Settings, from the Edge's actions; Escape back.
      for (const [label, place] of [['The Training Grounds', 'training'], ['Settings', 'settings']]) {
        await goto('edge')
        await clickText(label, `${B} section.actions`)
        check(await loc() === place, `${label} opens from the Edge's actions`)
        if (place === 'settings') check(await js(`!!document.querySelector('.frame select, .frame input[type=radio], .frame fieldset')`), 'Settings shows its choices')
        await esc()
        check(await loc() === 'castle', `Escape at ${label} returns to the Castle`)
      }
      await goto('edge'); await esc()
      check(await loc() === 'castle', 'Escape at the Edge of Town returns to the Castle')

      // The toggles: party, story and panels.
      await goto('edge')
      await clickText('Hide party', HEAD)
      check(!(await rect(D)) && (await headButtons()).includes('Show party'), 'Hide party hides the bar')
      await clickText('Show party', HEAD)
      check(!!(await rect(D)), 'Show party brings it back')
      const storyWas = !!(await rect(STORY))
      await clickText('Story', HEAD)
      check(!!(await rect(STORY)) !== storyWas, 'Story opens or closes the story box')
      if (!(await rect(STORY))) await clickText('Story', HEAD)
      await clickText('Hide panels', HEAD)
      check(!(await rect(B)) && !!(await rect('.restore')), 'Hide panels leaves the picture')
      await esc()
      check(!!(await rect(B)) && await loc() === 'edge', 'Escape shows the panels again, at the Edge')

      // A real expedition, as the player sets out: Escape in the maze
      // leaves nothing and the Castle is refused; the party comes home by
      // the game's own way (Camp, then the stairs up).
      await goto('edge')
      await clickText('Enter the Maze', B)
      if (await js(`!!document.querySelector('.check-equipment')`)) await clickText('Continue into maze', B)
      let rv = await view()
      check(rv.real && rv.screen === 'dungeon', `Enter the Maze sets out on a real expedition (${rv.screen})`)
      await esc(); await esc()
      rv = await view()
      check(rv.real && rv.screen === 'dungeon' && rv.location === 'maze', `Escape in the maze leaves nothing (${rv.screen}, ${rv.location})`)
      rv = JSON.parse(await js(`${S}.ReturnToCastle().then((v) => JSON.stringify(v))`))
      check(!!rv.error && rv.real, `and the Castle is refused during the expedition (${rv.error})`)
      await js(`${S}.DungeonAction('leave-camp', 0)`)
      rv = JSON.parse(await js(`${S}.DungeonAction('yes', 0).then((v) => JSON.stringify(v))`))
      await reload()
      check(!rv.real && rv.location === 'castle' && await title() === 'The Castle', `the party climbs out to the Castle (${rv.location})`)

      // The maze and combat in practice: Escape leaves neither; practice is
      // still the tests' (its Session methods), and it saves nothing.
      const before = sha(GAME)
      await goto('edge')
      let pv = JSON.parse(await js(`${S}.PracticeExpedition().then((v) => JSON.stringify(v))`))
      check(!pv.error && pv.screen === 'dungeon', `a practice expedition starts through its Session method (${pv.error || pv.screen})`)
      await reload(); await esc(); await esc()
      pv = await view()
      check(pv.screen === 'dungeon' && pv.location === 'maze', `Escape in the maze leaves nothing (${pv.screen}, ${pv.location})`)
      pv = JSON.parse(await js(`${S}.ReturnToCastle().then((v) => JSON.stringify(v))`))
      check(!!pv.error && pv.location === 'maze', `and the Castle is refused there (${pv.error})`)
      await js(`${S}.AbandonPractice()`)
      await goto('edge')
      pv = JSON.parse(await js(`${S}.PracticeEncounter(true).then((v) => JSON.stringify(v))`))
      check(!pv.error && pv.screen === 'combat', `a practice encounter with a chest starts through its Session method (${pv.error || pv.screen})`)
      await reload(); await esc(); await esc()
      check((await view()).screen === 'combat', 'Escape in combat leaves nothing')
      await js(`${S}.AbandonPractice()`)
      await goto('castle')
      check((await view()).screen === 'roster' && sha(GAME) === before, 'practice ended; nothing of it was saved')
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
