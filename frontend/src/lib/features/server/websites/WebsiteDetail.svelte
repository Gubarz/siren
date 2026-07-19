<script>
  import Button from '$components/ui/Button.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import WebsiteAddContentModal from './WebsiteAddContentModal.svelte'
  import { GetWebsite, RemoveWebsiteContent } from '../../../api/websites.js'
  import { dialog } from '../../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../../utils/errors.js'

  // Content-path table for one selected site. All server round-trips go
  // through the parent via callbacks so refresh state stays coherent
  // between the list and the detail pane.

  let {
    name = '',
    onchange = () => {},
  } = $props()

  let site = $state(null)
  let loading = $state(false)
  let error = $state('')

  let modalOpen = $state(false)
  let modalMode = $state('add')
  let modalPath = $state('')
  let modalCT = $state('')

  $effect(() => {
    if (name) load()
    else site = null
  })

  async function load() {
    loading = true
    error = ''
    try {
      site = await GetWebsite(name)
    } catch (err) {
      error = errorMessage(err, 'Failed to load site: ')
    } finally {
      loading = false
    }
  }

  let rows = $derived(site
    ? Object.values(site.Contents || site.contents || {}).sort((a, b) =>
        (a.Path || a.path || '').localeCompare(b.Path || b.path || ''))
    : [])
  let contentRows = $derived(rows.map((row, index) => ({
    _rowKey: row.Path || row.path || index,
    _raw: row,
    _path: row.Path || row.path || '-',
    _contentType: row.ContentType || row.contentType || '-',
    _size: humanBytes(row.Size ?? row.size),
    _sha256: row.Sha256 || row.sha256 || '-',
  })))

  const columns = [
    { key: '_path', label: 'Path' },
    { key: '_contentType', label: 'Content-Type', width: 180 },
    { key: '_size', label: 'Size', width: 100 },
    { key: '_sha256', label: 'SHA-256', width: 240 },
    { key: '_actions', label: '', width: 130, sortable: false },
  ]

  function humanBytes(n) {
    const size = Number(n) || 0
    if (size < 1024) return `${size} B`
    if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
    return `${(size / 1024 / 1024).toFixed(1)} MB`
  }

  function openAdd() {
    modalMode = 'add'
    modalPath = ''
    modalCT = ''
    modalOpen = true
  }

  function openReplace(row) {
    modalMode = 'replace'
    modalPath = row.Path || row.path
    modalCT = row.ContentType || row.contentType || ''
    modalOpen = true
  }

  async function removeRow(row) {
    const path = row.Path || row.path
    if (!(await dialog.confirm(`Remove ${path} from ${name}?`, 'Confirm Remove'))) return
    try {
      await RemoveWebsiteContent(name, [path])
      await load()
      onchange()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Remove failed: '), 'Website')
    }
  }

  async function onModalSuccess() {
    await load()
    onchange()
  }
</script>

<div class="flex flex-col h-full">
  <div class="flex items-center gap-2 px-3 py-2 border-b border-line">
    <div class="text-sm font-semibold">{name || 'No site selected'}</div>
    <div class="ml-auto flex gap-2">
      <Button color="dark" size="sm" onclick={load} disabled={!name || loading}>
        {loading ? 'Loading…' : 'Refresh'}
      </Button>
      <Button color="primary" size="sm" icon="plus" onclick={openAdd} disabled={!name}>Add content</Button>
    </div>
  </div>

  {#if !name}
    <div class="p-6 text-sm text-fg-muted">Select a site on the left, or add content to create a new one.</div>
  {:else}
    <div class="flex-1 min-h-0">
      <DataTable
        data={contentRows}
        {columns}
        keyField="_rowKey"
        {loading}
        error={error || null}
        emptyState={{ icon: 'globe', title: 'No content registered yet' }}
      >
        {#snippet children(row, col)}
          {#if col.key === '_path' || col.key === '_contentType'}
            <span class="font-mono">{row[col.key]}</span>
          {:else if col.key === '_sha256'}
            <span class="font-mono" title={row._sha256}>{row._sha256}</span>
          {:else if col.key === '_actions'}
            <div class="flex gap-2">
              <Button color="dark" size="xs" onclick={() => openReplace(row._raw)}>Replace</Button>
              <Button color="red" size="xs" onclick={() => removeRow(row._raw)}>Delete</Button>
            </div>
          {:else}
            {row[col.key]}
          {/if}
        {/snippet}
      </DataTable>
    </div>
  {/if}
</div>

<WebsiteAddContentModal
  bind:open={modalOpen}
  mode={modalMode}
  siteName={name}
  initialPath={modalPath}
  initialContentType={modalCT}
  onsuccess={onModalSuccess}
/>
