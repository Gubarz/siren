<script>
  import Button from '$components/ui/Button.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import TextArea from '$components/ui/TextArea.svelte'
  import { GetCase, UpdateCase, RemoveFromCase, ExportCaseReport } from '../../../api/cases.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../../utils/errors.js'

  // Right pane of the Cases split view. Loads the record on `caseID`
  // change, buffers edits locally, and pushes on Save. Add-to-case flows
  // live in each source panel's context menu — this pane only *removes*.

  let { caseID = '', onchange = () => {} } = $props()

  let record = $state(null)
  let name = $state('')
  let description = $state('')
  let notes = $state('')
  let error = $state('')
  let saving = $state(false)

  $effect(() => {
    if (!caseID) { record = null; return }
    load()
  })

  async function load() {
    error = ''
    try {
      record = await GetCase(caseID)
      if (record) {
        name = record.name || ''
        description = record.description || ''
        notes = record.notes || ''
      }
    } catch (err) {
      error = errorMessage(err, 'Load failed: ')
    }
  }

  async function save() {
    saving = true
    error = ''
    try {
      await UpdateCase(caseID, name, description, notes)
      onchange()
      await load()
    } catch (err) {
      error = errorMessage(err, 'Save failed: ')
    } finally { saving = false }
  }

  async function removeMember(collection, itemID) {
    try {
      await RemoveFromCase(caseID, collection, itemID)
      onchange()
      await load()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Remove failed: '), 'Case')
    }
  }

  async function exportReport() {
    try {
      await ExportCaseReport(caseID)
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Export failed: '), 'Case')
    }
  }

  function memberLine(itemID) { return itemID }

  function caseItemCount(c) {
    return (c.agentIds?.length || 0) +
      (c.lootIds?.length || 0) +
      (c.credIds?.length || 0) +
      (c.hostIds?.length || 0) +
      (c.canaryIds?.length || 0)
  }
</script>

<div class="flex flex-col h-full">
  {#if !caseID || !record}
    <div class="p-6 text-sm text-fg-muted">Select a case on the left, or click "New" to create one.</div>
  {:else}
    <div class="flex items-center gap-2 px-3 py-2 border-b border-line">
      <div class="text-sm font-semibold truncate">{name || record.name}</div>
      <div class="ml-auto flex gap-2">
        <Button color="dark" size="sm" icon="download" onclick={exportReport}>Export report</Button>
        <Button color="primary" size="sm" onclick={save} disabled={saving}>
          {saving ? 'Saving…' : 'Save'}
        </Button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-4">
      {#if error}<div class="text-sm text-danger-500 mb-3">{error}</div>{/if}

      <div class="grid gap-3 mb-4">
        <label class="text-xs uppercase tracking-wider text-fg-muted">Name
          <TextInput bind:value={name} class="mt-1" />
        </label>
        <label class="text-xs uppercase tracking-wider text-fg-muted">Description
          <TextInput bind:value={description} class="mt-1" />
        </label>
        <label class="text-xs uppercase tracking-wider text-fg-muted">Notes
          <TextArea bind:value={notes} rows={5} class="mt-1" />
        </label>
      </div>

      {#each [
        { key: 'agentIds', label: 'Agents', collection: 'agent' },
        { key: 'lootIds', label: 'Loot', collection: 'loot' },
        { key: 'credIds', label: 'Credentials', collection: 'cred' },
        { key: 'hostIds', label: 'Hosts', collection: 'host' },
        { key: 'canaryIds', label: 'Canaries', collection: 'canary' },
      ] as section}
        {@const items = record[section.key] || []}
        {#if items.length > 0}
          <div class="mb-4">
            <div class="text-xs uppercase tracking-wider text-fg-muted mb-2">{section.label} ({items.length})</div>
            <div class="flex flex-col gap-1">
              {#each items as itemID}
                <div class="flex items-center gap-2 px-2 py-1 rounded bg-surface-200 text-xs font-mono">
                  <span class="flex-1 truncate">{memberLine(itemID)}</span>
                  <IconButton
                    icon="x" label="Remove" size="xs" color="red"
                    onclick={() => removeMember(section.collection, itemID)}
                  />
                </div>
              {/each}
            </div>
          </div>
        {/if}
      {/each}

      {#if caseItemCount(record) === 0}
        <p class="text-xs text-fg-muted italic">
          Empty case. Use "Add to case…" in the Agents, Loot, Credentials, or Hosts panels
          to attach records here.
        </p>
      {/if}
    </div>
  {/if}
</div>
