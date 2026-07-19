<script>
  import Modal from '../../../components/patterns/Modal.svelte'
  import Button from '../../../components/ui/Button.svelte'
  import Select from '../../../components/ui/Select.svelte'
  import { GetBeacon, UpdateBeaconIntegrity } from '../../../api/operatorControls.js'
  import { errorMessage } from '../../../utils/errors.js'

  // Displays the freshest server-side beacon record — pulled via GetBeacon
  // rather than the cached list — and lets the operator retag the
  // integrity level after a getsystem / make-token / runas without
  // waiting for the next full agent refresh cycle.

  let {
    open = $bindable(false),
    onclose = () => {},
    beaconID = '',
  } = $props()

  let detail = $state(null)
  let loading = $state(false)
  let error = $state('')
  let integrityDraft = $state('')
  let saving = $state(false)

  const integrityOptions = [
    { value: '', label: '(unknown)' },
    { value: 'Untrusted', label: 'Untrusted' },
    { value: 'Low', label: 'Low' },
    { value: 'Medium', label: 'Medium' },
    { value: 'High', label: 'High' },
    { value: 'System', label: 'System' },
    { value: 'Protected', label: 'Protected' },
  ]

  $effect(() => {
    if (open && beaconID) {
      loadDetail()
    }
  })

  async function loadDetail() {
    loading = true
    error = ''
    try {
      detail = await GetBeacon(beaconID)
      integrityDraft = detail?.Integrity || ''
    } catch (e) {
      error = errorMessage(e, 'Failed to fetch beacon: ')
    } finally {
      loading = false
    }
  }

  async function saveIntegrity() {
    saving = true
    error = ''
    try {
      await UpdateBeaconIntegrity(beaconID, integrityDraft)
      await loadDetail()
    } catch (e) {
      error = errorMessage(e, 'Update failed: ')
    } finally {
      saving = false
    }
  }

  function pair(k, v) {
    return { k, v: v == null || v === '' ? '—' : String(v) }
  }

  let rows = $derived(detail ? [
    pair('Name', detail.Name),
    pair('ID', detail.ID),
    pair('Hostname', detail.Hostname),
    pair('User', detail.Username),
    pair('OS', `${detail.OS || '?'} / ${detail.Arch || '?'}`),
    pair('PID', detail.PID),
    pair('Integrity', detail.Integrity),
    pair('Interval / jitter', `${detail.Interval || 0}s / ${detail.Jitter || 0}s`),
    pair('Reconnect', `${detail.ReconnectInterval || 0}s`),
    pair('Active C2', detail.ActiveC2),
    pair('Remote', detail.RemoteAddress),
    pair('First contact', detail.FirstContact ? new Date(detail.FirstContact * 1000).toLocaleString() : ''),
    pair('Last checkin', detail.LastCheckin ? new Date(detail.LastCheckin * 1000).toLocaleString() : ''),
  ] : [])
</script>

<Modal bind:open title="Beacon Detail" size="2xl" {onclose}>
  {#if loading}
    <p class="text-fg-muted text-sm">Loading…</p>
  {:else if error}
    <p class="text-danger-500 text-sm">{error}</p>
  {:else if detail}
    <div class="grid grid-cols-2 gap-x-6 gap-y-1 text-sm">
      {#each rows as row}
        <div class="text-fg-muted">{row.k}</div>
        <div class="font-mono break-all">{row.v}</div>
      {/each}
    </div>

    <div class="mt-5 border-t border-line pt-4">
      <label class="block text-sm font-semibold text-fg mb-1" for="integrity-select">Integrity level</label>
      <div class="flex gap-2">
        <Select
          id="integrity-select"
          options={integrityOptions}
          bind:value={integrityDraft}
          class="flex-1"
        />
        <Button color="primary" size="xs" onclick={saveIntegrity} disabled={saving || integrityDraft === detail.Integrity}>
          {saving ? 'Saving…' : 'Update'}
        </Button>
      </div>
      <p class="mt-1 text-xs text-fg-muted">
        Retag after a token change (getsystem / make-token / runas) so the sessions table matches reality without waiting for the next full refresh.
      </p>
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <Button color="dark" onclick={() => open = false}>Close</Button>
    </div>
  {/snippet}
</Modal>
