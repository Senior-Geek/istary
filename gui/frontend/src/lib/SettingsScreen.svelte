<script>
  // Settings (Castle-0 C0-4): Istary's own utilities, reached from the Edge
  // of Town as the original's U)TILITIES are, and left for the Castle. The
  // save folder, reloading the roster from it, and resetting or restoring
  // the roster from a backup. The roster operations are Go's; reset and
  // restore back up first.
  //
  // And the illustrated places' appearance (ART-2C): their panels, their
  // story box and their picture, each chosen here and taking effect at
  // once. They are the window's settings (Session.SetAppearance, kept in
  // gui-settings.json): the same in every game, and in none of them.
  //
  // Props:
  //   view     the Go View (view.location is "settings")
  //   busy     a request is running
  //   call     (fn, ...args) => Promise   runs a Session method
  //   onopen   (rosterIndex) => void      open a party member's sheet
  //   onroster () => void                 the roster was replaced (close any sheet)
  import * as S from '../../wailsjs/go/main/Session.js'
  import LocationFrame from './LocationFrame.svelte'
  import RestorePanel from './RestorePanel.svelte'
  import { looks, lookOf } from './scenes.js'

  let { view, busy = false, call, onopen, onroster } = $props()

  const look = $derived(lookOf(view.settings))
  const parts = [
    { key: 'panels', legend: 'Panels', hint: 'The working panels: the actions, the information and the party.' },
    { key: 'story', legend: 'Story box', hint: 'The place’s story, over the picture.' },
    { key: 'artwork', legend: 'Picture', hint: 'Subdued is darker and quieter; Hidden shows none.' },
  ]
  const choose = (key, id) => {
    const next = { ...look, [key]: id }
    call(S.SetAppearance, next.panels, next.story, next.artwork)
  }

  let resetting = $state(false) // reset confirmation open
  let backups = $state(null) // backup list while the restore panel is open

  async function reset() {
    await call(S.ResetRoster)
    resetting = false
    onroster?.()
  }

  async function openRestore() {
    resetting = false
    backups = await S.ListBackups()
  }

  async function restore(file) {
    await call(S.RestoreBackup, file)
    if (!view.error) {
      backups = null
      onroster?.()
    }
  }
</script>

<LocationFrame {view} {busy} {call} {onopen}>
  {#snippet actions()}
    <button class="wide" disabled={busy} onclick={() => call(S.ReloadRoster)}>Reload from disk</button>
    <button class="wide danger" disabled={busy} onclick={() => { resetting = true; backups = null }}>Reset roster…</button>
    <button class="wide" disabled={busy} onclick={openRestore}>Restore roster…</button>
  {/snippet}

  <section class="appearance" aria-labelledby="appearance-title">
    <h2 id="appearance-title">Appearance</h2>
    <p class="muted">For the places with pictures, the Castle and the Edge of Town. Kept for this window
      in every game; no game is changed.</p>
    <div class="parts">
      {#each parts as p (p.key)}
        <fieldset class="part" disabled={busy}>
          <legend>{p.legend}</legend>
          <div class="choices">
            {#each looks[p.key] as o (o.id)}
              <label class="choice" class:on={look[p.key] === o.id}>
                <input type="radio" name="look-{p.key}" value={o.id} checked={look[p.key] === o.id}
                  onchange={() => choose(p.key, o.id)} />{o.label}{#if o.id === looks[p.key][0].id}<span class="dflt"> (default)</span>{/if}
              </label>
            {/each}
          </div>
          <p class="muted hint">{p.hint}</p>
        </fieldset>
      {/each}
    </div>
  </section>

  <section>
    <h2>Save folder</h2>
    <p class="path">{view.saveDir}</p>
    <p class="muted">The roster, the party and any expedition are saved here after every action. Backups are
      kept in its <b>backups</b> folder; the window's own settings in <b>gui-settings.json</b>.</p>
    {#if resetting}
      <div class="banner message confirm">
        Back up the current roster to <b>{view.saveDir}/backups</b> and start an empty roster?
        <div class="row" style="margin-top:8px">
          <button class="danger" disabled={busy} onclick={reset}>Back up and reset</button>
          <button disabled={busy} onclick={() => { resetting = false }}>Cancel</button>
        </div>
      </div>
    {/if}
  </section>
  {#if backups}
    <section>
      <h2>Restore roster</h2>
      <RestorePanel {backups} {busy} onrestore={restore} oncancel={() => { backups = null }} />
    </section>
  {/if}
</LocationFrame>

<style>
  section { margin-top: 0; }
  section + section { margin-top: 12px; }
  .wide { width: 100%; text-align: left; margin: 0 0 4px; }
  .parts { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 8px 16px; }
  .part { margin: 0; padding: 0; border: 0; min-width: 0; }
  legend { padding: 0; margin: 0 0 4px; font-size: 12px; text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted); }
  .choices { display: flex; flex-wrap: wrap; gap: 4px; }
  .choice { display: inline-flex; align-items: center; gap: 4px; margin: 0; padding: 3px 10px 3px 6px; border-radius: 4px; background: var(--btn); color: var(--btn-text); cursor: pointer; }
  .choice.on { background: var(--selected); }
  .choice input { margin: 0; padding: 0; accent-color: var(--focus); }
  .choice:has(input:focus-visible) { outline: 2px solid var(--focus); outline-offset: 2px; }
  .dflt { color: var(--muted); font-size: 11px; width: auto; }
  .hint { margin: 4px 0 0; }
  .path { font-family: ui-monospace, Menlo, monospace; font-size: 12px; word-break: break-all; margin: 0 0 8px; }
</style>
