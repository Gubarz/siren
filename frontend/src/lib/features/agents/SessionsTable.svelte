<script>
  import DataTable from '$components/patterns/DataTable.svelte'
  import StatusDot from '$components/ui/StatusDot.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import {
    agentRemoteAddress, isAgentOnline, isHighPrivilege, pivotParentMap, shortAgentID,
  } from '../../utils/agents.js'
  import { now } from '../../stores/ui/now.svelte.js'
  import { sessionNotes } from '$stores/resources/sessionNotes.svelte.js'
  import { agentTags } from '$stores/resources/agentTags.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(sessionNotes, agentTags)
  import { SaveAgentNote } from '../../api/agents.js'
  import { discoveryKey } from '../../utils/discovery.js'

  let {
    data = [],
    pivotGraph = null,
    selectedAgentIDs = [],
    filterable = true,
    discoveries = [],
    selectedDiscoveryKeys = [],
    onselect,
    oninteract,
    oncontextmenu,
    ondiscoveryselect,
    ondiscoverycontextmenu,
  } = $props()

  let notes = $state({})
  let tagsByAgent = $state({})

  $effect(() => {
    const d = sessionNotes.data
    if (d && typeof d === 'object') notes = { ...d }
  })

  $effect(() => {
    const d = agentTags.data
    if (d && typeof d === 'object') tagsByAgent = { ...d }
  })

  async function saveNote(id, text) {
    notes[id] = text
    try { await SaveAgentNote(id, text || '') } catch {}
  }



  function additiveSelection(event) {
    return event.ctrlKey || event.metaKey || event.shiftKey
  }

  function getOsIcon(osName) {
    const lower = (osName || '').toLowerCase()
    if (lower.includes('win')) return 'monitor'
    if (lower.includes('linux')) return 'terminal'
    if (lower.includes('darwin') || lower.includes('mac')) return 'apple'
    return 'monitor'
  }

  function getTypeStr(item) {
    return item._kind || (item.NextCheckin !== undefined ? 'beacon' : 'session')
  }

  function fmtCheckin(ts, nowSec) {
    if (!ts) return '-'
    const s = nowSec - ts
    if (s < 2) return 'just now'
    if (s < 60) return `${s}s ago`
    if (s < 3600) return `${Math.floor(s / 60)}m ago`
    if (s < 86400) return `${Math.floor(s / 3600)}h ago`
    return `${Math.floor(s / 86400)}d ago`
  }

  let pivotParents = $derived(pivotParentMap(pivotGraph))

  let discoveredByAgent = $derived(discoveries.reduce((groups, device) => {
    const key = device.observerIDs?.[0] || device.agentID
    ;(groups[key] ??= []).push(device)
    return groups
  }, {}))

  function deviceOsIcon(osHint) {
    const s = (osHint || '').toLowerCase()
    if (s.includes('windows')) return 'monitor'
    if (s.includes('linux') || s.includes('unix')) return 'terminal'
    return 'wifi'
  }

  function deviceRow(device, observer) {
    const observerCount = device.observerIDs?.length || 1
    return {
      _rowKey: discoveryKey(device),
      _isDevice: true,
      _device: device,
      _observerID: observer?.ID || device.agentID,
      _implantName: observerCount > 1 ? `${observerCount} agents` : (observer?.Name || '-'),
      ID: observer?.ID || device.agentID,
      _type: 'device',
      Transport: device.method || '-',
      _remoteHost: device.ip,
      Hostname: device.hostname || '-',
      Username: device.mac || '-',
      _osIcon: deviceOsIcon(device.osHint),
      OS: device.osHint || '',
      Filename: device.vendor || '-',
      PID: '-',
      _lastCheckin: device.lastSeen ? Math.floor(device.lastSeen / 1000) : 0,
      _note: '-',
      _online: null,
    }
  }

  let normalizedData = $derived(data.flatMap((agent) => {
    const agentRow = {
      ...agent,
      _rowKey: agent.ID,
      _isDevice: false,
      _privileged: isHighPrivilege(agent.Username),
      _note: notes[agent.ID] || '',
      _tags: tagsByAgent[agent.ID] || [],
      _implantName: agent.Name || '-',
      _remoteHost: agentRemoteAddress(agent, pivotParents, data),
      _lastCheckin: agent.LastCheckin ?? agent.lastCheckin ?? 0,
      _online: isAgentOnline(agent, now.value),
      _type: getTypeStr(agent),
      _osIcon: getOsIcon(agent.OS),
    }
    const devices = (discoveredByAgent[agent.ID] || []).map((d) => deviceRow(d, agent))
    return [agentRow, ...devices]
  }))

  let combinedSelected = $derived(new Set([...selectedAgentIDs, ...selectedDiscoveryKeys]))

  const columns = [
    { key: '_online', label: '', width: 36 },
    { key: 'ID', label: 'Agent ID', width: 68 },
    { key: '_implantName', label: 'Implant Name', width: 92 },
    { key: '_type', label: 'Type', width: 54 },
    { key: 'Transport', label: 'Transport', width: 62 },
    { key: '_remoteHost', label: 'Remote Address', width: 92 },
    { key: 'Hostname', label: 'Computer', width: 86 },
    { key: 'Username', label: 'User', width: 86 },
    { key: 'OS', label: 'OS', width: 30 },
    { key: 'Filename', label: 'Process', width: 92 },
    { key: 'PID', label: 'PID', width: 42 },
    { key: '_lastCheckin', label: 'Last Checkin', width: 74 },
    { key: '_tags', label: 'Tags', width: 108 },
    { key: '_note', label: 'Note', width: 96 },
  ]
</script>

<DataTable data={normalizedData} {columns} keyField="_rowKey" {filterable} selectable="multi" selected={combinedSelected}
  rowClass={(item) => item._privileged ? 'text-danger-500 [&_td]:!text-danger-500' : ''}
  onRowClick={(item, e) => item._isDevice
    ? ondiscoveryselect?.({ key: item._rowKey, additive: additiveSelection(e) })
    : onselect?.({ id: item.ID, additive: additiveSelection(e) })}
  onRowDblClick={(item) => !item._isDevice && oninteract?.(item.ID)}
  onRowContextMenu={(item, e) => item._isDevice
    ? ondiscoverycontextmenu?.({ event: e, device: item._device })
    : oncontextmenu?.({ event: e, session: item })}>
  {#snippet children(item, col)}
    {#if col.key === '_online'}
      {#if item._isDevice}
        <StatusDot variant="discovered" label="Discovered device" />
      {:else}
        <StatusDot variant={item._online ? 'online' : 'offline'} />
      {/if}
    {:else if col.key === '_type'}
      <Badge variant={item._type}>{item._type.toUpperCase()}</Badge>
    {:else if col.key === 'OS'}
      {#if item.OS || !item._isDevice}
        <Icon name={item._osIcon} size={14} />
      {:else}
        <span>-</span>
      {/if}
    {:else if col.key === '_lastCheckin'}
      <span class="font-mono">{fmtCheckin(item._lastCheckin, now.value)}</span>
    {:else if col.key === '_tags'}
      {#if item._isDevice}
        <span class="text-fg-muted">-</span>
      {:else if item._tags && item._tags.length > 0}
        <div class="flex flex-wrap gap-1">
          {#each item._tags as tag}
            <span class="px-2 py-1 rounded bg-surface-200 text-3xs font-mono">{tag}</span>
          {/each}
        </div>
      {:else}
        <span class="text-fg-muted text-xs">-</span>
      {/if}
    {:else if col.key === '_note'}
      {#if item._isDevice}
        <span class="text-fg-muted">-</span>
      {:else}
        <TextInput
          size="sm"
          placeholder="Add note..."
          value={item._note}
          oninput={(e) => saveNote(item.ID, e.target.value)}
          class="note-input"
        />
      {/if}
    {:else if col.key === 'ID'}
      <span title={item.ID}>{shortAgentID(item.ID)}</span>
    {:else}
      {item[col.key] ?? '-'}
    {/if}
  {/snippet}
</DataTable>
