<script>
  // A character's condition (R-3), the same on Inspect and the map: the
  // word Go gives, with a mark and colour from lib/condition.js.
  //
  // Props:
  //   label  the condition (app.ConditionLabel)
  import { condition } from './condition.js'

  let { label } = $props()
  const c = $derived(condition(label))
</script>

<span class="cond {c.cls}">{#if c.mark}<span class="mark" aria-hidden="true">{c.mark}</span>{/if}{c.label}</span>

<style>
  .cond { display: inline-flex; align-items: center; gap: 3px; font: 700 11px ui-monospace, Menlo, monospace; letter-spacing: 0.5px;
    border: 1px solid currentColor; border-radius: 3px; padding: 0 4px; white-space: nowrap; }
  .mark { font-weight: 400; }
  .ok { color: #4fd16a; }
  .asleep, .afraid { color: #9fd3ff; }
  .paralyzed { color: #f2c14e; border-style: dashed; }
  .stoned { color: #c9c3b6; border-style: double; border-width: 3px; }
  .dead { color: #ff7a6b; }
  .ashes { color: #b9a2a0; border-style: dotted; }
  .lost, .other { color: var(--muted); }
</style>
