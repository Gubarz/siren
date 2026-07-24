<script>
  import { onMount } from 'svelte'
  import Modal from '../../../components/patterns/Modal.svelte'
  import Button from '../../../components/ui/Button.svelte'
  import TextField from '../../../components/forms/TextField.svelte'
  import TextInput from '../../../components/ui/TextInput.svelte'
  import Icon from '../../../components/ui/Icon.svelte'
  import Menu from '../../../components/ui/Menu.svelte'
  import MenuItem from '../../../components/ui/MenuItem.svelte'
  import { ReconfigureAgent } from '../../../api/operatorControls.js'
  import { GetServerInfo } from '../../../api/server.js'
  import { errorMessage } from '../../../utils/errors.js'
  import { jobs } from '../../../stores/resources/jobs.svelte.js'
  import { useResource } from '../../../stores/lib/createResource.svelte.js'

  useResource(jobs)

  let {
    open = $bindable(false),
    onclose = () => {},
    agent = null,
  } = $props()

  let isBeacon = $derived(agent?._kind === 'beacon')
  let reconnectInterval = $state('')
  let beaconInterval = $state('')
  let beaconJitter = $state('')
  let c2Uri = $state('')
  let submitting = $state(false)
  let error = $state('')

  const C2_PROTOCOLS = ['mtls', 'http', 'https', 'dns', 'wg']
  const PROTO_PRESETS = ['mtls://', 'https://', 'http://', 'dns://', 'wg://', 'tcp-pivot://', 'namedpipe://']

  let listenerOpen = $state(false)
  let protoOpen = $state(false)

  let c2Listeners = $derived.by(() => {
    const list = jobs?.data || []
    return list
      .map((job) => {
        const protocol = listenerProtocol(job)
        return {
          id: job.ID ?? job.id,
          name: job.Name ?? job.name,
          protocol,
          port: job.Port ?? job.port,
          host: listenerHost(job, serverHost),
          domains: job.Domains ?? job.domains ?? [],
        }
      })
      .filter((l) => C2_PROTOCOLS.includes(l.protocol))
  })

  $effect(() => {
    if (open && agent) {
      reconnectInterval = String(Math.floor((agent.ReconnectInterval ?? 0) / 1e9))
      beaconInterval = isBeacon ? String(Math.floor((agent.Interval ?? 0) / 1e9)) : ''
      beaconJitter = isBeacon ? String(Math.floor((agent.Jitter ?? 0) / 1e9)) : ''
      c2Uri = isBeacon ? (agent.ActiveC2 || '') : ''
      error = ''
    }
  })

  let serverHost = $state('')

  onMount(async () => {
    jobs.refresh?.()
    try {
      const info = await GetServerInfo()
      if (info?.host) serverHost = info.host
    } catch { /* server info unavailable */ }
  })

  function toSeconds(value) {
    const n = Number(value)
    return Number.isFinite(n) && n > 0 ? Math.floor(n) : 0
  }

  async function submit() {
    if (!agent) return
    submitting = true
    error = ''
    try {
      await ReconfigureAgent({
        agentId: agent.ID,
        reconnectInterval: toSeconds(reconnectInterval),
        beaconInterval: toSeconds(beaconInterval),
        beaconJitter: toSeconds(beaconJitter),
        c2Uri: c2Uri.trim() || '',
      })
      open = false
      onclose()
    } catch (e) {
      error = errorMessage(e, 'Reconfigure failed: ')
    } finally {
      submitting = false
    }
  }

  function listenerProtocol(job) {
    const candidates = [
      job.Protocol ?? job.protocol,
      job.Name ?? job.name,
      job.Description ?? job.description,
    ].map((v) => String(v ?? '').toLowerCase())
    for (const text of candidates) {
      if (text.includes('mtls')) return 'mtls'
      if (text.includes('https')) return 'https'
      if (text.includes('http')) return 'http'
      if (text.includes('dns')) return 'dns'
      if (text.includes('wireguard') || /\bwg\b/.test(text)) return 'wg'
    }
    return ''
  }

  function listenerHost(job, fallback = '') {
    const domains = job.Domains ?? job.domains ?? []
    const firstDomain = Array.isArray(domains) ? domains.find(Boolean) : ''
    if (firstDomain && !['', '0.0.0.0', '::', '[::]', '*'].includes(String(firstDomain).trim())) return firstDomain
    const candidates = [job.Host ?? job.host, job.BindHost ?? job.bindHost, job.BindAddr ?? job.bindAddr]
    for (const c of candidates) {
      const host = String(c ?? '').trim()
      if (host && !['0.0.0.0', '::', '[::]', '*'].includes(host)) return host
    }
    return fallback || ''
  }

  function setProto(prefix) {
    const stripped = c2Uri.replace(/^(mtls|https?|dns|wg|tcp-pivot|namedpipe):\/\//i, '')
    c2Uri = prefix + stripped
    protoOpen = false
  }

  function pickListener(listener) {
    if (listener.protocol === 'dns') {
      c2Uri = `dns://${listener.host || ''}`
    } else {
      c2Uri = `${listener.protocol}://${listener.host || serverHost || '<server>'}:${listener.port}`
    }
    listenerOpen = false
  }

  function protocolLabel(url) {
    return (url?.match(/^([a-z-]+):\/\//i) || ['', 'proto'])[1]
  }
</script>

<Modal bind:open title="Reconfigure Agent" size="lg" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    {#if isBeacon}
      Adjusts the beacon's polling interval, jitter, reconnect delay, and C2 URI in-place — takes effect at the next callback.
    {:else}
      Adjusts the session's reconnect delay in-place — takes effect if the session drops and needs to re-establish.
    {/if}
    Leave fields blank to keep the current value.
  </p>

  <div class="grid gap-3 md:grid-cols-2">
    <TextField
      bind:value={reconnectInterval}
      label="Reconnect interval (s)"
      type="number"
      placeholder="e.g. 60"
    />
    {#if isBeacon}
      <TextField
        bind:value={beaconInterval}
        label="Beacon interval (s)"
        type="number"
        placeholder="e.g. 60"
      />
      <TextField
        bind:value={beaconJitter}
        label="Beacon jitter (s)"
        type="number"
        placeholder="e.g. 30"
      />
      <div class="md:col-span-2 mt-1">
        <label class="text-xs font-medium text-fg-muted mb-1 block">C2 URI to switch to</label>
        <div class="flex items-center gap-2">
          <div class="relative inline-flex items-center">
            <Button color="dark" size="xs" class="!font-mono">
              <span class="lowercase">{protocolLabel(c2Uri)}</span>
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
              bind:value={c2Uri}
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
        </div>
      </div>
    {/if}
  </div>

  {#if error}
    <div class="mt-3 text-sm text-danger-500">{error}</div>
  {/if}

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <Button color="dark" onclick={() => open = false} disabled={submitting}>Cancel</Button>
      <Button color="primary" onclick={submit} disabled={submitting}>
        {submitting ? 'Applying…' : 'Apply'}
      </Button>
    </div>
  {/snippet}
</Modal>
