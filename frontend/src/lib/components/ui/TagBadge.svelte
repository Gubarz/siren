<script>
  import IconButton from './IconButton.svelte'
  import { parseTag, getTagCategoryStyle } from '../../utils/tags.js'

  let {
    tag = '',
    removable = false,
    onremove = () => {},
    class: className = '',
  } = $props()

  let parsed = $derived(parseTag(tag))
  let style = $derived(parsed.isTyped ? getTagCategoryStyle(parsed.key) : null)
</script>

{#if parsed.isTyped}
  <span class="inline-flex items-center rounded border text-3xs font-mono overflow-hidden shadow-xs shrink-0 select-none {style.container} {className}">
    <span class="px-1 py-0.5 font-semibold uppercase tracking-wider text-4xs leading-none {style.keyBg}">
      {parsed.key}
    </span>
    <span class="px-1.5 py-0.5 font-mono leading-none flex items-center gap-1">
      {parsed.value}
      {#if removable}
        <IconButton icon="x" label="Remove tag" size="xs" onclick={onremove} class="ml-0.5" />
      {/if}
    </span>
  </span>
{:else}
  <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded border border-line bg-surface-200 text-fg-muted text-3xs font-mono shadow-xs shrink-0 select-none {className}">
    <span>#{parsed.value}</span>
    {#if removable}
      <IconButton icon="x" label="Remove tag" size="xs" onclick={onremove} />
    {/if}
  </span>
{/if}
