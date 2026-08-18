<script>
  import { onMount } from 'svelte'
  import Button from '$components/ui/Button.svelte'
  import PanelBody from '$components/patterns/PanelBody.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { listMemfiles, MemfilesAdd, MemfilesRemove } from '../../api/memfiles.js'
  import { errorMessage } from '../../utils/errors.js'
  import { formatBytes } from '../../utils/formats.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'

  let { sessionID = '' } = $props()

  let files = $state([])
  let path = $state('/memfs')
  let loading = $state(false)
  let error = $state('')

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      const resp = await listMemfiles(sessionID)
      files = resp.files || []
      path = resp.path
    } catch (err) {
      error = errorMessage(err, 'Failed to list memfiles: ')
    } finally {
      loading = false
    }
  }

  async function addMemfile() {
    try {
      await MemfilesAdd(sessionID)
      await refresh()
    } catch (err) {
      error = errorMessage(err, 'Add failed: ')
    }
  }

  async function removeMemfile(file) {
    let fd = fdFromFile(file)
    if (fd == null) {
      const input = await dialog.prompt('FD to remove (enter number):', 'Remove Memfile', '')
      if (!input) return
      fd = parseFd(input)
    }
    if (fd == null) { error = 'Invalid FD'; return }
    if (!(await dialog.confirm(`Remove memfile fd=${fd}?`, 'Confirm'))) return
    try {
      await MemfilesRemove(sessionID, fd)
      await refresh()
    } catch (err) {
      error = errorMessage(err, 'Remove failed: ')
    }
  }

  function fdFromFile(file) {
    return parseFd(file.Fd ?? file.fd)
      ?? parseFd(file.Name ?? file.name)
      ?? parseFd(file.Link ?? file.link)
  }

  function parseFd(value) {
    const match = String(value ?? '').match(/(?:^|\/)(\d+)$/)
    if (!match) return null
    const fd = Number.parseInt(match[1], 10)
    return Number.isNaN(fd) ? null : fd
  }
</script>

<div class="flex flex-col h-full">
  <Toolbar class="justify-end gap-1">
    <Button color="primary" size="xs" icon="plus" onclick={addMemfile}>Add Memfile</Button>
    <Button color="dark" size="xs" onclick={refresh} disabled={loading}>Refresh</Button>
  </Toolbar>

  <PanelBody error={error || null} empty={!loading && !error && files.length === 0} emptyIcon="file" emptyTitle="No memfiles">
    <div class="p-2">
      <div class="text-xs text-fg-muted mb-2 font-mono">{path}</div>
      <table class="w-full border-collapse text-xs">
        <thead>
          <tr class="border-b border-line bg-table-header text-left text-fg-muted">
            <th class="px-3 py-2 font-medium">Name</th>
            <th class="px-3 py-2 font-medium">Size</th>
            <th class="px-3 py-2 font-medium">Mode</th>
            <th class="px-3 py-2 text-right font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each files as f}
            <tr class="border-b border-line hover:bg-row-hover">
              <td class="px-3 py-2 font-mono">{f.Name || f.name}</td>
              <td class="px-3 py-2 font-mono text-fg-muted">{formatBytes(f.Size ?? f.size)}</td>
              <td class="px-3 py-2 font-mono text-fg-muted">{f.Mode || f.mode}</td>
              <td class="px-3 py-2 text-right">
                <Button color="red" size="xs" onclick={() => removeMemfile(f)}>Remove</Button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </PanelBody>
</div>
