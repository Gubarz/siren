<script>
  import { onMount } from 'svelte'
  import Badge from '$components/ui/Badge.svelte'
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { GetCertificates } from '../../api/operatorControls.js'
  import { errorMessage } from '../../utils/errors.js'
  import { getExpiryStatus, parseExpiry } from '../../utils/certificates.js'

  let { embedded = false, onclose } = $props()

  let certs = $state([])
  let query = $state('')
  let loading = $state(false)
  let error = $state('')
  let selectedID = $state('')

  let filteredCerts = $derived(
    certs.filter((cert) => {
      const needle = query.toLowerCase()
      if (!needle) return true
      const cn = (cert.CN || cert.cn || '').toLowerCase()
      const id = String(cert.ID || cert.id || '').toLowerCase()
      return cn.includes(needle) || id.includes(needle)
    })
  )

  let certRows = $derived(filteredCerts.map((cert) => {
    const id = cert.ID || cert.id || '-'
    const expiryDate = parseExpiry(cert.ValidityExpiry || cert.validityExpiry || '')
    const status = getExpiryStatus(expiryDate)
    return {
      _rowKey: id,
      _raw: cert,
      _id: shortID(id),
      _cn: cert.CN || cert.cn || '-',
      _type: cert.Type || cert.type || '-',
      _expires: expiryDate ? expiryDate.toLocaleDateString() : '\u2014',
      _expiryStatus: status,
    }
  }))

  let selectedCert = $derived(certRows.find((r) => r._rowKey === selectedID)?._raw || null)

  const columns = [
    { key: '_id', label: 'ID', width: 100 },
    { key: '_cn', label: 'Common Name' },
    { key: '_type', label: 'Type', width: 140 },
    { key: '_expires', label: 'Expires', width: 120 },
  ]

  $effect(() => {
    if (filteredCerts.length === 0) {
      selectedID = ''
      return
    }
    if (!selectedID || !filteredCerts.some((cert) => (cert.ID || cert.id) === selectedID)) {
      selectedID = filteredCerts[0].ID || filteredCerts[0].id || ''
    }
  })

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      const resp = await GetCertificates()
      const raw = resp?.Info || resp?.info || []
      certs = [...raw].sort((a, b) => {
        const aExp = parseExpiry(a.ValidityExpiry || a.validityExpiry || '')
        const bExp = parseExpiry(b.ValidityExpiry || b.validityExpiry || '')
        if (!aExp && !bExp) return 0
        if (!aExp) return 1
        if (!bExp) return -1
        return aExp - bExp
      })
    } catch (err) {
      error = errorMessage(err, 'Failed to load certificates: ')
    } finally {
      loading = false
    }
  }

  function selectCert(row) {
    if (row?._raw) {
      selectedID = row._raw.ID || row._raw.id || ''
    }
  }

  function shortID(value) {
    return value ? String(value).slice(0, 8) : '-'
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Certificates'} icon={embedded ? '' : 'shield'}>
  <Toolbar class="justify-between">
    <div class="w-64">
      <TextInput size="sm" bind:value={query} placeholder="Search certificates..." />
    </div>
    <Button color="dark" size="sm" onclick={refresh} disabled={loading}>
      {loading ? 'Loading...' : 'Refresh'}
    </Button>
  </Toolbar>

  <div class="flex-1 min-h-0 overflow-auto">
    {#if error && !loading}
      <div class="p-3 text-sm text-danger-500">{error}</div>
    {:else}
      <div class="grid h-full min-h-96 grid-cols-5 gap-0 text-xs">
        <div class="col-span-2 min-h-0 border-r border-line">
          <DataTable
            data={certRows}
            {columns}
            keyField="_rowKey"
            {loading}
            emptyState={{ icon: 'shield', title: 'No certificates' }}
            onRowClick={selectCert}
            rowClass={(row) => row._rowKey === selectedID ? 'bg-row-selected' : ''}
            rowStyle={(row) => row._expiryStatus?.style || ''}
          >
            {#snippet children(row, col)}
              {#if col.key === '_id'}
                <span class="font-mono">{row._id}</span>
              {:else if col.key === '_cn'}
                <span class="font-mono">{row._cn}</span>
              {:else if col.key === '_type'}
                <span>{row._type}</span>
              {:else if col.key === '_expires'}
                <span class="font-mono">{row._expires}</span>
              {:else}
                {row[col.key]}
              {/if}
            {/snippet}
          </DataTable>
        </div>

        <div class="col-span-3 min-w-0 overflow-auto p-3">
          {#if selectedCert}
            {@const expiryDate = parseExpiry(selectedCert.ValidityExpiry || selectedCert.validityExpiry || '')}
            {@const status = getExpiryStatus(expiryDate)}
            {@const startDate = parseExpiry(selectedCert.ValidityStart || selectedCert.validityStart || '')}
            {@const createdDate = parseExpiry(selectedCert.CreationTime || selectedCert.creationTime || '')}

            <div class="mb-4 flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h3 class="truncate text-sm font-semibold text-fg font-mono">{selectedCert.CN || selectedCert.cn || '-'}</h3>
                <div class="mt-1 font-mono text-fg-muted">{selectedCert.ID || selectedCert.id || '-'}</div>
              </div>
              <Badge variant={status.variant} size="xs">{status.label}</Badge>
            </div>

            <section class="mb-4">
              <h4 class="mb-2 text-xs font-semibold uppercase text-fg-muted">Details</h4>
              <div class="grid grid-cols-2 gap-x-4 gap-y-2">
                <div class="text-fg-muted">Type</div>
                <div class="font-mono">{selectedCert.Type || selectedCert.type || '\u2014'}</div>
                <div class="text-fg-muted">Key Algorithm</div>
                <div class="font-mono">{selectedCert.KeyAlgorithm || selectedCert.keyAlgorithm || '\u2014'}</div>
                <div class="text-fg-muted">Created</div>
                <div class="font-mono">{createdDate ? createdDate.toLocaleString() : (selectedCert.CreationTime || selectedCert.creationTime || '\u2014')}</div>
                <div class="text-fg-muted">Valid From</div>
                <div class="font-mono">{startDate ? startDate.toLocaleString() : (selectedCert.ValidityStart || selectedCert.validityStart || '\u2014')}</div>
                <div class="text-fg-muted">Expires</div>
                <div class="font-mono">{expiryDate ? expiryDate.toLocaleString() : (selectedCert.ValidityExpiry || selectedCert.validityExpiry || '\u2014')}</div>
              </div>
            </section>

            {#if status}
              <section class="mb-4">
                <h4 class="mb-2 text-xs font-semibold uppercase text-fg-muted">Expiry Status</h4>
                <div class="flex items-center gap-3">
                  <Badge variant={status.variant} size="xs">{status.label}</Badge>
                  <span class="text-fg-muted">{status.relative}</span>
                </div>
              </section>
            {/if}
          {:else}
            <div class="text-fg-muted">Select a certificate.</div>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</Panel>
