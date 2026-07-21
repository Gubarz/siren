<script>
  import { hosts } from '$stores/resources/hosts.svelte.js'
  import { sessions } from '$stores/resources/sessions.svelte.js'
  import { beacons } from '$stores/resources/beacons.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(hosts, sessions, beacons)
  import Badge from '$components/ui/Badge.svelte'
  import Button from '$components/ui/Button.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { getHost, RemoveHost, RemoveHostIOC } from '../../api/hosts.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'
  import { addToCase } from '$stores/ui/addToCase.svelte.js'
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js'

  let { embedded = false, onclose } = $props()

  let query = $state('')
  let selectedUUID = $state('')
  let detail = $state(null)
  let detailLoading = $state(false)
  let actionError = $state('')

  let filteredHosts = $derived(
    (hosts.data || []).filter((host) => {
      const needle = query.toLowerCase()
      return !needle || [host.hostname, host.hostUUID, host.osVersion, host.locale]
        .some((value) => String(value || '').toLowerCase().includes(needle))
    })
  )
  let selectedHost = $derived(filteredHosts.find((host) => host.hostUUID === selectedUUID) || null)
  let currentHost = $derived(detail || selectedHost)
  let linkedSessions = $derived(linkedAgents(sessions.data || [], currentHost?.hostUUID))
  let linkedBeacons = $derived(linkedAgents(beacons.data || [], currentHost?.hostUUID))
  let hostRows = $derived(filteredHosts.map((host, index) => ({
    _rowKey: host.hostUUID || index,
    _raw: host,
    _hostUUID: host.hostUUID,
    _hostname: host.hostname || '-',
    _summary: `${shortID(host.hostUUID)} / ${host.osVersion || '-'}`,
    _sessionCount: linkedAgents(sessions.data || [], host.hostUUID).length,
    _beaconCount: linkedAgents(beacons.data || [], host.hostUUID).length,
    _iocCount: (host.iocs || []).length,
  })))
  let iocRows = $derived((currentHost?.iocs || []).map((ioc, index) => ({
    _rowKey: ioc.id || ioc.path || ioc.fileHash || index,
    _raw: ioc,
    _path: ioc.path || '-',
    _hash: ioc.fileHash || '-',
    _id: ioc.id,
  })))

  const hostColumns = [
    { key: '_hostname', label: 'Host' },
    { key: '_agents', label: 'Agents', width: 150, sortable: false },
    { key: '_iocCount', label: 'IOCs', width: 70 },
    { key: '_actions', label: '', width: 150, sortable: false },
  ]
  const iocColumns = [
    { key: '_path', label: 'Path' },
    { key: '_hash', label: 'Hash', width: 240 },
    { key: '_actions', label: '', width: 56, sortable: false },
  ]

  $effect(() => {
    if (filteredHosts.length === 0) {
      selectedUUID = ''
      detail = null
      return
    }
    if (!selectedUUID || !filteredHosts.some((host) => host.hostUUID === selectedUUID)) {
      selectHost(filteredHosts[0])
    }
  })

  async function selectHost(host) {
    if (!host?.hostUUID) return
    selectedUUID = host.hostUUID
    detail = null
    detailLoading = true
    actionError = ''
    try {
      const next = await getHost(host.hostUUID)
      if (selectedUUID === host.hostUUID) detail = next
    } catch (err) {
      actionError = errorMessage(err, 'Host detail failed: ')
    } finally {
      detailLoading = false
    }
  }

  async function removeHost(host) {
    if (!host?.hostUUID) return
    const name = host.hostname || shortID(host.hostUUID)
    if (!(await dialog.confirm(`Forget host "${name}"?`, 'Confirm Remove'))) return
    actionError = ''
    try {
      await RemoveHost(host.hostUUID)
      selectedUUID = ''
      detail = null
      await hosts.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Remove host failed: ')
    }
  }

  async function removeIOC(ioc) {
    if (!ioc?.id || !selectedUUID) return
    if (!(await dialog.confirm(`Remove IOC "${ioc.path || ioc.fileHash || ioc.id}"?`, 'Confirm Remove'))) return
    actionError = ''
    try {
      await RemoveHostIOC(ioc.id)
      detail = await getHost(selectedUUID)
      await hosts.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Remove IOC failed: ')
    }
  }

  function linkedAgents(agents, hostUUID) {
    if (!hostUUID) return []
    return agents.filter((agent) => (agent.UUID ?? agent.uuid ?? '') === hostUUID)
  }

  function agentID(agent) {
    return agent.ID ?? agent.id ?? ''
  }

  function shortID(value) {
    return value ? String(value).slice(0, 8) : '-'
  }

  function fmtTime(seconds) {
    return seconds ? new Date(seconds * 1000).toLocaleString() : '-'
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Hosts'} icon={embedded ? '' : 'server'}>
  <Toolbar class="justify-between">
    <div class="w-64">
      <TextInput size="sm" bind:value={query} placeholder="Search hosts..." />
    </div>
    <Button color="dark" size="sm" onclick={() => hosts.refresh()} disabled={hosts.loading}>
      {hosts.loading ? 'Loading...' : 'Refresh'}
    </Button>
  </Toolbar>

  {#if actionError}
    <div class="border-b border-line px-3 py-2 text-xs text-danger-500">{actionError}</div>
  {/if}

  <div class="flex-1 min-h-0 overflow-auto">
    {#if hosts.error && !hosts.loading}
      <div class="p-3 text-sm text-danger-500">{hosts.error}</div>
    {:else}
      <div class="grid h-full min-h-96 grid-cols-5 gap-0 text-xs">
        <div class="col-span-2 min-h-0 border-r border-line">
          <DataTable
            data={hostRows}
            columns={hostColumns}
            keyField="_rowKey"
            loading={hosts.loading}
            emptyState={{ icon: 'server', title: 'No hosts' }}
            onRowClick={(host) => selectHost(host._raw)}
            rowClass={(host) => host._hostUUID === selectedUUID ? 'bg-row-selected' : ''}
          >
            {#snippet children(host, col)}
              {#if col.key === '_hostname'}
                <div class="font-mono">{host._hostname}</div>
                <div class="truncate text-fg-muted">{host._summary}</div>
              {:else if col.key === '_agents'}
                <div class="flex flex-wrap gap-1">
                  <Badge size="xs" variant="session">{host._sessionCount} session</Badge>
                  <Badge size="xs" variant="beacon">{host._beaconCount} beacon</Badge>
                </div>
              {:else if col.key === '_iocCount'}
                <span class="font-mono">{host._iocCount}</span>
              {:else if col.key === '_actions'}
                <div class="flex justify-end gap-2">
                  <Button color="dark" size="xs" onclick={(event) => { event.stopPropagation(); selectHost(host._raw) }}>View</Button>
                  <Button color="dark" size="xs" icon="message-square" onclick={(event) => {
                    event.stopPropagation()
                    commentsModal.openComments('host', host._hostUUID, host._hostname || host._hostUUID)
                  }}>Comments</Button>
                  <IconButton icon="folder" label="Add to case" tooltip="Add to case" size="xs" onclick={(event) => {
                    event.stopPropagation()
                    addToCase.open({
                      collection: 'host', itemID: host._hostUUID, label: host._hostname || host._hostUUID,
                    })
                  }} />
                  <IconButton icon="trash" label="Forget host" tooltip="Forget host" color="red" size="xs" onclick={(event) => {
                    event.stopPropagation()
                    removeHost(host._raw)
                  }} />
                </div>
              {:else}
                {host[col.key]}
              {/if}
            {/snippet}
          </DataTable>
        </div>

        <div class="col-span-3 min-w-0 overflow-auto p-3">
        {#if detailLoading}
          <div class="text-fg-muted">Loading host...</div>
        {:else if currentHost}
          <div class="mb-3 flex items-start justify-between gap-3">
            <div class="min-w-0">
              <h3 class="truncate text-sm font-semibold text-fg">{currentHost.hostname || '-'}</h3>
              <div class="mt-1 font-mono text-fg-muted">{currentHost.hostUUID || '-'}</div>
              <div class="mt-1 text-fg-muted">{currentHost.osVersion || '-'} / {currentHost.locale || '-'}</div>
              <div class="mt-1 text-fg-muted">First contact: {fmtTime(currentHost.firstContact)}</div>
            </div>
            <div class="flex items-center gap-1">
              <Button color="dark" size="xs" icon="message-square" onclick={() => commentsModal.openComments('host', currentHost.hostUUID, currentHost.hostname || currentHost.hostUUID)}>Comments</Button>
              <IconButton icon="trash" label="Forget host" tooltip="Forget host" color="red" size="sm" onclick={() => removeHost(currentHost)} />
            </div>
          </div>

          <section class="mb-4">
            <h4 class="mb-2 text-xs font-semibold uppercase text-fg-muted">Linked Agents</h4>
            <div class="flex flex-wrap gap-2">
              {#each linkedSessions as session}
                <Badge size="xs" variant="session">{shortID(agentID(session))}</Badge>
              {/each}
              {#each linkedBeacons as beacon}
                <Badge size="xs" variant="beacon">{shortID(agentID(beacon))}</Badge>
              {/each}
              {#if linkedSessions.length === 0 && linkedBeacons.length === 0}
                <span class="text-fg-muted">No active sessions or beacons.</span>
              {/if}
            </div>
          </section>

          <section class="mb-4">
            <h4 class="mb-2 text-xs font-semibold uppercase text-fg-muted">IOCs</h4>
            {#if (currentHost.iocs || []).length === 0}
              <div class="text-fg-muted">No IOCs recorded.</div>
            {:else}
              <div class="max-h-64 min-h-24">
                <DataTable
                  data={iocRows}
                  columns={iocColumns}
                  keyField="_rowKey"
                  emptyState={{ icon: 'file', title: 'No IOCs recorded' }}
                >
                  {#snippet children(ioc, col)}
                    {#if col.key === '_path'}
                      <span class="font-mono">{ioc._path}</span>
                    {:else if col.key === '_hash'}
                      <span class="font-mono text-fg-muted" title={ioc._hash}>{ioc._hash}</span>
                    {:else if col.key === '_actions'}
                      <div class="flex justify-end">
                        <IconButton icon="trash" label="Remove IOC" tooltip="Remove IOC" color="red" size="xs" disabled={!ioc._id} onclick={() => removeIOC(ioc._raw)} />
                      </div>
                    {:else}
                      {ioc[col.key]}
                    {/if}
                  {/snippet}
                </DataTable>
              </div>
            {/if}
          </section>

          <section>
            <h4 class="mb-2 text-xs font-semibold uppercase text-fg-muted">Extension Data</h4>
            {#if (currentHost.extensionData || []).length === 0}
              <div class="text-fg-muted">No extension data recorded.</div>
            {:else}
              <div class="grid gap-2">
                {#each currentHost.extensionData || [] as item}
                  <div class="border border-line bg-panel p-2">
                    <div class="mb-1 font-mono text-fg">{item.name}</div>
                    <pre class="max-h-40 overflow-auto whitespace-pre-wrap text-fg-muted">{item.output || '-'}</pre>
                  </div>
                {/each}
              </div>
            {/if}
          </section>
        {:else}
          <div class="text-fg-muted">Select a host.</div>
        {/if}
        </div>
      </div>
    {/if}
  </div>
</Panel>
