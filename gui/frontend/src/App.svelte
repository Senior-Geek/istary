<script>
  // The page only renders the Go-side View and calls Session methods. Every
  // rule (eligibility, point limits, messages, where each action belongs)
  // comes from Go. In town each place has its own screen in the shared
  // location frame (Castle-0); the maze, Camp and combat have theirs. A
  // character's sheet is shown in the frame, in place of the artwork
  // (UX-3B): this page keeps whose it is and whether it is shown, and gives
  // both to the frame (lib/sheet.js SHEET).
  import { setContext } from 'svelte'
  import * as S from '../wailsjs/go/main/Session.js'
  import { SHEET, available } from './lib/sheet.js'
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
  import TempleScreen from './lib/TempleScreen.svelte'
  import GamesScreen from './lib/GamesScreen.svelte'

  let view = $state(null)
  let busy = $state(false)
  let selected = $state(-1) // roster index of the character sheet's character, or -1
  let sheetShown = $state(false) // the sheet is shown in place of the artwork

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

  // Town and the dungeon (with Camp) fill the window, the party bar in view.
  const shell = $derived(view && !view.fatal && view.screen !== 'combat' && view.screen !== 'games')

  // A roster row: show that character's sheet, or hide it if it is the
  // one shown. A party member: show theirs.
  const select = (index) => {
    if (sheetShown && selected === index) sheetShown = false
    else open(index)
  }
  const open = (index) => { selected = index; sheetShown = true }
  const forget = () => { selected = -1; sheetShown = false }

  setContext(SHEET, {
    get index() { return selected },
    get shown() { return sheetShown },
    show: open,
    hide: () => { sheetShown = false },
    forget,
    // follow makes index the sheet's character without showing it: a place
    // whose party bar picks (Boltac's customer, the Inn's guest) keeps the
    // sheet on the member it is serving.
    follow: (index) => { if (index >= 0) selected = index },
  })

  // Escape in town is "← Castle" (ReturnToCastle), as in the frame's head;
  // a shown character sheet takes Escape first and is hidden. The maze,
  // Camp and combat keep their own keys.
  function onkey(e) {
    if (e.key !== 'Escape' || e.defaultPrevented || busy) return
    if (view?.screen === 'combat' || view?.screen === 'dungeon') return
    if (sheetShown && available(view)) {
      e.preventDefault()
      sheetShown = false
      return
    }
    if (!view?.return?.open) return
    e.preventDefault()
    call(S.ReturnToCastle)
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
        {#if view.notice && !view.shop?.last && !view.inn?.last && !view.gold && !view.temple?.last}<div class="banner notice">{#if view.title}<strong class="title">{view.title}</strong>{/if}{view.notice}</div>{/if}
        {#if view.settings?.note}<div class="banner error">{view.settings.note}</div>{/if}
      </div>

      {#if view.screen === 'games' && view.games}
        <GamesScreen {view} {busy} {call} />
      {:else if view.screen === 'combat' && view.combat}
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
            <TrainingScreen {view} {busy} {call} selected={sheetShown ? selected : -1} onselect={select} onopen={open} />
          {:else if view.location === 'tavern'}
            <TavernScreen {view} {busy} {call} selected={sheetShown ? selected : -1} onselect={select} onopen={open} />
          {:else if view.location === 'trading'}
            <TradingScreen {view} {busy} {call} onopen={open} />
          {:else if view.location === 'inn'}
            <InnScreen {view} {busy} {call} onopen={open} />
          {:else if view.location === 'temple'}
            <TempleScreen {view} {busy} {call} onopen={open} />
          {:else if view.location === 'settings'}
            <SettingsScreen {view} {busy} {call} onopen={open} onroster={forget} />
          {:else}
            <LocationFrame {view} {busy} {call} onopen={open} />
          {/if}
        </div>

      {/if}
    {/if}
  {:else}
    <p class="muted">Loading…</p>
  {/if}
</main>
