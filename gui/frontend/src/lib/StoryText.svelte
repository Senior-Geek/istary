<script>
  // A story (CONTENT-1A): its Markdown file (gui/content), parsed by Go
  // (story.go) into blocks of plain text, drawn here with ordinary
  // elements. Every run is text, which Svelte escapes: nothing in a story
  // file is ever HTML or script. A challenge is drawn in the story box's
  // own gold, the colour of its title.
  //
  // Props:
  //   story   the View's story: { title, blocks }
  let { story } = $props()
</script>

{#snippet runs(text)}{#each text ?? [] as r}{#if r.bold && r.italic}<b><i>{r.text}</i></b>{:else if r.bold}<b>{r.text}</b>{:else if r.italic}<i>{r.text}</i>{:else}{r.text}{/if}{/each}{/snippet}

{#each story.blocks as b}
  {#if b.kind === 'heading'}
    {#if b.level <= 2}<h4>{@render runs(b.text)}</h4>{:else}<h5>{@render runs(b.text)}</h5>{/if}
  {:else if b.kind === 'list' && b.ordered}
    <ol>{#each b.items as item}<li>{@render runs(item)}</li>{/each}</ol>
  {:else if b.kind === 'list'}
    <ul>{#each b.items as item}<li>{@render runs(item)}</li>{/each}</ul>
  {:else if b.kind === 'quote'}
    <blockquote>{#each b.items as p}<p>{@render runs(p)}</p>{/each}</blockquote>
  {:else if b.kind === 'challenge'}
    {#each b.items as p}<p class="challenge">{@render runs(p)}</p>{/each}
  {:else}
    <p>{@render runs(b.text)}</p>
  {/if}
{/each}

<style>
  h4, h5 { margin: 4px 0 6px; color: var(--gold); font-size: 14px; font-weight: 700; }
  h5 { font-size: 13px; }
  ul, ol { margin: 0 0 8px; padding-left: 20px; }
  ul:last-child, ol:last-child, blockquote:last-child { margin-bottom: 0; }
  blockquote { margin: 0 0 8px; padding-left: 10px; border-left: 2px solid var(--glass-line); }
  .challenge { color: var(--gold); font-weight: 700; }
</style>
