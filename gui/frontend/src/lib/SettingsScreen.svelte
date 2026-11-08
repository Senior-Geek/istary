<script>
  // Settings (Castle-0 C0-4): Istary's own utilities, reached from the Edge
  // of Town as the original's U)TILITIES are, and left for the Castle. The
  // save folder, reloading the roster from it, and resetting or restoring
  // the roster from a backup. The roster operations are Go's; reset and
  // restore back up first.
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

  let { view, busy = false, call, onopen, onroster } = $props()

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
  .path { font-family: ui-monospace, Menlo, monospace; font-size: 12px; word-break: break-all; margin: 0 0 8px; }
</style>
