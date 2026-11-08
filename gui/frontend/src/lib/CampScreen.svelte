<script>
  // The Camp screen (M6b-3B). It renders view.camp (Go's gui.CampPanel) and
  // calls Session.CampAction with one of the actions Go lists; it keeps no
  // game state. Choosing which spell to cast or which item to use is only a
  // picker here: nothing is sent until the choice is made.
  //
  // Props:
  //   camp   view.camp
  //   busy   a request is running
  //   call   (fn, ...args) => Promise   runs a Session method and stores the new view
  import * as S from '../../wailsjs/go/main/Session.js'
  import { condition } from './combat.js'
  import { labels, operation, partyPick, back, buttons, menuOrder, inspectOrder, spellBook, slots, whyNot } from './camp.js'

  let { camp, busy = false, call } = $props()

  const v = $derived(camp.view)
  const member = $derived(v.member >= 0 ? camp.party[v.member] : null)
  const items = $derived(member?.items ?? [])
  const castable = $derived((v.spells ?? []).filter((sp) => sp.castable))
  let choosing = $state('') // 'cast' or 'use' while picking; nothing is sent yet
  let gold = $state(0)

  function act(action, args = {}) {
    choosing = ''
    return call(S.CampAction, { action, slot: 0, n: 0, gold: 0, spell: '', priest: false, ...args })
  }

  // Inspect's buttons; Cast and Use open a picker, the rest act at once.
  const inspectButtons = $derived(
    buttons(camp, inspectOrder).map((b) => {
      if (b.action === 'cast' && b.enabled && !castable.length)
        return { ...b, enabled: false, reason: 'No spell this character knows can be cast in Camp now.' }
      if (b.action === 'use' && b.enabled && !items.length) return { ...b, enabled: false, reason: 'Carrying nothing.' }
      return b
    }),
  )

  function inspectClick(b) {
    if (b.action === 'cast' || b.action === 'use') choosing = choosing === b.action ? '' : b.action
    else act(b.action)
  }

  function pick(slot) {
    const p = partyPick(camp, slot)
    if (p) act(p.action, { slot })
  }

  function onkey(e) {
    if (busy || e.target.closest?.('input, select, textarea') || e.metaKey || e.ctrlKey) return
    if (e.key === 'Escape') {
      if (choosing) choosing = ''
      else if (back(camp)) act(back(camp))
      else return
      e.preventDefault()
    } else if (/^[1-6]$/.test(e.key) && partyPick(camp, Number(e.key) - 1)) {
      e.preventDefault()
      pick(Number(e.key) - 1)
    }
  }
</script>

<svelte:window onkeydown={onkey} />

<div class="campscreen">
  <div class="op" aria-live="polite">
    <b>{operation(camp)}</b>
    {#if v.message}<span class="msg">{v.message}</span>{/if}
  </div>

  <section class="party" aria-label="Party">
    {#each camp.party as m, i (m.name)}
      {@const p = partyPick(camp, i)}
      {@const placed = v.placed?.indexOf(i) ?? -1}
      <button class="card {condition(m)}" class:current={i === v.member} class:pickable={!!p}
        disabled={busy || !p} onclick={() => pick(i)} title={p ? `${labels[p.action]} ${m.name}` : ''}>
        <span class="line1"><b>{i + 1} {m.name}</b><span>{m.status}</span></span>
        <span class="line2"><span>{m.short}</span><span>AC {m.ac}</span><span>{m.hp}/{m.maxHp}</span></span>
        <span class="hp"><i style="width:{Math.max(0, (100 * m.hp) / Math.max(1, m.maxHp))}%"></i></span>
        {#if m.poison}<span class="poison">poisoned</span>{/if}
        {#if placed >= 0}<span class="placed">place {placed + 1}</span>{/if}
      </button>
    {/each}
  </section>

  {#if v.screen === 'menu'}
    <section class="panel">
      <p class="muted">Inspect a member: click them above, or press 1–{camp.party.length}.</p>
      <div class="row">
        {#each buttons(camp, menuOrder) as b (b.action)}
          <button class:primary={b.action === 'leave-camp'} disabled={busy || !b.enabled} onclick={() => act(b.action)}>{b.label}</button>
        {/each}
      </div>
      {#each buttons(camp, menuOrder).filter((b) => !b.enabled) as b (b.action)}
        <p class="why"><b>{b.label}:</b> {b.reason}</p>
      {/each}
    </section>
  {:else if v.screen === 'reorder'}
    <section class="panel">
      <p class="muted">Click the members in their new order. The last one takes the last place.</p>
    </section>
  {:else if member}
    <section class="panel inspect">
      <div class="sheet">
        <div class="name">{member.name}</div>
        <div>{member.race} {member.class} · {member.alignment} · Level {member.level}</div>
        <div class="muted">{member.status}{#if member.poison}{' '}(poisoned){/if} · HP {member.hp}/{member.maxHp} · AC {member.ac} · Gold {member.gold}</div>
        <h3>Items ({items.length} / 8)</h3>
        {#if items.length}
          <ol>
            {#each items as it, i (i)}
              <li>{it.name}
                {#if it.equipped}<span class="tag">equipped</span>{/if}
                {#if !it.identified}<span class="tag">unidentified</span>{/if}
                {#if it.cursed}<span class="tag bad">cursed</span>{/if}
              </li>
            {/each}
          </ol>
        {:else}<p class="muted">Carrying nothing.</p>{/if}
        <h3>Spells left</h3>
        <div class="small">Mage {slots(member.mageSpells, member.maxMage)}</div>
        <div class="small">Priest {slots(member.priestSpells, member.maxPriest)}</div>
      </div>

      <div class="work">
        {#if v.screen === 'inspect'}
          <div class="row">
            {#each inspectButtons as b (b.action)}
              <button class:on={choosing === b.action} disabled={busy || !b.enabled} onclick={() => inspectClick(b)}>{b.label}</button>
            {/each}
          </div>
          {#each inspectButtons.filter((b) => !b.enabled && b.reason) as b (b.action)}
            <p class="why"><b>{b.label}:</b> {b.reason}</p>
          {/each}
          {#if choosing === 'cast'}
            <h3>Cast which spell?</h3>
            <div class="row">
              {#each castable as sp (sp.name)}
                <button disabled={busy} onclick={() => act('cast', { spell: sp.name })}>{sp.name}{#if sp.target}{' '}…{/if}</button>
              {/each}
              <button class="small" disabled={busy} onclick={() => (choosing = '')}>Cancel</button>
            </div>
            <p class="muted small">Spells marked … are cast on one member, chosen next.</p>
            {#each whyNot(v.spells) as g (g.reason)}
              <p class="why"><b>{g.reason}</b> {g.names.join(', ')}</p>
            {/each}
          {:else if choosing === 'use'}
            <h3>Use which item?</h3>
            <div class="row">
              {#each items as it, i (i)}
                <button disabled={busy} onclick={() => act('use', { n: i + 1 })}>{i + 1}) {it.name}</button>
              {/each}
              <button class="small" disabled={busy} onclick={() => (choosing = '')}>Cancel</button>
            </div>
          {/if}
        {:else if v.screen === 'equip'}
          <h3>{v.category}</h3>
          <div class="row">
            {#each v.choices ?? [] as c (c.number)}
              <button disabled={busy} onclick={() => act('equip-choose', { n: c.number })}>{c.number}) {c.name}</button>
            {/each}
            <button disabled={busy} onclick={() => act('equip-skip')}>{labels['equip-skip']}</button>
          </div>
          {#if !v.choices?.length}<p class="muted">Nothing {member.name} carries fits here.</p>{/if}
        {:else if v.screen === 'drop' || v.screen === 'tradeItem'}
          {@const a = v.screen === 'drop' ? 'drop-item' : 'trade-item'}
          {@const end = v.screen === 'drop' ? 'end-drop' : 'end-trade'}
          <div class="row">
            {#each items as it, i (i)}
              <button disabled={busy} onclick={() => act(a, { n: i + 1 })}>{labels[a]} {i + 1}) {it.name}</button>
            {/each}
            <button class="primary" disabled={busy} onclick={() => act(end)}>{labels[end]}</button>
          </div>
        {:else if v.screen === 'trade'}
          <p class="muted">Click the member who receives, above.</p>
          <button disabled={busy} onclick={() => act('end-trade')}>{labels['end-trade']}</button>
        {:else if v.screen === 'tradeGold'}
          <form class="row" onsubmit={(e) => { e.preventDefault(); act('trade-gold', { gold: Number(gold) || 0 }); gold = 0 }}>
            <label style="margin:0"><span>Gold to give ({member.name} has {member.gold})</span>
              <input type="number" min="0" step="1" bind:value={gold} />
            </label>
            <button class="primary" type="submit" disabled={busy}>{labels['trade-gold']}</button>
            <button type="button" disabled={busy} onclick={() => act('end-trade')}>{labels['end-trade']}</button>
          </form>
          <p class="muted small">0 gives no gold and goes on to the items.</p>
        {:else if v.screen === 'spellBooks'}
          <div class="row">
            <button disabled={busy} onclick={() => act('spell-book', { priest: false })}>Mage spells</button>
            <button disabled={busy} onclick={() => act('spell-book', { priest: true })}>Priest spells</button>
            <button disabled={busy} onclick={() => act('close-book')}>{labels['close-book']}</button>
          </div>
        {:else if v.screen === 'spellList'}
          {@const book = spellBook(v.spells, v.spellBook)}
          {#if book.length}
            <table class="book">
              <tbody>
                {#each book as l (l.level)}
                  <tr><th>Level {l.level}</th><td>{#each l.spells as sp, i (sp.name)}{i ? ', ' : ''}<span class:castable={sp.castable} title={sp.reason}>{sp.name}</span>{/each}</td></tr>
                {/each}
              </tbody>
            </table>
            <p class="muted small">Highlighted spells can be cast in Camp now; hover another for why not.</p>
          {:else}<p class="muted">{member.name} knows no {v.spellBook} spells.</p>{/if}
          <button disabled={busy} onclick={() => act('close-book')}>{labels['close-book']}</button>
        {:else if v.screen === 'spellTarget'}
          <p class="muted">Click the member to cast {v.spell} on, above. Cancel keeps the spell.</p>
          <button disabled={busy} onclick={() => act('cancel-cast')}>{labels['cancel-cast']}</button>
        {:else if v.screen === 'dumapic' && v.dumapic}
          <!-- The terminal's words (render/town.go renderDumapicScreen). -->
          <div class="dumapic" role="status">
            <div><b>PARTY LOCATION:</b></div>
            <div>THE PARTY IS FACING {v.dumapic.facing}.</div>
            <div>YOU ARE {v.dumapic.east} SQUARES EAST AND {v.dumapic.north} SQUARES NORTH OF THE STAIRS
              TO THE CASTLE, AND {v.dumapic.level} LEVELS BELOW IT.</div>
          </div>
          <button class="primary" disabled={busy} onclick={() => act('leave-dumapic')}>{labels['leave-dumapic']}</button>
        {:else}
          <p class="muted">{operation(camp)}</p>
        {/if}
      </div>
    </section>
  {/if}
</div>

<style>
  .campscreen { display: grid; gap: 12px; }
  .op { display: flex; gap: 14px; align-items: baseline; flex-wrap: wrap; padding: 8px 10px; border-radius: 6px;
    background: var(--panel); border: 1px solid var(--line); }
  .msg { font-family: ui-monospace, Menlo, monospace; color: #f2c14e; }
  .party { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; }
  @media (max-width: 720px) { .party { grid-template-columns: repeat(2, 1fr); } }
  .card { position: relative; display: grid; gap: 3px; text-align: left; padding: 6px 8px; border-radius: 6px;
    background: var(--panel); border: 1px solid var(--line); color: inherit; font: inherit; }
  .card:disabled { cursor: default; opacity: 1; }
  .card.pickable { border-color: var(--accent); cursor: pointer; }
  .card.current { outline: 2px solid var(--accent); }
  .card .line1, .card .line2 { display: flex; justify-content: space-between; gap: 6px; font-size: 12px; white-space: nowrap; }
  .card .hp { display: block; height: 3px; background: #333; border-radius: 2px; }
  .card .hp i { display: block; height: 100%; background: var(--accent); border-radius: 2px; }
  .card.dead { opacity: 0.5; border-style: dashed; }
  .card.paralyzed { background: #1d2430; }
  .poison { position: absolute; bottom: 2px; right: 6px; color: #7bd36a; font-size: 10px; }
  .placed { position: absolute; bottom: 2px; left: 8px; color: var(--accent); font-size: 10px; }
  .panel { padding: 10px; border-radius: 6px; border: 1px solid var(--line); }
  .inspect { display: grid; grid-template-columns: minmax(220px, 1fr) 2fr; gap: 16px; }
  @media (max-width: 720px) { .inspect { grid-template-columns: 1fr; } }
  .name { font-size: 17px; font-weight: 600; }
  h3 { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); margin: 10px 0 4px; }
  ol { margin: 0; padding-left: 20px; }
  .tag { margin-left: 4px; font-size: 11px; color: var(--muted); border: 1px solid #444; border-radius: 3px; padding: 0 4px; }
  .tag.bad { color: var(--error); border-color: var(--error); }
  .row { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
  .work { display: grid; gap: 8px; align-content: start; justify-items: start; }
  .work .dumapic { justify-self: stretch; }
  .why { margin: 2px 0; font-size: 12px; color: var(--muted); }
  button.on { border-color: var(--accent); }
  .small { font-size: 12px; }
  .book th { text-align: left; padding-right: 10px; color: var(--muted); font-weight: 500; white-space: nowrap; }
  .book .castable { color: var(--accent); font-weight: 600; }
  .dumapic { font: 600 14px ui-monospace, Menlo, monospace; color: #f2c14e; display: grid; gap: 4px; padding: 10px;
    background: #0d0c0b; border: 1px solid var(--line); border-radius: 6px; }
</style>
