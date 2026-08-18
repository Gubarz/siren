<script>
  import { onMount } from 'svelte'
  import Modal from '../../../components/patterns/Modal.svelte'
  import Button from '../../../components/ui/Button.svelte'
  import TextField from '../../../components/forms/TextField.svelte'
  import C2UriInput from '../../../components/forms/C2UriInput.svelte'
  import { ReconfigureAgent } from '../../../api/operatorControls.js'
  import { GetServerInfo } from '../../../api/server.js'
  import { errorMessage } from '../../../utils/errors.js'
  import { listenerHost, listenerProtocol } from '../../../utils/listeners.js'
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
        <C2UriInput
          bind:value={c2Uri}
          listeners={c2Listeners}
          {serverHost}
        />
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
