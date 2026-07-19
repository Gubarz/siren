<script>
  import Modal from '../../../components/patterns/Modal.svelte'
  import Button from '../../../components/ui/Button.svelte'
  import TextField from '../../../components/forms/TextField.svelte'
  import { ReconfigureAgent } from '../../../api/operatorControls.js'
  import { errorMessage } from '../../../utils/errors.js'

  // Reconfigure lets an operator retune a live beacon's cadence (interval,
  // jitter, reconnect) or a session's reconnect timer without recompiling
  // and re-implanting. Zero values fall through to the current implant
  // setting server-side.

  let {
    open = $bindable(false),
    onclose = () => {},
    agent = null,
  } = $props()

  let isBeacon = $derived(agent?._kind === 'beacon')
  let reconnectInterval = $state('')
  let beaconInterval = $state('')
  let beaconJitter = $state('')
  let submitting = $state(false)
  let error = $state('')

  $effect(() => {
    if (open && agent) {
      reconnectInterval = String(agent.ReconnectInterval ?? '')
      beaconInterval = isBeacon ? String(agent.Interval ?? '') : ''
      beaconJitter = isBeacon ? String(agent.Jitter ?? '') : ''
      error = ''
    }
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
      Adjusts the beacon's polling interval, jitter, and reconnect delay in-place — takes effect at the next callback.
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
