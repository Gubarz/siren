<script>
  import Modal from '$components/patterns/Modal.svelte'
  import Button from '$components/ui/Button.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import LoadingState from '$components/ui/LoadingState.svelte'
  import {
    DeleteClientConfig, Disconnect, ExportClientConfig,
    GetClientConfigDetails, ImportClientConfig,
  } from '../../api/connection.js'
  import { connection } from '$stores/connection.svelte.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'
  import { toast } from '$stores/ui/toast.svelte.js'
  import { errorMessage } from '../../utils/errors.js'

  let { open = $bindable(false) } = $props()

  let profiles = $state([])
  let loading = $state(false)
  let busy = $state(false)
  let fileInputEl = $state(null)

  $effect(() => { if (open) load() })

  async function load() {
    loading = true
    try {
      profiles = await GetClientConfigDetails() || []
    } catch (e) {
      toast.push({ variant: 'error', message: `Could not load profiles: ${errorMessage(e)}` })
      profiles = []
    } finally {
      loading = false
    }
  }

  function isActive(p) {
    return connection.profile === p.name
  }

  async function switchTo(profile) {
    if (busy || isActive(profile)) return
    if (connection.connected && !(await dialog.confirm(
      `Disconnect from "${connection.profile}" and connect to "${profile.name}"?`,
      'Switch teamserver'))) return
    busy = true
    try {
      if (connection.connected) {
        await Disconnect()
        connection.disconnect()
      }
      await connection.connect(profile.name)
      open = false
      toast.push({ variant: 'success', message: `Connected to ${profile.name}` })
    } catch (e) {
      toast.push({ variant: 'error', message: `Switch failed: ${errorMessage(e)}` })
    } finally {
      busy = false
    }
  }

  async function disconnect() {
    if (!connection.connected || busy) return
    if (!(await dialog.confirm(`Disconnect from "${connection.profile}"?`, 'Disconnect'))) return
    busy = true
    try {
      await Disconnect()
      connection.disconnect()
      open = false
    } catch (e) {
      toast.push({ variant: 'error', message: `Disconnect failed: ${errorMessage(e)}` })
    } finally {
      busy = false
    }
  }

  async function exportOne(profile) {
    try {
      const raw = await ExportClientConfig(profile.name)
      const blob = new Blob([raw], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      const safe = profile.name.replace(/[^\w.@-]+/g, '_')
      a.download = `${safe}.cfg`
      document.body.appendChild(a); a.click(); a.remove()
      URL.revokeObjectURL(url)
    } catch (e) {
      toast.push({ variant: 'error', message: `Export failed: ${errorMessage(e)}` })
    }
  }

  async function deleteOne(profile) {
    if (isActive(profile)) {
      await dialog.alert('Disconnect first before deleting the active profile.')
      return
    }
    if (!(await dialog.confirm(`Delete profile "${profile.name}"?`, 'Delete profile'))) return
    try {
      await DeleteClientConfig(profile.name)
      await load()
    } catch (e) {
      toast.push({ variant: 'error', message: `Delete failed: ${errorMessage(e)}` })
    }
  }

  function pickImport() { fileInputEl?.click() }

  async function handleFile(event) {
    const file = event.currentTarget.files?.[0]
    event.currentTarget.value = ''
    if (!file) return
    try {
      const text = await file.text()
      const name = await ImportClientConfig(text)
      toast.push({ variant: 'success', message: `Imported ${name}` })
      await load()
    } catch (e) {
      toast.push({ variant: 'error', message: `Import failed: ${errorMessage(e)}` })
    }
  }
</script>

<Modal bind:open title="Teamserver profiles" icon="server" size="lg">
  <div class="flex gap-2 mb-3">
    <Button size="sm" color="alternative" icon="upload" onclick={pickImport}>Import .cfg</Button>
    {#if connection.connected}
      <Button size="sm" color="alternative" icon="x" onclick={disconnect} disabled={busy}>Disconnect</Button>
    {/if}
  </div>
  <input bind:this={fileInputEl} type="file" accept=".cfg,.json,application/json" class="hidden" onchange={handleFile} />

  {#if loading}
    <LoadingState />
  {:else if profiles.length === 0}
    <p class="text-sm text-fg-muted">No profiles found. Import a .cfg file to add one.</p>
  {:else}
    <div class="flex flex-col gap-1">
      {#each profiles as profile (profile.name)}
        <div class={`flex items-center gap-3 p-2 rounded border ${isActive(profile) ? 'border-brand bg-panel' : 'border-line'}`}>
          <Icon name={isActive(profile) ? 'check-circle' : 'server'} class={isActive(profile) ? 'text-success-500' : 'text-fg-muted'} />
          <div class="flex-1 min-w-0">
            <div class="font-medium truncate">{profile.name}</div>
            <div class="text-xs text-fg-muted">{profile.operator}@{profile.lhost}:{profile.lport}</div>
          </div>
          {#if !isActive(profile)}
            <Button size="sm" color="primary" disabled={busy} onclick={() => switchTo(profile)}>Connect</Button>
          {/if}
          <Button size="sm" color="alternative" icon="download" title="Export" onclick={() => exportOne(profile)} />
          <Button size="sm" color="alternative" icon="trash" title="Delete" onclick={() => deleteOne(profile)} />
        </div>
      {/each}
    </div>
  {/if}
</Modal>
