<script>
  // The combat screen (M7a). It renders view.combat (Go's app.CombatView)
  // and calls Session methods; it holds no combat state beyond what the
  // player is pointing at (an action waiting for its target) and the pacing
  // preference. Legality, targets and outcomes all come from Go.
  //
  // Props:
  //   view   the Go View (view.combat is the combat view)
  //   busy   a request is running
  //   call   (fn, ...args) => Promise   runs a Session method and stores the new view
  import * as S from '../../wailsjs/go/main/Session.js'
  import { condition, ranks, step, delay, needsOrder, orderFor, orderLabel, current, highlight, chips, targetKind } from './combat.js'

  let { view, busy = false, call } = $props()

  const c = $derived(view.combat)
  // Go sends empty lists as null; the screen reads them through these.
  const groups = $derived(c.groups ?? [])
  const party = $derived(c.party ?? [])
  const r = $derived(ranks(party))
  const ev = $derived(current(c))
  const hl = $derived(highlight(ev))
  const fx = $derived(chips(ev))
  const missing = $derived(needsOrder(c))

  let auto = $state(false) // automatic pacing of playback and results
  let speed = $state('normal')
  let details = $state(false) // the expanded review
  let pending = $state(null) // { action, needs: 'group'|'member'|'spell'|'item', spell? }
  let chestAction = $state('') // chest action waiting for who does it

  // A new actor, phase or step drops whatever was half chosen.
  $effect(() => {
    void c.actor, c.phase, c.next
    pending = null
  })
  $effect(() => {
    if (c.next !== 'chest') chestAction = ''
  })

  // Pacing is the front end's. The opening pause always runs on; playback,
  // round ends and chest results run on when automatic pacing is on.
  $effect(() => {
    const st = step(c)
    if (busy || !st.method || !(st.auto && (auto || c.next === 'start'))) return
    const t = setTimeout(() => call(S[st.method]), delay(c.next, speed))
    return () => clearTimeout(t)
  })

  function onkey(e) {
    if (e.target.closest?.('input, select, textarea')) return
    const st = step(c)
    if ((e.key === ' ' || e.key === 'Enter') && st.method && !busy) {
      e.preventDefault()
      call(S[st.method])
    }
  }

  // Order entry. After an order is accepted, move on to the next member who
  // still needs one (navigation only; Go checks everything).
  async function order(req) {
    await call(S.CombatOrder, { slot: c.actor, action: '', group: 0, spell: '', ally: 0, item: -1, ...req })
    pending = null
    const next = needsOrder(view.combat)
    if (!view.error && view.combat?.next === 'orders' && next.length) await call(S.CombatSelect, next[0])
  }

  function choose(opt) {
    if (!opt.available) return
    const kind = targetKind(opt, c)
    if (kind === 'group' && groups.length === 1) return order({ action: opt.action, group: groups[0].number })
    if (kind === 'none') return order({ action: opt.action })
    pending = { action: opt.action, needs: kind }
  }

  function chooseSpell(sp) {
    if (sp.target === 'none') return order({ action: 'spell', spell: sp.name })
    if (sp.target === 'group' && groups.length === 1) return order({ action: 'spell', spell: sp.name, group: groups[0].number })
    pending = { action: 'spell', needs: sp.target, spell: sp.name }
  }

  function pickGroup(number) {
    if (pending?.needs === 'group') order({ action: pending.action, spell: pending.spell ?? '', group: number })
  }

  function pickMember(m) {
    if (c.next === 'orders' && pending?.needs === 'member') return order({ action: 'spell', spell: pending.spell, ally: m.slot })
    if (c.next === 'orders' && m.canAct) return call(S.CombatSelect, m.slot)
    if (c.next === 'chest' && chestAction) {
      const who = (c.chest.members ?? []).find((x) => x.slot === m.slot)
      if (who?.canAct) call(S.ChestAction, chestAction, m.slot)
    }
  }

  function chooseChest(a) {
    if (!a.available) return
    if (a.action === 'leave') return call(S.ChestAction, 'leave', 0)
    chestAction = a.action
  }

  const chestVerb = { open: 'open the chest', inspect: 'inspect it', calfo: 'cast CALFO', disarm: 'disarm it' }
  const phaseTitle = { opening: 'An encounter', friendly: 'A friendly encounter', orders: 'Orders', confirm: 'Confirm orders',
    playback: 'The round', chest: 'A chest', victory: 'Victory', defeat: 'Defeat' }
</script>

<svelte:window onkeydown={onkey} />

<div class="combat">
  <header class="bar">
    <h2>{phaseTitle[c.phase] ?? 'Combat'} <span class="muted">round {c.round}</span></h2>
    <span class="tag">Practice encounter{view.practiceChest ? ' · with a chest' : ''} · results are not kept</span>
    <div class="spacer"></div>
    <label class="muted"><input type="checkbox" bind:checked={auto} /> automatic</label>
    <select bind:value={speed} aria-label="Automatic speed">
      <option value="slow">slow</option><option value="normal">normal</option><option value="fast">fast</option>
    </select>
    <button class="small" disabled={busy} onclick={() => call(S.AbandonPractice)}>Abandon</button>
  </header>

  <!-- The battlefield: monster groups ahead on a neutral backdrop. -->
  <section class="stage">
    <div class="groups">
      {#each groups as g (g.number)}
        <button class="group" class:actor={hl.actorGroup === g.number} class:target={hl.targetGroup === g.number}
          class:pick={pending?.needs === 'group'} disabled={pending?.needs !== 'group' || busy} onclick={() => pickGroup(g.number)}>
          <span class="num">{g.number}</span>
          <span class="tokens" aria-hidden="true">{#each Array(Math.min(g.count, 9)) as _}<i></i>{/each}</span>
          <span class="name">{g.count} {g.name}</span>
          {#each fx.filter((f) => f.group === g.number) as f}<span class="chip {f.kind}">{f.text}</span>{/each}
        </button>
      {/each}
      {#if !groups.length}<p class="muted">No monsters remain.</p>{/if}
    </div>

    <!-- The one message area. -->
    <div class="messages" aria-live="polite">
      {#if c.phase === 'opening'}
        <p>AN ENCOUNTER</p>
        {#if c.surprise === 'party'}<p>YOU SURPRISED THE MONSTERS!</p>{/if}
        {#if c.surprise === 'monsters'}<p>THE MONSTERS SURPRISED YOU!</p>{/if}
      {:else if c.phase === 'friendly'}
        <p>THEY HAIL YOU IN WELCOME!</p>
      {:else if c.phase === 'defeat'}
        <p>TOTAL PARTY KILL</p><p>THE PARTY HAS PERISHED</p>
      {:else if c.phase === 'victory'}
        <!-- As the terminal: the shares, not the reward messages. -->
      {:else if c.lines?.length}
        {#each c.lines as line}<p>{line}</p>{/each}
      {:else if c.phase === 'playback' && ev?.silent}
        <p class="muted">…</p>
      {/if}
      {#if c.phase === 'victory'}
        <p>FOR KILLING THE MONSTERS EACH SURVIVOR GETS {c.xp} EXPERIENCE POINTS</p>
        <p>EACH SHARE IS WORTH {c.gold} GP!</p>
      {/if}
    </div>
  </section>

  <!-- Commands: what the next step needs. -->
  <section class="commands">
    {#if c.next === 'friendly'}
      <button class="primary" disabled={busy} onclick={() => call(S.CombatFriendly, true)}>Fight</button>
      <button disabled={busy} onclick={() => call(S.CombatFriendly, false)}>Leave in peace</button>
    {:else if c.next === 'orders'}
      {@const actor = party.find((m) => m.slot === c.actor)}
      {#if actor}
        <div class="menu">
          <div class="who"><b>{actor.slot} {actor.name}</b> <span class="muted">{actor.slot <= 3 ? 'front row' : 'back row'}</span></div>
          {#if !pending}
            {#each c.menu ?? [] as opt}
              <button class="act" class:off={!opt.available} disabled={busy || !opt.available} title={opt.reason}
                onclick={() => choose(opt)}>
                <kbd>{opt.key}</kbd> {opt.action}
                {#if opt.reason}<span class="why">{opt.reason}</span>{/if}
                {#if opt.note}<span class="why">{opt.note}</span>{/if}
              </button>
            {/each}
          {:else if pending.needs === 'spell'}
            <div class="muted">Which spell?</div>
            {#each c.spells ?? [] as sp}
              <button class="act" disabled={busy} onclick={() => chooseSpell(sp)}>{sp.name} <span class="why">{sp.kind} {sp.level} · {sp.target === 'none' ? 'no target' : sp.target}</span></button>
            {:else}<div class="muted">No spell can be cast now.</div>{/each}
            <button class="small" onclick={() => (pending = null)}>Back</button>
          {:else if pending.needs === 'item'}
            <div class="muted">Use which item?</div>
            {#each c.items ?? [] as it}
              <button class="act" disabled={busy} onclick={() => order({ action: 'use', item: it.index })}>{it.name}</button>
            {/each}
            <button class="small" onclick={() => (pending = null)}>Back</button>
          {:else if pending.needs === 'group'}
            <div class="muted">{pending.spell ?? pending.action}: choose a monster group above.</div>
            <button class="small" onclick={() => (pending = null)}>Back</button>
          {:else if pending.needs === 'member'}
            <div class="muted">{pending.spell}: choose a party member below.</div>
            <button class="small" onclick={() => (pending = null)}>Back</button>
          {/if}
        </div>
      {/if}
      <div class="finish">
        <span class="muted">{missing.length ? `${missing.length} still need orders` : 'Every member who can act has an order.'}</span>
        <button class="primary" disabled={busy || missing.length > 0} onclick={() => call(S.CombatFinishOrders)}>Review orders</button>
      </div>
    {:else if c.next === 'confirm'}
      <div class="review">
        <table>
          <tbody>
            {#each party as m}
              {@const o = orderFor(c, m.slot)}
              <tr><td>{m.slot}</td><td>{m.name}</td><td>{o ? orderLabel(o) : m.canAct ? '—' : 'cannot act'}</td></tr>
            {/each}
          </tbody>
        </table>
        <button class="primary" disabled={busy} onclick={() => call(S.CombatCommit)}>Fight! <kbd>↵</kbd></button>
        <button disabled={busy} onclick={() => call(S.CombatBack)}>Change orders (clears all)</button>
      </div>
    {:else if c.next === 'choice'}
      <div class="muted">WHICH BOON WILL YOU INVOKE ?</div>
      {#each c.boons ?? [] as b, i}
        <button class="act" disabled={busy} onclick={() => call(S.CombatChooseBoon, i + 1)}>{i + 1}) {b}</button>
      {/each}
    {:else if c.next === 'chest'}
      <div class="menu">
        {#if !chestAction}
          {#each c.chest.actions ?? [] as a}
            <button class="act" disabled={busy || !a.available} onclick={() => chooseChest(a)}>
              <kbd>{a.key}</kbd> {a.action}{#if a.note}<span class="why">{a.note}</span>{/if}
            </button>
          {/each}
        {:else}
          <div class="muted">Who will {chestVerb[chestAction]}? Choose a party member below.</div>
          <button class="small" onclick={() => (chestAction = '')}>Back</button>
        {/if}
      </div>
    {:else if step(c).method}
      <button class="primary" disabled={busy} onclick={() => call(S[step(c).method])}>{step(c).label} <kbd>space</kbd></button>
    {/if}

    {#if c.chest?.present && c.chest.reports?.length}
      <div class="reports">
        <div class="muted">What the party has been told (reports can be wrong):</div>
        {#each c.chest.reports as rep}
          <div>{rep.member} ({rep.how}): {rep.said || 'nothing'}{rep.setOff ? ' — it went off' : ''}</div>
        {/each}
      </div>
    {/if}
  </section>

  <!-- The team strip: front row and back row, in formation order. -->
  <section class="strip">
    {#each [['Front row', r.front], ['Back row', r.back]] as [label, row]}
      <div class="rank">
        <div class="muted">{label}</div>
        <div class="cards">
          {#each row as m (m.slot)}
            {@const cond = condition(m)}
            {@const o = orderFor(c, m.slot)}
            {@const pickable = (c.next === 'orders' && (pending?.needs === 'member' || (!pending && m.canAct))) ||
              (c.next === 'chest' && chestAction && (c.chest.members ?? []).find((x) => x.slot === m.slot)?.canAct)}
            <button class="card {cond}" class:selected={c.next === 'orders' && c.actor === m.slot}
              class:actor={hl.actorSlot === m.slot} class:target={hl.targetSlot === m.slot} class:pick={pickable}
              disabled={busy || !pickable} onclick={() => pickMember(m)}>
              <div class="line1"><b>{m.slot} {m.name}</b><span class="status">{m.status}</span></div>
              <div class="line2"><span>{m.class}</span><span>AC {m.ac}</span><span class="hp">{m.hp}/{m.maxHp}</span></div>
              <div class="bar"><i style="width:{Math.max(0, (100 * m.hp) / Math.max(1, m.maxHp))}%"></i></div>
              {#if m.poisoned}<span class="poison" title="poisoned">●</span>{/if}
              {#if c.phase === 'orders' || c.phase === 'confirm'}
                <div class="order">{o ? orderLabel(o) : m.canAct ? '—' : 'cannot act'}</div>
              {/if}
              {#each fx.filter((f) => f.slot === m.slot) as f}<span class="chip {f.kind}">{f.text}</span>{/each}
            </button>
          {/each}
        </div>
      </div>
    {/each}
    {#if c.frozen}<div class="muted frozen">Values as at the start of the round (Classic); they update when the round ends.</div>{/if}
  </section>

  <!-- The expandable review: this round's orders and what has been shown. -->
  <section class="details">
    <button class="small" onclick={() => (details = !details)}>{details ? 'Hide' : 'Show'} details</button>
    {#if details}
      <div class="log">
        {#if c.orders?.length}
          <h3>Orders</h3>
          {#each c.orders as o}<div>{o.slot} {o.member}: {orderLabel(o)}</div>{/each}
        {/if}
        {#if c.events?.length}
          <h3>This round so far</h3>
          {#each c.events as e, i}
            <div class="ev" class:now={i === c.current}>{e.silent ? '…' : e.lines.join(' ')}</div>
          {/each}
        {/if}
      </div>
    {/if}
  </section>
</div>

<style>
  .combat { display: grid; gap: 12px; }
  .bar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
  .bar h2 { margin: 0; }
  .spacer { flex: 1; }
  .tag { font-size: 11px; border: 1px solid var(--line); border-radius: 9px; padding: 1px 8px; color: var(--muted); }
  .stage { position: relative; min-height: 220px; border-radius: 8px; padding: 14px;
    background: radial-gradient(ellipse at 50% 120%, #3a3329 0%, #1d1a16 55%, #0d0c0b 100%); border: 1px solid var(--line); }
  .groups { display: flex; flex-wrap: wrap; gap: 10px; justify-content: center; }
  .group { display: grid; justify-items: center; gap: 4px; min-width: 140px; padding: 8px 10px; border-radius: 8px;
    background: rgba(0, 0, 0, 0.35); border: 1px solid #555; color: var(--text); position: relative; }
  .group:disabled { opacity: 1; cursor: default; }
  .group .num { font-weight: 800; color: #e0a54a; }
  .tokens { display: flex; gap: 4px; }
  .tokens i { width: 12px; height: 16px; border-radius: 6px 6px 3px 3px; background: #6b7a4e; display: block; }
  .group.actor { border-color: #f2c14e; box-shadow: 0 0 0 1px #f2c14e, 0 0 12px rgba(242, 193, 78, 0.45); }
  .group.target { border-color: #fff; box-shadow: 0 0 0 1px #fff; }
  .group.pick { border-style: dashed; border-color: var(--accent); cursor: pointer; }
  .messages { margin-top: 14px; min-height: 64px; font-family: ui-monospace, Menlo, monospace; text-align: center; }
  .messages p { margin: 2px 0; }
  .commands { display: grid; gap: 8px; }
  .menu { display: grid; gap: 4px; max-width: 520px; }
  .who { margin-bottom: 2px; }
  .act { display: flex; gap: 8px; align-items: baseline; text-align: left; text-transform: capitalize; }
  .act.off { opacity: 0.55; }
  .why { font-size: 11px; color: #b6a98e; text-transform: none; margin-left: auto; }
  kbd { font: 600 11px ui-monospace, Menlo, monospace; border: 1px solid var(--line); border-radius: 3px; padding: 0 4px; }
  .finish { display: flex; gap: 10px; align-items: center; }
  .review table { border-collapse: collapse; margin-bottom: 8px; }
  .review td { padding: 2px 10px 2px 0; }
  .reports { font-size: 13px; }
  .strip { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
  .cards { display: flex; gap: 6px; }
  .card { flex: 1; min-width: 0; text-align: left; position: relative; padding: 6px 8px; border-radius: 6px;
    background: var(--panel); border: 1px solid var(--line); color: var(--text); }
  .card:disabled { opacity: 1; cursor: default; }
  .card .line1, .card .line2 { display: flex; justify-content: space-between; gap: 6px; font-size: 12px; white-space: nowrap; }
  .card .bar { height: 3px; background: #333; border-radius: 2px; margin-top: 3px; }
  .card .bar i { display: block; height: 100%; background: var(--accent); border-radius: 2px; }
  .card .order { font-size: 11px; color: #f2c14e; margin-top: 3px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .card.selected { border-color: #f2c14e; box-shadow: 0 0 0 1px #f2c14e; }
  .card.actor { border-color: #f2c14e; box-shadow: 0 0 0 1px #f2c14e, 0 0 12px rgba(242, 193, 78, 0.45); }
  .card.target { border-color: #fff; box-shadow: 0 0 0 1px #fff; }
  .card.pick { border-style: dashed; border-color: var(--accent); cursor: pointer; }
  /* Conditions: dead members stay, translucent; paralysis is distinct. */
  .card.dead { opacity: 0.5; border-style: dashed; }
  .card.dead .status { color: #ddd; }
  .card.paralyzed { background: #1d2430; }
  .card.paralyzed .status { color: #cfe3ff; }
  .card.asleep .status, .card.afraid .status { color: #c9c2ff; }
  .poison { position: absolute; top: 2px; right: 4px; color: #7bd36a; font-size: 10px; }
  .chip { position: absolute; top: -8px; right: 6px; font: 700 12px ui-monospace, Menlo, monospace; padding: 0 5px; border-radius: 8px; background: #000; }
  .chip.hurt { color: #ff7a5c; }
  .chip.heal { color: #8fe38f; }
  .chip.status, .chip.monster { color: #f2c14e; top: auto; bottom: -8px; }
  .frozen { grid-column: 1 / -1; }
  .details .log { margin-top: 6px; font-size: 13px; max-height: 260px; overflow: auto; }
  .details h3 { font-size: 13px; margin: 6px 0 2px; }
  .ev.now { color: #f2c14e; }
</style>
