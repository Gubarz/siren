<script>
  import { loot } from '$stores/resources/loot.svelte.js'
  import { entityColors } from '$stores/resources/entityColors.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(loot, entityColors)
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import Button from '$components/ui/Button.svelte'
  import EntityTagBadges from '$components/ui/EntityTagBadges.svelte'
  import { DownloadLoot, RemoveLoot } from '../../api/server.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { addToCase } from '$stores/ui/addToCase.svelte.js'
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js'
  import { tagsModal } from '$stores/ui/tagsModal.svelte.js'
  import { entityColorStyle } from '../../utils/entityTags.js'

  let {
    embedded = false,
    onclose,
  } = $props()

  let lootRows = $derived((loot.data || []).map((item, index) => ({
    _rowKey: item.ID || item.id || index,
    _id: item.ID || item.id || '',
    _shortID: (item.ID || item.id || '').substring(0, 8),
    _name: item.Name || item.name || '-',
    _fileType: item.FileType || item.fileType || '-',
  })))

  const columns = [
    { key: '_shortID', label: 'ID', width: 100 },
    { key: '_name', label: 'Name' },
    { key: '_tags', label: 'Tags', width: 108, sortable: false },
    { key: '_fileType', label: 'File Type', width: 150 },
    { key: '_actions', label: '', width: 260, sortable: false },
  ]

  async function download(id) { try { await DownloadLoot(id) } catch {} }
  async function remove(id) {
    if (!(await dialog.confirm('Delete this loot item?', 'Confirm Delete'))) return
    try { await RemoveLoot(id); loot.refresh() } catch {}
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Loot'} icon={embedded ? '' : 'download'}>
  <Toolbar class="justify-end">
    <Button color="dark" size="sm" onclick={() => loot.refresh()}>Refresh</Button>
  </Toolbar>

  <div class="flex-1 min-h-0">
    <DataTable
      data={lootRows}
      {columns}
      keyField="_rowKey"
      loading={loot.loading}
      error={loot.error && !loot.loading ? loot.error : null}
      emptyState={{ icon: 'download', title: 'No loot stored' }}
      rowStyle={(item) => entityColorStyle(entityColors.data, 'loot', item._id)}
    >
      {#snippet children(item, col)}
        {#if col.key === '_shortID'}
          <span class="font-mono">{item._shortID}</span>
        {:else if col.key === '_name'}
          {item._name}
        {:else if col.key === '_tags'}
          <EntityTagBadges entityType="loot" entityID={item._id} showEmpty />
        {:else if col.key === '_actions'}
          <div class="flex gap-2">
            <Button color="dark" size="xs" onclick={() => download(item._id)}>Download</Button>
            <Button color="dark" size="xs" icon="tag" onclick={() => tagsModal.openTags('loot', item._id, item._name)}>Tags</Button>
            <Button color="dark" size="xs" icon="message-square" onclick={() => commentsModal.openComments('loot', item._id, item._name)}>Comments</Button>
            <Button color="dark" size="xs" icon="folder" onclick={() => addToCase.open({
              collection: 'loot', itemID: item._id, label: item._name,
            })}>Case</Button>
            <Button color="red" size="xs" onclick={() => remove(item._id)}>Delete</Button>
          </div>
        {:else}
          {item[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
