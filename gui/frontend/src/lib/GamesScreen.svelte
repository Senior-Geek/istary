<script>
  // Game selection (UX-3A): what Istary opens on. Six slots, each its own
  // adventure; an empty one begins a new game (or takes the earlier save),
  // a saved one continues, and replacing one is asked twice. The slots and
  // what each allows are Go's (view.games); the words are games.js's.
  //
  // Props:
  //   view  the Go View (view.screen is "games")
  //   busy  a request is running
  //   call  (fn, ...args) => Promise   runs a Session method
  import * as S from '../../wailsjs/go/main/Session.js'
  import { summary, choices, heading, state } from './games.js'

  let { view, busy = false, call } = $props()
  const games = $derived(view.games)

  const acts = {
    new: S.NewGame,
    continue: S.ContinueGame,
    import: S.ImportEarlierGame,
    replace: S.ReplaceGame,
    confirmReplace: S.ConfirmReplaceGame,
  }
  function act(c, slot) {
    if (c.act === 'cancelReplace') call(S.CancelReplaceGame)
    else call(acts[c.act], slot.slot)
  }
</script>

<section class="games" aria-label="Game selection">
  <header>
    <h2>Choose a game</h2>
    <p class="muted">Each game is its own adventure: its own characters, party, maze and map. Everything is saved as it happens.</p>
  </header>

  {#if games.legacy}
    <div class="legacy" class:bad={!!games.legacy.problem}>
      <div><b>A game saved by an earlier Istary</b> <span class="muted">saved {games.legacy.saved}</span></div>
      {#if games.legacy.problem}
        <span>cannot be imported: {games.legacy.problem}. It is left as it is.</span>
      {:else}
        <span>{summary(games.legacy).join(' · ')}.</span>
        <span class="muted">Import it into an empty slot below; the original is kept as it is.</span>
      {/if}
    </div>
  {/if}

  <ol class="slots">
    {#each games.slots as slot (slot.slot)}
      <li class="slot {slot.state}" class:replacing={games.replacing === slot.slot}>
        <div class="head">
          <span class="name">{heading(slot)}</span>
          <span class="badge">{state(slot)}</span>
        </div>
        <div class="body">
          {#if slot.state === 'game'}
            {#each summary(slot) as line}<div>{line}</div>{/each}
            <div class="muted">Saved {slot.saved}{#if slot.created}{' · begun '}{slot.created}{/if}</div>
          {:else if slot.state === 'damaged'}
            <div class="problem">This game cannot be opened: {slot.problem}. It is left as it is.</div>
          {:else}
            <div class="muted">No game in this slot.</div>
          {/if}
          {#if games.replacing === slot.slot}
            <div class="confirm" role="alertdialog" aria-label="Replace game {slot.slot}">
              Replace game {slot.slot} with a new game? Its characters, party and maze are set aside in the save
              folder's replaced folder, not deleted, but this slot will no longer hold them.
            </div>
          {/if}
        </div>
        <div class="choices">
          {#each choices(slot, games) as c}
            <button class={c.kind} disabled={busy} onclick={() => act(c, slot)}>{c.label}</button>
          {/each}
        </div>
      </li>
    {/each}
  </ol>
  <p class="muted where">Saved in {view.saveDir}</p>
</section>

<style>
  .games { max-width: 900px; padding: 4px 8px 12px; }
  header h2 { margin: 0 0 2px; font-size: 18px; }
  header p { margin: 0 0 10px; }
  .legacy { border: 1px solid var(--notice); border-radius: 6px; padding: 6px 10px; margin-bottom: 10px; display: grid; gap: 2px; }
  .legacy.bad { border-color: var(--error); }
  .slots { list-style: none; padding: 0; margin: 0; display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 8px; }
  .slot { background: var(--card); border: 1px solid var(--line); border-radius: 6px; padding: 8px 10px; display: grid; grid-template-rows: auto 1fr auto; gap: 6px; min-height: 120px; }
  .slot.empty { background: var(--card-back); border-style: dashed; }
  .slot.damaged { border-color: var(--error); }
  .slot.replacing { border-color: var(--error); border-style: solid; }
  .head { display: flex; justify-content: space-between; align-items: baseline; gap: 6px; }
  .name { font-weight: 600; }
  .badge { font-size: 11px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.05em; }
  .slot.game .badge { color: var(--accent); }
  .slot.damaged .badge { color: var(--error); }
  .body { font-size: 13px; display: grid; gap: 2px; align-content: start; }
  .problem { color: var(--error); }
  .confirm { margin-top: 4px; color: var(--btn-danger-text); }
  .choices { display: flex; gap: 6px; flex-wrap: wrap; }
  .choices button { margin: 0; }
  .where { font-size: 11px; margin: 10px 0 0; word-break: break-all; }
</style>
