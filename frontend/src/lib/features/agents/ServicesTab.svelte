<script>
  import { onMount } from 'svelte'
  import { GetServices, StartService, StopService, RemoveService } from '../../api/agents.js'
  import { errorMessage } from '../../utils/errors.js'
  import { contextMenu } from '$stores/ui/contextMenu.svelte.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js'
  import { tagsModal } from '$stores/ui/tagsModal.svelte.js'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Button from '$components/ui/Button.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import EntityTagBadges from '$components/ui/EntityTagBadges.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import ErrorState from '$components/ui/ErrorState.svelte'
  import { entityColors } from '$stores/resources/entityColors.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'
  import { entityColorStyle } from '../../utils/entityTags.js'

  let { sessionID = '' } = $props()

  useResource(entityColors)

  let services = $state([])
  let loading = $state(false)
  let error = $state('')
  let filterText = $state('')

  let columns = [
    { key: 'name', label: 'Name', width: 200 },
    { key: '_tags', label: 'Tags', width: 108, sortable: false },
    { key: 'display', label: 'Display Name', width: 250 },
    { key: 'status', label: 'Status', width: 120 },
    { key: 'startup', label: 'Startup Type', width: 120 },
    { key: 'account', label: 'Account', width: 150 },
  ]

  let filtered = $derived(
    !filterText
      ? normalized
      : normalized.filter((s) =>
          [s.name, s.display, s.account, s.status].some((v) =>
            v.toLowerCase().includes(filterText.toLowerCase())
          )
        )
  )

  let normalized = $derived((services || []).map((s) => ({
    raw: s,
    _name: s.Name || s.name || '',
    name: s.Name || s.name || '',
    display: s.DisplayName || s.displayName || '',
    status: translateStatus(s.Status || s.status),
    startup: translateStartup(s.StartupType || s.startupType),
    account: s.Account || s.account || '',
    binpath: s.BinPath || s.binPath || '',
    description: s.Description || s.description || '',
    isRunning: (s.Status || s.status) === 4,
    isStopped: (s.Status || s.status) === 1,
  })))

  function translateStatus(v) {
    const m = { 1: 'Stopped', 2: 'Start Pending', 3: 'Stop Pending', 4: 'Running', 5: 'Continue Pending', 6: 'Pause Pending', 7: 'Paused' }
    return m[v] || `Unknown (${v})`
  }

  function translateStartup(v) {
    const m = { 0: 'Boot', 1: 'System', 2: 'Automatic', 3: 'Manual', 4: 'Disabled' }
    return m[v] || `Unknown (${v})`
  }

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      const resp = await GetServices(sessionID)
      services = resp?.Details || resp?.details || []
    } catch (err) {
      error = errorMessage(err, 'Failed to list services')
    } finally {
      loading = false
    }
  }

  async function doStart(svc) {
    try {
      await StartService(sessionID, svc.name)
      await refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Start failed'), 'Service')
    }
  }

  async function doStop(svc) {
    try {
      await StopService(sessionID, svc.name)
      await refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Stop failed'), 'Service')
    }
  }

  async function doRemove(svc) {
    if (!(await dialog.confirm(`Remove service "${svc.name}"?`, 'Confirm'))) return
    try {
      await RemoveService(sessionID, svc.name)
      await refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Remove failed'), 'Service')
    }
  }

  function handleRightClick(e, item) {
    const svc = normalized.find((s) => s.raw === item)
    if (!svc) return
    const items = []
    if (svc.isStopped) items.push({ icon: 'play', label: 'Start', on: () => doStart(svc) })
    if (svc.isRunning) items.push({ icon: 'stop', label: 'Stop', on: () => doStop(svc) })
    items.push({ icon: 'info', label: 'Details', on: () => showDetail(svc) })
    items.push({ icon: 'tag', label: 'Tags / Color…', on: () => tagsModal.openTags('service', svc.name, svc.displayName || svc.name) })
    items.push({ icon: 'message-square', label: 'Comments / Notes…', on: () => commentsModal.openComments('service', svc.name, svc.displayName || svc.name) })
    items.push({ icon: 'trash', label: 'Remove', danger: true, on: () => doRemove(svc) })
    contextMenu.open({ x: e.clientX, y: e.clientY, sections: [{ items }] })
  }

  let detailSvc = $state(null)

  function showDetail(svc) {
    detailSvc = svc
  }

  async function detailAction(action) {
    if (!detailSvc) return
    try {
      if (action === 'start') await StartService(sessionID, detailSvc.name)
      else if (action === 'stop') await StopService(sessionID, detailSvc.name)
      else if (action === 'remove') {
        if (!(await dialog.confirm(`Remove service "${detailSvc.name}"?`, 'Confirm'))) return
        await RemoveService(sessionID, detailSvc.name)
      }
      detailSvc = null
      await refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, `${action} failed`), 'Service')
    }
  }
</script>

<div class="flex flex-col h-full">
  <div class="tab-header">
    <div class="max-w-75 flex-1">
      <TextInput size="sm" placeholder="Filter..." bind:value={filterText} />
    </div>
    <Button color="primary" size="xs" onclick={refresh}>Refresh</Button>
  </div>
  <div class="flex-1 overflow-y-auto">
    {#if error}
      <ErrorState {error} title="Failed to load services" class="m-2" />
    {/if}
    <DataTable
      data={filtered}
      columns={columns}
      keyField="_name"
      loading={loading}
      emptyState={{ title: filterText ? 'No matching services.' : 'No services found.' }}
      rowStyle={(item) => entityColorStyle(entityColors.data, 'service', item.name)}
      onRowDblClick={(item) => showDetail(item)}
      onRowContextMenu={(item, e) => handleRightClick(e, item.raw)}
    >
      {#snippet children(item, col)}
        {#if col.key === 'name'}
          {item.name}
        {:else if col.key === '_tags'}
          <EntityTagBadges entityType="service" entityID={item.name} showEmpty />
        {:else}
          {item[col.key] ?? ''}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</div>

{#if detailSvc}
  <div class="fixed inset-0 bg-black/60 z-100 flex items-center justify-center" onclick={() => detailSvc = null} onkeydown={(e) => { if (e.key === 'Escape') detailSvc = null }} role="dialog" tabindex="-1">
    <div class="min-w-120 max-w-160 bg-panel border border-line rounded-lg flex flex-col overflow-hidden shadow-2xl" role="none" onclick={(e) => e.stopPropagation()}>
      <div class="flex items-center px-4 py-2 bg-chrome border-b border-line">
        <span class="flex-1 text-base font-semibold">{detailSvc.name}</span>
        <IconButton icon="x" label="Close" tooltip="Close" onclick={() => detailSvc = null} />
      </div>
      <div class="px-4 py-3">
        <div class="flex py-1 text-xs"><span class="w-30 shrink-0 text-fg-muted">Name</span><span class="flex-1">{detailSvc.name}</span></div>
        <div class="flex py-1 text-xs"><span class="w-30 shrink-0 text-fg-muted">Display Name</span><span class="flex-1">{detailSvc.display}</span></div>
        <div class="flex py-1 text-xs"><span class="w-30 shrink-0 text-fg-muted">Description</span><span class="flex-1">{detailSvc.description}</span></div>
        <div class="flex py-1 text-xs"><span class="w-30 shrink-0 text-fg-muted">Status</span><span class="flex-1">{detailSvc.status}</span></div>
        <div class="flex py-1 text-xs"><span class="w-30 shrink-0 text-fg-muted">Startup Type</span><span class="flex-1">{detailSvc.startup}</span></div>
        <div class="flex py-1 text-xs"><span class="w-30 shrink-0 text-fg-muted">Account</span><span class="flex-1">{detailSvc.account}</span></div>
        <div class="flex py-1 text-xs"><span class="w-30 shrink-0 text-fg-muted">Binary Path</span><span class="flex-1 font-mono">{detailSvc.binpath}</span></div>
      </div>
      <div class="flex gap-2 px-4 py-2 border-t border-line">
        {#if detailSvc.isStopped}
          <Button color="primary" size="xs" onclick={() => detailAction('start')}>Start</Button>
        {/if}
        {#if detailSvc.isRunning}
          <Button color="primary" size="xs" onclick={() => detailAction('stop')}>Stop</Button>
        {/if}
        <Button color="red" size="xs" onclick={() => detailAction('remove')}>Remove</Button>
      </div>
    </div>
  </div>
{/if}
