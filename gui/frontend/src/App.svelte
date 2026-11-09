<script>
  // The page only renders the Go-side View and calls Session methods. Every
  // rule (eligibility, point limits, messages, where each action belongs)
  // comes from Go. In town each place has its own screen in the shared
  // location frame (Castle-0); the maze, Camp and combat have theirs.
  import * as S from '../wailsjs/go/main/Session.js'
  import FloatingPanel from './lib/FloatingPanel.svelte'
  import CharacterSheet from './lib/CharacterSheet.svelte'
  import CombatScreen from './lib/CombatScreen.svelte'
  import DungeonScreen from './lib/DungeonScreen.svelte'
  import LocationFrame from './lib/LocationFrame.svelte'
  import CastleScreen from './lib/CastleScreen.svelte'
  import EdgeScreen from './lib/EdgeScreen.svelte'
  import TrainingScreen from './lib/TrainingScreen.svelte'
  import TavernScreen from './lib/TavernScreen.svelte'
  import SettingsScreen from './lib/SettingsScreen.svelte'
  import TradingScreen from './lib/TradingScreen.svelte'
  import InnScreen from './lib/InnScreen.svelte'

  let view = $state(null)
  let busy = $state(false)
  let selected = $state(-1) // roster index shown in the character sheet

  async function call(fn, ...args) {
    busy = true
    try {
      view = await fn(...args)
    } catch (e) {
      view = { ...view, error: String(e) }
    } finally {
      busy = false
    }
  }

  call(S.State)

  const sel = $derived(view?.roster?.find((ch) => ch.index === selected))
  // Town and the dungeon (with Camp) fill the window, the party bar in view.
  const shell = $derived(view && !view.fatal && view.screen !== 'combat')

  const select = (index) => { selected = selected === index ? -1 : index }
  const open = (index) => { selected = index }

  // Escape in town is "← Castle" (ReturnToCastle), as in the frame's head;
  // an open character sheet takes Escape first. The maze, Camp and combat
  // keep their own keys.
  function onkey(e) {
    if (e.key !== 'Escape' || e.defaultPrevented || busy || !view?.return?.open) return
    if (view.screen === 'combat' || view.screen === 'dungeon' || (sel && view.screen === 'roster')) return
    e.preventDefault()
    call(S.ReturnToCastle)
  }

  async function renameSelected(name) {
    await call(S.RenameCharacter, selected, name)
  }

  async function deleteSelected(password) {
    await call(S.DeleteCharacter, sel.name, password)
    if (!view.error) selected = -1
  }
</script>

<svelte:window onkeydown={onkey} />

<main class:shell>
  {#if view}
    {#if !shell}<h1>Istary — Wizardry I</h1>{/if}

    {#if view.fatal}
      <div class="muted">Saves: {view.saveDir}</div>
      <div class="banner error">{view.fatal}
        <button class="small" disabled={busy} onclick={() => call(S.TryAgain)}>Try again</button>
      </div>
    {:else}
      <div class="banners">
        {#if view.error}<div class="banner error">{view.error}</div>{/if}
        {#if view.unsaved}
          <!-- What failed to be saved has happened in this window, once; trying
               again writes it as it is (SaveNow), it does not repeat it. -->
          <div class="banner error">
            {#if !view.error}Not saved: {view.unsaved}.{' '}{/if}What this window shows is not saved yet: closing
            Istary now would lose it. The next change saves everything, or
            <button class="small" disabled={busy} onclick={() => call(S.SaveNow)}>Try saving again</button>
          </div>
        {/if}
        {#if view.warning}<div class="banner error">{view.warning}</div>{/if}
        {#if view.recovery}
          <div class="banner error">
            The saved expedition cannot be resumed: {view.recovery}
            <div class="row">
              <button class="small" disabled={busy} onclick={() => call(S.TryResumeAgain)}>Try resuming again</button>
              {#if view.canReturn}
                <button class="small" disabled={busy} onclick={() => call(S.ReturnExpeditionToCastle)}>Return its party to the castle</button>
              {/if}
            </div>
          </div>
        {/if}
        <!-- At Boltac's and the Inn the receipt says it, saved or not, and at
             the Tavern the gold's. -->
        {#if view.notice && !view.shop?.last && !view.inn?.last && !view.gold}<div class="banner notice">{view.notice}</div>{/if}
        {#if view.settings?.note}<div class="banner error">{view.settings.note}</div>{/if}
      </div>

      {#if view.screen === 'combat' && view.combat}
        <CombatScreen {view} {busy} {call} />
      {:else if view.screen === 'dungeon' && view.dungeon}
        <div class="place"><DungeonScreen {view} {busy} {call} /></div>
      {:else}
        <div class="place">
          {#if view.location === 'castle'}
            <CastleScreen {view} {busy} {call} onopen={open} />
          {:else if view.location === 'edge'}
            <EdgeScreen {view} {busy} {call} onopen={open} />
          {:else if view.location === 'training'}
            <TrainingScreen {view} {busy} {call} {selected} onselect={select} onopen={open} />
          {:else if view.location === 'tavern'}
            <TavernScreen {view} {busy} {call} {selected} onselect={select} onopen={open} />
          {:else if view.location === 'trading'}
            <TradingScreen {view} {busy} {call} onopen={open} />
          {:else if view.location === 'inn'}
            <InnScreen {view} {busy} {call} onopen={open} />
          {:else if view.location === 'settings'}
            <SettingsScreen {view} {busy} {call} onopen={open} onroster={() => { selected = -1 }} />
          {:else}
            <LocationFrame {view} {busy} {call} onopen={open} />
          {/if}
        </div>

        {#if sel && view.screen === 'roster'}
          {#key sel.index}
            <FloatingPanel title="Character — {sel.name}" onclose={() => { selected = -1 }}>
              <!-- The sheet offers what the place allows: renaming and deleting
                   at the Training Grounds, joining and leaving at the Tavern. -->
              <CharacterSheet character={sel} statNames={view.statNames} nameMax={view.nameMax}
                {busy} error={view.error}
                onrename={view.location === 'training' ? renameSelected : undefined}
                ondelete={view.location === 'training' ? deleteSelected : undefined}
                onjoin={view.location === 'tavern' ? () => call(S.JoinParty, sel.index) : undefined}
                onleave={view.location === 'tavern' ? () => call(S.LeaveParty, sel.partySlot - 1) : undefined}
                onequip={(n, equip) => call(S.EquipAction, { index: sel.index, n, equip })} />
            </FloatingPanel>
          {/key}
        {/if}
      {/if}
    {/if}
  {:else}
    <p class="muted">Loading…</p>
  {/if}
</main>
