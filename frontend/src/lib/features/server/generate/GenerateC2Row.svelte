<script>
  import Icon from '$components/ui/Icon.svelte'
  import Button from '$components/ui/Button.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import Menu from '$components/ui/Menu.svelte'
  import MenuItem from '$components/ui/MenuItem.svelte'
  import TextInput from '$components/ui/TextInput.svelte'

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

  const PROTO_PRESETS = ['mtls://', 'https://', 'http://', 'dns://', 'wg://', 'tcp-pivot://', 'namedpipe://']

  let protoOpen = $state(false)
  let listenerOpen = $state(false)

  function protocolLabel(url) {
    return (url?.match(/^([a-z-]+):\/\//i) || ['', 'proto'])[1]
  }

  function setProto(prefix) {
    onproto?.(index, prefix)
    protoOpen = false
  }

  function pickListener(listener) {
    onlistener?.(index, listener)
    listenerOpen = false
  }
</script>

<div class="mb-2 grid grid-c2-row items-center gap-2">
  <span class="text-center font-mono text-xs text-fg-muted">{index + 1}</span>

  <div class="relative inline-flex items-center">
    <Button color="dark" size="xs" class="!font-mono" title="Set protocol">
      <span class="lowercase">{protocolLabel(c2Url)}</span>
      <Icon name="chevron-down" size={10} />
    </Button>
    <Menu bind:isOpen={protoOpen} minWidth="9rem">
      {#each PROTO_PRESETS as prefix}
        <MenuItem onclick={() => setProto(prefix)}>
          <span class="font-mono">{prefix}</span>
        </MenuItem>
      {/each}
    </Menu>
  </div>

  <div class="min-w-0">
    <TextInput
      size="sm"
      value={c2Url}
      oninput={(event) => onupdate?.(index, event.currentTarget.value)}
      placeholder="mtls://10.0.0.1:443"
      spellcheck="false"
      autocomplete="off"
      class="font-mono"
    />
  </div>

  <div class="relative inline-flex items-center">
    <Button
      color="dark"
      size="xs"
      icon="headphones"
      aria-haspopup="true"
      aria-expanded={listenerOpen}
      title={c2Listeners.length === 0 ? 'No active listeners' : 'Pick from an active listener'}
    >
      Listener
      <Icon name="chevron-down" size={10} />
    </Button>
    <Menu bind:isOpen={listenerOpen} placement="bottom-end" minWidth="15rem">
      {#if c2Listeners.length === 0}
        <div class="px-3 py-2 text-center text-xs text-fg-muted">No active listeners</div>
      {:else}
        {#each c2Listeners as listener}
          <MenuItem onclick={() => pickListener(listener)} class="justify-between">
            <span class="uppercase font-mono">{listener.protocol}</span>
            <span class="font-mono text-fg-muted">{listener.host || '<server>'}{listener.protocol === 'dns' ? '' : `:${listener.port}`}</span>
            {#if listener.name}<span class="truncate text-xs text-fg-muted">{listener.name}</span>{/if}
          </MenuItem>
        {/each}
      {/if}
    </Menu>
  </div>

  <IconButton icon="chevron-up" label="Higher priority" tooltip="Higher priority" onclick={() => onmoveup?.(index)} disabled={index === 0} />
  <IconButton icon="chevron-down" label="Lower priority" tooltip="Lower priority" onclick={() => onmovedown?.(index)} disabled={index === total - 1} />
  <IconButton icon="x" label="Remove" tooltip="Remove" color="red" onclick={() => onremove?.(index)} disabled={total === 1} />
</div>
