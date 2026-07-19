<script>
  import { onMount } from 'svelte'
  import Button from '$components/ui/Button.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { GetCanaries } from '../../api/operatorControls.js'
  import { errorMessage } from '../../utils/errors.js'

  // Lists every DNS canary the teamserver has issued for implants and
  // whether it's been resolved — a resolved canary usually means someone
  // (sandbox / analyst) triggered the built-in DNS check.

  let { embedded = false, onclose } = $props()

  let canaries = $state([])
  let loading = $state(false)
  let error = $state('')
  let canaryRows = $derived(canaries.map((canary, index) => ({
    _rowKey: canary.ID || canary.id || canary.Domain || canary.domain || index,
    _domain: canary.Domain || canary.domain || '-',
    _implant: canary.ImplantName || canary.implantName || '-',
    _triggered: Boolean(canary.Triggered || canary.triggered),
    _firstTrigger: fmtTime(canary.FirstTriggered ?? canary.firstTriggered),
    _latestTrigger: fmtTime(canary.LatestTrigger ?? canary.latestTrigger),
  })))

  const columns = [
    { key: '_domain', label: 'Domain' },
    { key: '_implant', label: 'Implant' },
    { key: '_triggered', label: 'Triggered', width: 90 },
    { key: '_firstTrigger', label: 'First Trigger', width: 180 },
    { key: '_latestTrigger', label: 'Latest Trigger', width: 180 },
  ]

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      const resp = await GetCanaries()
      canaries = resp?.Canaries || resp?.canaries || []
    } catch (err) {
      error = errorMessage(err, 'Failed to load canaries: ')
    } finally {
      loading = false
    }
  }

  function fmtTime(v) {
    const n = Number(v)
    if (!n) return '—'
    return new Date(n * 1000).toLocaleString()
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Canaries'} icon={embedded ? '' : 'bird'}>
  <Toolbar class="justify-end">
    <Button color="dark" size="sm" onclick={refresh} disabled={loading}>
      {loading ? 'Loading…' : 'Refresh'}
    </Button>
  </Toolbar>

  <div class="flex-1 min-h-0">
    <DataTable
      data={canaryRows}
      {columns}
      keyField="_rowKey"
      {loading}
      error={error || null}
      emptyState={{ icon: 'bird', title: 'No DNS canaries' }}
    >
      {#snippet children(canary, col)}
        {#if col.key === '_domain'}
          <span class="font-mono">{canary._domain}</span>
        {:else if col.key === '_triggered'}
          {#if canary._triggered}
            <span class="text-danger-500 font-semibold">yes</span>
          {:else}
            <span class="text-fg-muted">no</span>
          {/if}
        {:else}
          {canary[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
