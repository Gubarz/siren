<script>
  import DataTable from '$components/patterns/DataTable.svelte'
  import StatusDot from '$components/ui/StatusDot.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import TagBadge from '$components/ui/TagBadge.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import EntityTagBadges from '$components/ui/EntityTagBadges.svelte'
  import {
    agentKind, agentRemoteAddress, isAgentOnline, isHighPrivilege, osIcon, pivotParentMap, shortAgentID,
  } from '../../utils/agents.js'
  import { formatRelativeTime } from '../../utils/formats.js'
  import { now } from '../../stores/ui/now.svelte.js'
  import { sessionNotes } from '$stores/resources/sessionNotes.svelte.js'
  import { agentTags } from '$stores/resources/agentTags.svelte.js'
  import { agentColors } from '$stores/resources/agentColors.svelte.js'
  import { entityColors } from '$stores/resources/entityColors.svelte.js'
  import { entityTags } from '$stores/resources/entityTags.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(sessionNotes, agentTags, agentColors, entityTags, entityColors)
  import { SaveAgentNote } from '../../api/agents.js'
  import { discoveryKey } from '../../utils/discovery.js'
  import { agentColorStyle } from '../../utils/agentColors.js'
  import { errorMessage } from '../../utils/errors.js'

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
  let noteDrafts = $state({})
  let noteSaving = $state({})
  let noteErrors = $state({})
  let tagsByAgent = $state({})
  let colorsByAgent = $state({})
  let colorsByEntity = $state({})

  $effect(() => {
    const d = sessionNotes.data
    if (!d || typeof d !== 'object' || Array.isArray(d)) return
    notes = { ...d }
  })

  $effect(() => {
    const d = agentColors.data
    if (d && typeof d === 'object') colorsByAgent = { ...d }
  })

  function agentRowStyle(item) {
    if (item._isDevice) return agentColorStyle(colorsByEntity[`device:${item._entityID}`])
    return agentColorStyle(colorsByAgent[item.ID])
  }

  $effect(() => {
    const d = entityColors.data
    if (d && typeof d === 'object') colorsByEntity = { ...d }
  })

  $effect(() => {
    const d = agentTags.data
    if (d && typeof d === 'object') tagsByAgent = { ...d }
  })

  function noteValue(id) {
    return noteDrafts[id] ?? notes[id] ?? ''
  }

  function setNoteDraft(id, text) {
    noteDrafts = { ...noteDrafts, [id]: text }
    if (noteErrors[id]) {
      const next = { ...noteErrors }
      delete next[id]
      noteErrors = next
    }
  }

  async function saveNote(id, text) {
    const nextText = String(text ?? '').trim()
    if ((notes[id] ?? '') === nextText) return

    noteSaving = { ...noteSaving, [id]: true }
    try {
      await SaveAgentNote(id, nextText)
      if (nextText) {
        notes = { ...notes, [id]: nextText }
      } else {
        const nextNotes = { ...notes }
        delete nextNotes[id]
        notes = nextNotes
      }
      const nextDrafts = { ...noteDrafts }
      delete nextDrafts[id]
      noteDrafts = nextDrafts
      await sessionNotes.refresh()
    } catch (err) {
      noteErrors = { ...noteErrors, [id]: errorMessage(err, 'Save failed: ') }
    } finally {
      const nextSaving = { ...noteSaving }
      delete nextSaving[id]
      noteSaving = nextSaving
    }
  }

  function commitNote(event, id) {
    saveNote(id, event.currentTarget.value)
  }

  function noteKeydown(event) {
    event.stopPropagation()
    if (event.key === 'Enter') {
      event.preventDefault()
      event.currentTarget.blur()
    }
  }



  function additiveSelection(event) {
    return event.ctrlKey || event.metaKey || event.shiftKey
  }

  let pivotParents = $derived(pivotParentMap(pivotGraph))

  let discoveredByAgent = $derived(discoveries.reduce((groups, device) => {
    const key = device.observerIDs?.[0] || device.agentID
    ;(groups[key] ??= []).push(device)
    return groups
  }, {}))

  function deviceRow(device, observer) {
    const observerCount = device.observerIDs?.length || 1
    return {
      _rowKey: discoveryKey(device),
      _isDevice: true,
      _device: device,
      _entityID: device.ip || device.agentID,
      _observerID: observer?.ID || device.agentID,
      _implantName: observerCount > 1 ? `${observerCount} agents` : (observer?.Name || '-'),
      ID: observer?.ID || device.agentID,
      _type: 'device',
      Transport: device.method || '-',
      _remoteHost: device.ip,
      Hostname: device.hostname || '-',
      Username: device.mac || '-',
      _osIcon: osIcon(device.osHint),
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
      _type: agentKind(agent),
      _osIcon: osIcon(agent.OS),
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
  rowStyle={agentRowStyle}
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
        <StatusDot variant={isAgentOnline(item, now.value) ? 'online' : 'offline'} />
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
      <span class="font-mono">
        {#if !item._isDevice && item._type === 'session' && isAgentOnline(item, now.value)}
          Active
        {:else}
          {formatRelativeTime(item._lastCheckin, now.value)}
        {/if}
      </span>
    {:else if col.key === '_tags'}
      {#if item._isDevice}
        <EntityTagBadges entityType="device" entityID={item._entityID} showEmpty />
      {:else if item._tags && item._tags.length > 0}
        <div class="flex flex-wrap gap-1">
          {#each item._tags as tag}
            <TagBadge {tag} />
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
          value={noteValue(item.ID)}
          oninput={(e) => setNoteDraft(item.ID, e.currentTarget.value)}
          onchange={(e) => commitNote(e, item.ID)}
          onkeydown={(e) => noteKeydown(e)}
          onclick={(e) => e.stopPropagation()}
          ondblclick={(e) => e.stopPropagation()}
          title={noteErrors[item.ID] || (noteSaving[item.ID] ? 'Saving note...' : '')}
          class="note-input {noteErrors[item.ID] ? 'border-danger-500!' : ''}"
        />
      {/if}
    {:else if col.key === 'ID'}
      <span title={item.ID}>{shortAgentID(item.ID)}</span>
    {:else}
      {item[col.key] ?? '-'}
    {/if}
  {/snippet}
</DataTable>
