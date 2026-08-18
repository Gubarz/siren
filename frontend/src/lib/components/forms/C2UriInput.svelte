<script>
  import Button from '$components/ui/Button.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import Menu from '$components/ui/Menu.svelte'
  import MenuItem from '$components/ui/MenuItem.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import { formatListenerC2 } from '$utils/listeners.js'

  let {
    value = $bindable(''),
    listeners = [],
    placeholder = 'mtls://10.0.0.1:443',
    disabled = false,
    serverHost = '',
    onchange,
    onproto,
    onlistener,
  } = $props()

  const PROTO_PRESETS = ['mtls://', 'https://', 'http://', 'dns://', 'wg://', 'tcp-pivot://', 'namedpipe://']

  let protoOpen = $state(false)
  let listenerOpen = $state(false)

  function protocolLabel(url) {
    return (url?.match(/^([a-z-]+):\/\//i) || ['', 'proto'])[1]
  }

  function setProto(prefix) {
    const rest = (value || '').replace(/^[a-z-]+:\/\//i, '')
    value = `${prefix}${rest}`
    protoOpen = false
    onproto?.(prefix)
    onchange?.(value)
  }

  function pickListener(listener) {
    const formatted = formatListenerC2(listener, serverHost)
    value = formatted
    listenerOpen = false
    onlistener?.(listener)
    onchange?.(value)
  }

  function handleInput(event) {
    value = event.currentTarget.value
    onchange?.(value)
  }
</script>

<div class="flex items-center gap-2">
  <div class="relative inline-flex items-center">
    <Button color="dark" size="xs" class="!font-mono" title="Set protocol" {disabled}>
      <span class="lowercase">{protocolLabel(value)}</span>
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

  <div class="flex-1 min-w-0">
    <TextInput
      size="sm"
      {value}
      oninput={handleInput}
      {placeholder}
      {disabled}
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
      {disabled}
      title={listeners.length === 0 ? 'No active listeners' : 'Pick from an active listener'}
    >
      Listener
      <Icon name="chevron-down" size={10} />
    </Button>
    <Menu bind:isOpen={listenerOpen} placement="bottom-end" minWidth="15rem">
      {#if listeners.length === 0}
        <div class="px-3 py-2 text-center text-xs text-fg-muted">No active listeners</div>
      {:else}
        {#each listeners as listener}
          <MenuItem onclick={() => pickListener(listener)} class="justify-between">
            <span class="uppercase font-mono">{listener.protocol}</span>
            <span class="font-mono text-fg-muted">{listener.host || '<server>'}{listener.protocol === 'dns' ? '' : `:${listener.port}`}</span>
            {#if listener.name}<span class="truncate text-xs text-fg-muted">{listener.name}</span>{/if}
          </MenuItem>
        {/each}
      {/if}
    </Menu>
  </div>
</div>
