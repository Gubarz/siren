<script>
  import { onMount } from 'svelte'
  import { connection } from '$stores/connection.svelte.js'
  import { Connect, GetClientConfigs } from '../../api/connection.js'
  import Select from '$components/ui/Select.svelte'
  import Button from '$components/ui/Button.svelte'
  import { errorMessage } from '../../utils/errors.js'

  let { onconnected } = $props()

  let configs = $state([])
  let loading = $state(true)
  let error = $state('')
  let selectedConfig = $state('')
  let connecting = $state(false)
  const connectTimeoutMs = 15000

  onMount(async () => {
    try {
      configs = await GetClientConfigs()
      if (!configs || configs.length === 0) {
        error = 'No sliver profiles found in ~/.sliver-client/configs'
        loading = false
        return
      }
      selectedConfig = configs[0]
      loading = false
      if (configs.length === 1 && !connection.intentionallyDisconnected) {
        await connect(selectedConfig)
      }
      connection.intentionallyDisconnected = false
    } catch (e) {
      error = errorMessage(e, 'Failed to load configs: ')
      loading = false
    }
  })

  async function connect(profile) {
    if (connecting) return
    connecting = true
    error = ''
    try {
      await withTimeout(Connect(profile), connectTimeoutMs, 'Connection timed out after 15 seconds')
      onconnected?.(profile)
    } catch (e) {
      error = errorMessage(e, 'Connection failed: ')
      connecting = false
      loading = false
    }
  }

  function withTimeout(promise, ms, message) {
    let timeout
    const timer = new Promise((_, reject) => {
      timeout = setTimeout(() => reject(new Error(message)), ms)
    })
    return Promise.race([promise, timer]).finally(() => clearTimeout(timeout))
  }
</script>

<div class="fixed inset-0 z-10000 flex items-center justify-center bg-canvas">
  <div class="bg-panel border border-panel-border rounded-lg p-7 w-88 max-w-pct-90 shadow-2xl text-center">
    <div class="mb-6">
      <img src="/wails.png" alt="Sliver Logo" class="w-20 h-20 mb-2 mx-auto" />
      <h2 class="m-0 text-fg font-light text-xl">Siren</h2>
    </div>

    {#if connection.reconnecting}
      <div class="text-fg-muted mb-6 animate-pulse">
        Connection lost.<br />
        Reconnecting to <strong class="text-fg">{connection.profile}</strong>...
      </div>
      <Button color="secondary" size="sm" fullWidth onclick={() => { connection.reconnecting = false }}>
        Cancel
      </Button>

    {:else if loading}
      <div class="text-fg-muted">Loading profiles...</div>
    {:else if error}
      <div class="text-danger-500 bg-danger-500/10 p-2 rounded mb-5 text-left text-sm wrap-break-word">{error}</div>
      <Button color="primary" size="sm" onclick={() => window.location.reload()}>Retry</Button>
    {:else}
      <div class="text-left mb-5">
        <label for="teamserver-profile" class="block mb-2 text-fg-muted text-sm">Select Teamserver Profile</label>
        <Select id="teamserver-profile" bind:value={selectedConfig} options={configs.map(p => ({value: p, label: p}))} disabled={connecting} />
      </div>

      <Button color="primary" size="sm" fullWidth onclick={() => connect(selectedConfig)} disabled={connecting}>
        {connecting ? 'Connecting...' : 'Connect'}
      </Button>
    {/if}
  </div>
</div>
