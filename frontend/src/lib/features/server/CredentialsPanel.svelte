<script>
  import Button from '$components/ui/Button.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import Select from '$components/ui/Select.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import { credentials } from '$stores/resources/credentials.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(credentials)
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { AddCredential, RemoveCredential } from '../../api/server.js'
  import {
    GetCredentialByID,
    GetCredentialsByHashType,
    GetPlaintextCredentialsByHashType,
    SniffCredentialHashType,
    UpdateCredential,
  } from '../../api/operatorControls.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'
  import { addToCase } from '$stores/ui/addToCase.svelte.js'

  let {
    embedded = false,
    onclose,
  } = $props()

  let filter = $state('')
  let hashTypeFilter = $state('')
  let plaintextOnly = $state(false)
  let serverFilteredCredentials = $state(null)
  let serverFilterLoading = $state(false)
  let serverFilterError = $state('')
  let filterReload = $state(0)
  let showAdd = $state(false)
  let nu = $state(''), np = $state(''), nh = $state(''), ncol = $state('')
  let sniffedType = $state('')
  let editingID = $state('')
  let edit = $state({})

  let hashTypeOptions = $derived(
    Array.from(new Set(
      (credentials.data || [])
        .map((c) => c.HashType ?? c.hashType)
        .filter((t) => t != null && t !== '' && t !== 0),
    )).sort((a, b) => Number(a) - Number(b)),
  )
  let hashFilterOptions = $derived([
    { value: '', label: 'All hash types' },
    ...hashTypeOptions.map((type) => ({ value: String(type), label: `Type ${type}` })),
  ])
  let credentialSource = $derived(hashTypeFilter !== '' ? (serverFilteredCredentials || []) : (credentials.data || []))

  let shown = $derived(
    credentialSource.filter((c) => {
      const type = c.HashType ?? c.hashType
      if (hashTypeFilter !== '' && serverFilteredCredentials == null && String(type) !== String(hashTypeFilter)) return false
      if (plaintextOnly && hashTypeFilter === '' && !(c.Plaintext || c.plaintext)) return false
      if (!filter) return true
      const hay = `${c.Username || c.username || ''} ${c.Plaintext || c.plaintext || ''} ${c.Hash || c.hash || ''}`.toLowerCase()
      return hay.includes(filter.toLowerCase())
    })
  )
  let credentialRows = $derived(shown.map((credential, index) => ({
    _rowKey: credential.ID || credential.id || index,
    _raw: credential,
    _id: credential.ID || credential.id,
    _username: credential.Username || credential.username || '-',
    _plaintext: credential.Plaintext || credential.plaintext || '-',
    _hash: credential.Hash || credential.hash || '-',
    _hashType: credential.HashType || credential.hashType || '-',
    _collection: credential.Collection || credential.collection || '-',
  })))

  const columns = [
    { key: '_username', label: 'Username' },
    { key: '_plaintext', label: 'Plaintext' },
    { key: '_hash', label: 'Hash', width: 260 },
    { key: '_hashType', label: 'Type', width: 90 },
    { key: '_collection', label: 'Collection' },
    { key: '_actions', label: '', width: 220, sortable: false },
  ]

  $effect(() => {
    const selectedType = hashTypeFilter
    const onlyPlaintext = plaintextOnly
    void filterReload

    if (!selectedType) {
      serverFilteredCredentials = null
      serverFilterError = ''
      serverFilterLoading = false
      return
    }

    let cancelled = false
    serverFilterLoading = true
    serverFilterError = ''
    ;(async () => {
      try {
        const fetcher = onlyPlaintext ? GetPlaintextCredentialsByHashType : GetCredentialsByHashType
        const resp = await fetcher(Number(selectedType))
        if (!cancelled) serverFilteredCredentials = resp?.Credentials || resp?.credentials || []
      } catch (err) {
        if (!cancelled) {
          serverFilteredCredentials = []
          serverFilterError = errorMessage(err, 'Hash filter failed: ')
        }
      } finally {
        if (!cancelled) serverFilterLoading = false
      }
    })()

    return () => {
      cancelled = true
    }
  })

  async function refreshCredentials() {
    await credentials.refresh()
    filterReload += 1
  }

  async function add() {
    try { await AddCredential(nu, np, nh, ncol); nu = np = nh = ncol = ''; sniffedType = ''; showAdd = false; refreshCredentials() } catch {}
  }

  async function remove(id) {
    if (!id || !(await dialog.confirm('Delete this credential?', 'Confirm Delete'))) return
    try { await RemoveCredential(id); refreshCredentials() } catch {}
  }

  async function detectHashType() {
    if (!nh) return
    try {
      const result = await SniffCredentialHashType(nh)
      const type = result?.HashType ?? result?.hashType
      sniffedType = type != null ? `Detected type: ${type}` : 'Detected: (unclassified)'
    } catch (err) {
      sniffedType = `Detect failed: ${err?.message || err}`
    }
  }

  async function startEdit(c) {
    const id = c.ID || c.id
    let fresh
    try {
      fresh = await GetCredentialByID(id)
    } catch {
      fresh = c
    }
    edit = {
      username: fresh.Username ?? fresh.username ?? '',
      plaintext: fresh.Plaintext ?? fresh.plaintext ?? '',
      hash: fresh.Hash ?? fresh.hash ?? '',
      hashType: Number(fresh.HashType ?? fresh.hashType ?? 0),
      collection: fresh.Collection ?? fresh.collection ?? '',
      isCracked: Boolean(fresh.IsCracked ?? fresh.isCracked ?? false),
    }
    editingID = id
  }

  function cancelEdit() {
    editingID = ''
    edit = {}
  }

  async function saveEdit() {
    if (!editingID) return
    try {
      await UpdateCredential({ id: editingID, ...edit })
      cancelEdit()
      refreshCredentials()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Update failed: '), 'Credentials')
    }
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Credentials'} icon={embedded ? '' : 'key'}>
  <Toolbar class="justify-end">
    <Checkbox bind:checked={plaintextOnly} label="Plaintext only" />
    <Select
      bind:value={hashTypeFilter}
      options={hashFilterOptions}
      aria-label="Hash type filter"
      size="sm"
      class="w-40"
    />
    <TextInput size="sm" placeholder="filter…" bind:value={filter} class="font-mono" />
    <Button color="primary" size="sm" icon="plus" onclick={() => showAdd = !showAdd}>Add</Button>
    <Button color="dark" size="sm" onclick={refreshCredentials}>Refresh</Button>
  </Toolbar>

  {#if serverFilterError}
    <div class="border-b border-line px-3 py-2 text-xs text-danger-500">{serverFilterError}</div>
  {/if}

  {#if showAdd}
    <div class="flex gap-2 flex-wrap px-3 py-2 border-b border-line">
      <div class="flex-1 min-w-30"><TextInput size="sm" aria-label="Username" placeholder="username" bind:value={nu} /></div>
      <div class="flex-1 min-w-30"><TextInput size="sm" aria-label="Plaintext" placeholder="plaintext" bind:value={np} /></div>
      <div class="flex-1 min-w-30"><TextInput size="sm" aria-label="Hash" placeholder="hash" bind:value={nh} /></div>
      <Button color="dark" size="sm" onclick={detectHashType} disabled={!nh}>Detect type</Button>
      <div class="flex-1 min-w-30"><TextInput size="sm" aria-label="Collection" placeholder="collection" bind:value={ncol} /></div>
      <Button color="primary" size="sm" onclick={add}>Save</Button>
      <Button color="dark" size="sm" onclick={() => { showAdd = false; sniffedType = '' }}>Cancel</Button>
    </div>
    {#if sniffedType}
      <div class="px-3 py-1 text-xs text-fg-muted border-b border-line">{sniffedType}</div>
    {/if}
  {/if}

  <div class="flex-1 min-h-0">
    <DataTable
      data={credentialRows}
      {columns}
      keyField="_rowKey"
      loading={credentials.loading || serverFilterLoading}
      error={credentials.error && !credentials.loading ? credentials.error : null}
      emptyState={{ icon: 'key', title: 'No credentials' }}
      onRowDblClick={(credential) => startEdit(credential._raw)}
      rowClass={(credential) => editingID === credential._id ? 'bg-brand-muted' : ''}
    >
      {#snippet children(credential, col)}
        {#if editingID === credential._id}
          {#if col.key === '_username'}
            <TextInput size="xs" bind:value={edit.username} />
          {:else if col.key === '_plaintext'}
            <TextInput size="xs" bind:value={edit.plaintext} />
          {:else if col.key === '_hash'}
            <TextInput size="xs" bind:value={edit.hash} />
          {:else if col.key === '_hashType'}
            <TextInput size="xs" type="number" bind:value={edit.hashType} />
          {:else if col.key === '_collection'}
            <TextInput size="xs" bind:value={edit.collection} />
          {:else if col.key === '_actions'}
            <div class="flex gap-1">
              <Button color="primary" size="xs" onclick={saveEdit}>Save</Button>
              <Button color="dark" size="xs" onclick={cancelEdit}>Cancel</Button>
            </div>
          {/if}
        {:else if col.key === '_plaintext' || col.key === '_hash'}
          <span class="font-mono">{credential[col.key]}</span>
        {:else if col.key === '_actions'}
          <div class="flex gap-2">
            <Button color="dark" size="xs" onclick={() => startEdit(credential._raw)}>Edit</Button>
            <Button color="dark" size="xs" icon="folder" onclick={() => addToCase.open({
              collection: 'cred', itemID: credential._id, label: credential._username || 'credential',
            })}>Case</Button>
            <Button color="red" size="xs" onclick={() => remove(credential._id)}>Delete</Button>
          </div>
        {:else}
          {credential[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
