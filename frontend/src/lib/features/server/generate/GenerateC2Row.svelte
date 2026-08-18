<script>
  import IconButton from '$components/ui/IconButton.svelte'
  import C2UriInput from '$components/forms/C2UriInput.svelte'

  let {
    c2Url = '',
    index = 0,
    total = 1,
    c2Listeners = [],
    onupdate,
    onproto,
    onlistener,
    onmoveup,
    onmovedown,
    onremove,
  } = $props()
</script>

<div class="mb-2 flex items-center gap-2">
  <span class="w-7 text-center font-mono text-xs text-fg-muted">{index + 1}</span>

  <div class="flex-1 min-w-0">
    <C2UriInput
      value={c2Url}
      listeners={c2Listeners}
      onchange={(val) => onupdate?.(index, val)}
      onproto={(prefix) => onproto?.(index, prefix)}
      onlistener={(listener) => onlistener?.(index, listener)}
    />
  </div>

  <IconButton icon="chevron-up" label="Higher priority" tooltip="Higher priority" onclick={() => onmoveup?.(index)} disabled={index === 0} />
  <IconButton icon="chevron-down" label="Lower priority" tooltip="Lower priority" onclick={() => onmovedown?.(index)} disabled={index === total - 1} />
  <IconButton icon="x" label="Remove" tooltip="Remove" color="red" onclick={() => onremove?.(index)} disabled={total === 1} />
</div>
