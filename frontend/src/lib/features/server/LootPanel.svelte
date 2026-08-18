<script>
  import { loot } from '$stores/resources/loot.svelte.js'
  import { entityColors } from '$stores/resources/entityColors.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(loot, entityColors)
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import Button from '$components/ui/Button.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import EntityTagBadges from '$components/ui/EntityTagBadges.svelte'
  import LootPreviewModal from '$features/loot/LootPreviewModal.svelte'
  import { DownloadLoot, RemoveLoot } from '../../api/server.js'
  import { contextMenu } from '../../stores/ui/contextMenu.svelte.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { addToCase } from '$stores/ui/addToCase.svelte.js'
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js'
  import { tagsModal } from '$stores/ui/tagsModal.svelte.js'
  import { entityColorStyle } from '../../utils/entityTags.js'
  import { formatBytes } from '../../utils/formats.js'

  let {
    embedded = false,
    onclose,
  } = $props()

  let selected = $state(new Set())
  let previewID = $state('')
  let previewName = $state('')
  let previewFileType = $state(0)
  let previewOpen = $state(false)

  let lootRows = $derived((loot.data || []).map((item, index) => ({
    _rowKey: item.ID || item.id || index,
    _id: item.ID || item.id || '',
    _shortID: (item.ID || item.id || '').substring(0, 8),
    _name: item.Name || item.name || '-',
    _fileType: item.FileType === 0 || item.FileType === 1 ? (item.FileType === 0 ? 'TEXT' : 'BINARY') : (item.fileType === 0 || item.fileType === 1 ? (item.fileType === 0 ? 'TEXT' : 'BINARY') : '-'),
    _fileTypeNum: item.FileType ?? item.fileType ?? 0,
    _size: item.Size || item.size || 0,
    _tags: item.Tags || item.tags || [],
  })))

  const columns = [
    { key: '_shortID', label: 'ID', width: 100 },
    { key: '_name', label: 'Name' },
    { key: '_tags', label: 'Tags', width: 108, sortable: false },
    { key: '_fileType', label: 'File Type', width: 120 },
    { key: '_size', label: 'Size', width: 80 },
  ]

  function findRow(id) {
    return lootRows.find((r) => r._id === id)
  }

  function getSingleSelected() {
    if (selected.size === 1) {
      const id = [...selected][0]
      return findRow(id)
    }
    return null
  }

  function handleContextMenu(item, event) {
    event.preventDefault()
    const id = item._id
    if (!selected.has(id)) {
      selected = new Set([id])
    }
    openItemMenu(item, event.clientX, event.clientY)
  }

  function openItemMenu(item, x, y) {
    contextMenu.open({
      x,
      y,
      target: item,
      sections: [{
        items: [
          { icon: 'eye', label: 'Preview', on: () => openPreview(item) },
          { icon: 'download', label: 'Download', on: () => download(item._id) },
          { icon: 'tag', label: 'Tags', on: () => tagsModal.openTags('loot', item._id, item._name) },
          { icon: 'message-square', label: 'Comments', on: () => commentsModal.openComments('loot', item._id, item._name) },
          { icon: 'folder', label: 'Add to Case', on: () => addToCase.open({ collection: 'loot', itemID: item._id, label: item._name }) },
          { divider: true },
          { icon: 'trash', label: 'Delete', danger: true, on: () => remove(item._id) },
        ],
      }],
    })
  }

  function openMoreMenu(event) {
    const rect = event.currentTarget.getBoundingClientRect()
    const item = getSingleSelected()
    const items = item
      ? [
          { icon: 'eye', label: 'Preview', on: () => openPreview(item) },
          { icon: 'download', label: 'Download', on: () => download(item._id) },
          { icon: 'tag', label: 'Tags', on: () => tagsModal.openTags('loot', item._id, item._name) },
          { icon: 'message-square', label: 'Comments', on: () => commentsModal.openComments('loot', item._id, item._name) },
          { icon: 'folder', label: 'Add to Case', on: () => addToCase.open({ collection: 'loot', itemID: item._id, label: item._name }) },
          { divider: true },
          { icon: 'trash', label: 'Delete', danger: true, on: () => remove(item._id) },
        ]
      : [
          { icon: 'eye', label: 'Preview', disabled: true },
          { icon: 'download', label: 'Download', disabled: true },
          { icon: 'tag', label: 'Tags', disabled: true },
          { icon: 'message-square', label: 'Comments', disabled: true },
          { icon: 'folder', label: 'Add to Case', disabled: true },
          { divider: true },
          { icon: 'trash', label: 'Delete', danger: true, disabled: true },
        ]
    contextMenu.open({
      x: rect.left,
      y: rect.bottom + 4,
      target: item,
      sections: [{ items }],
    })
  }

  function openPreview(item) { previewID = item._id; previewName = item._name; previewFileType = item._fileTypeNum; previewOpen = true }
  async function download(id) { try { await DownloadLoot(id) } catch {} }
  async function remove(id) {
    if (!(await dialog.confirm('Delete this loot item?', 'Confirm Delete'))) return
    try { await RemoveLoot(id); loot.refresh() } catch {}
  }

  function exportCSV() {
    if (!lootRows.length) return
    const header = 'ID,Name,FileType,Size\n'
    const rows = lootRows.map((r) =>
      [r._id, `"${(r._name ?? '').replaceAll('"', '""')}"`, r._fileType, r._size].join(',')
    ).join('\n')
    const blob = new Blob([header + rows], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = 'loot-export.csv'; a.click()
    URL.revokeObjectURL(url)
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Loot'} icon={embedded ? '' : 'download'}>
  <Toolbar class="justify-end gap-2">
    <Button
      color="dark" size="sm"
      disabled={lootRows.length === 0}
      onclick={exportCSV}>Export CSV</Button>
    <Button color="dark" size="sm" onclick={() => loot.refresh()}>Refresh</Button>
    <Button
      color="alternative" size="sm"
      class="border-0! bg-transparent! shadow-none! text-fg-muted hover:text-fg!"
      aria-haspopup="true"
      onclick={openMoreMenu}
    >
      <Icon name="ellipsis-vertical" size={14} />
      More
    </Button>
  </Toolbar>

  <div class="flex-1 min-h-0">
    <DataTable
      data={lootRows}
      {columns}
      keyField="_rowKey"
      loading={loot.loading}
      error={loot.error && !loot.loading ? loot.error : null}
      emptyState={{ icon: 'download', title: 'No loot stored' }}
      selectable="multi"
      bind:selected
      onRowContextMenu={handleContextMenu}
      rowStyle={(item) => entityColorStyle(entityColors.data, 'loot', item._id)}
    >
      {#snippet children(item, col)}
        {#if col.key === '_shortID'}
          <span class="font-mono">{item._shortID}</span>
        {:else if col.key === '_name'}
          {item._name}
        {:else if col.key === '_tags'}
          <EntityTagBadges entityType="loot" entityID={item._id} showEmpty />
        {:else if col.key === '_size'}
          {formatBytes(item._size, { zeroText: '-' })}
        {:else}
          {item[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>

<LootPreviewModal bind:open={previewOpen} lootID={previewID} lootName={previewName} lootFileType={previewFileType} />
