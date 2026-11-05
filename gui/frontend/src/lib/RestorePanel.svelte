<script>
  // Lists roster backups (from Go) and asks for confirmation before
  // restoring one. Restoring is done by the caller's onrestore(file).
  let { backups, busy = false, onrestore, oncancel } = $props()

  let chosen = $state('')
  const pick = $derived(backups.find((b) => b.file === chosen))
</script>

<div class="restore">
  {#if !backups.length}
    <p class="muted">No backups yet. Reset roster makes one, and so does every restore.</p>
  {:else}
    <table>
      <thead><tr><th></th><th>Backup</th><th>Date</th><th class="num">Characters</th></tr></thead>
      <tbody>
        {#each backups as b}
          <tr class:pick={b.readable} class:selected={b.file === chosen}
            onclick={() => b.readable && (chosen = b.file)}>
            <td><input type="radio" name="backup" value={b.file} bind:group={chosen} disabled={!b.readable} /></td>
            <td>{b.file}</td>
            <td>{b.date}</td>
            <td class="num">{b.readable ? b.characters : '—'}</td>
          </tr>
          {#if b.file === chosen && b.names?.length}
            <tr class="names"><td></td><td colspan="3" class="muted">{b.names.join(', ')}</td></tr>
          {/if}
          {#if !b.readable}
            <tr class="names"><td></td><td colspan="3" class="bad">Cannot be restored: {b.problem}.</td></tr>
          {/if}
        {/each}
      </tbody>
    </table>
  {/if}

  <div class="banner message confirm">
    {#if pick}
      Replace the current roster with <b>{pick.file}</b> ({pick.characters} character{pick.characters === 1 ? '' : 's'})?
      The current roster is backed up first.
    {:else}
      Choose a backup to restore.
    {/if}
    <div class="row" style="margin-top:8px">
      <button class="danger" disabled={busy || !pick} onclick={() => onrestore?.(chosen)}>Restore</button>
      <button disabled={busy} onclick={() => oncancel?.()}>Cancel</button>
    </div>
  </div>
</div>

<style>
  .restore table { margin-top: 6px; }
  tr.names td { border-bottom: none; padding-top: 0; font-size: 12px; }
  td.bad { color: var(--error); font-size: 12px; }
  tr.pick { cursor: pointer; }
</style>
