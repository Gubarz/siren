<script>
  import TextInput from '../ui/TextInput.svelte'

  // Paired host + port input. Emits a bindable `value` in "host:port" form
  // (or "" if both parts are empty) so consumers can pass it straight to a
  // cobra flag like --bind. Individual parts are also exposed via `host` and
  // `port` binds for callers that need them separately.
  let {
    value = $bindable(''),
    host = $bindable(''),
    port = $bindable(''),
    label = '',
    description = '',
    hostPlaceholder = '127.0.0.1',
    portPlaceholder = '8080',
    required = false,
  } = $props()

  // Hydrate host/port from initial `value` on mount so consumers can pass
  // either shape ("host:port" or split).
  let hydrated = false
  $effect(() => {
    if (hydrated) return
    hydrated = true
    if (value && !host && !port) {
      const idx = value.lastIndexOf(':')
      if (idx > 0) {
        host = value.slice(0, idx)
        port = value.slice(idx + 1)
      } else {
        port = value
      }
    }
  })

  // Reflect edits back to `value` in canonical form.
  $effect(() => {
    if (!hydrated) return
    if (host && port) value = `${host}:${port}`
    else if (port) value = `:${port}`
    else if (host) value = host
    else value = ''
  })
</script>

<div class="mb-2">
  <label class="block text-base font-medium text-fg mb-1" for="hp-{label}-host">
    {label}
    {#if required}<span class="text-danger-500 ml-1">*</span>{/if}
  </label>
  <div class="flex items-center gap-2">
    <div class="flex-1 min-w-0">
      <TextInput
        id="hp-{label}-host"
        type="text"
        size="sm"
        bind:value={host}
        placeholder={hostPlaceholder}
        spellcheck="false"
        autocomplete="off"
      />
    </div>
    <span class="font-medium text-fg-muted">:</span>
    <div class="flex-none w-26">
      <TextInput
        id="hp-{label}-port"
        type="number"
        size="sm"
        bind:value={port}
        placeholder={portPlaceholder}
        min="1"
        max="65535"
      />
    </div>
  </div>
  {#if description}
    <span class="block text-xs mt-1 text-fg-muted">{description}</span>
  {/if}
</div>
