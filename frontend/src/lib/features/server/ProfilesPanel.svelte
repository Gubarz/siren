<script>
  import { profiles } from '$stores/resources/profiles.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(profiles)
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import Button from '$components/ui/Button.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import { DeleteProfile, GenerateImplantFromProfile } from '../../api/server.js'
  import { GenerateStage } from '../../api/staging.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'
  import { overlays } from '$stores/ui/overlays.svelte.js'

  let {
    embedded = false,
    onclose,
  } = $props()

  let profSuccess = $state('')
  let profError = $state('')
  let stagePrependSize = $state(true)
  let profileRows = $derived((profiles.data || []).map((profile, index) => {
    const config = profile.Config || profile.config || {}
    const name = profile.Name || profile.name || '-'
    return {
      _rowKey: config.ID || config.id || name || index,
      _name: name,
      _configID: config.ID || config.id,
      _osArch: `${config.GOOS || config.goos || '?'}/${config.GOARCH || config.goarch || '?'}`,
      _format: fmtFormat(config.Format ?? config.format),
      _formatValue: config.Format ?? config.format ?? 0,
      _type: (config.IsBeacon ?? config.isBeacon) ? 'beacon' : 'session',
    }
  }))

  const columns = [
    { key: '_name', label: 'Name' },
    { key: '_osArch', label: 'OS / Arch', width: 120 },
    { key: '_format', label: 'Format', width: 120 },
    { key: '_type', label: 'Type', width: 90 },
    { key: '_actions', label: '', width: 210, sortable: false },
  ]

  async function delProfile(name) {
    if (!(await dialog.confirm(`Delete profile "${name}"?`, 'Confirm Delete'))) return
    try { await DeleteProfile(name); profiles.refresh() } catch {}
  }

  async function generateProfile(id, name, format) {
    try {
      profError = ''
      profSuccess = 'Generating...'
      const path = await GenerateImplantFromProfile(id, name, format || 0)
      profSuccess = path ? 'Saved to ' + path : ''
    } catch (err) {
      profSuccess = ''
      profError = errorMessage(err, 'Generate failed: ')
    }
  }

  async function generateStage(name) {
    try {
      profError = ''
      profSuccess = 'Generating stage...'
      const path = await GenerateStage({ profile: name, prependSize: stagePrependSize })
      profSuccess = path ? 'Stage saved to ' + path : ''
    } catch (err) {
      profSuccess = ''
      profError = errorMessage(err, 'Stage failed: ')
    }
  }

  function fmtFormat(f) {
    return ({ 0: 'shared lib', 1: 'shellcode', 2: 'executable', 3: 'service', 4: 'third-party' })[f] ?? f
  }
</script>

<Panel {embedded} {onclose}>
  <Toolbar class="justify-end">
    <Checkbox bind:checked={stagePrependSize} label="Prepend stage size" />
    <Button color="dark" size="sm" onclick={() => profiles.refresh()}>Refresh</Button>
    <Button color="primary" size="sm" icon="plus" onclick={() => overlays.open('generate', { initialValues: { name: '' } })}>New Profile</Button>
  </Toolbar>

  {#if profSuccess}
    <div class="px-3 py-2 text-xs text-success-500">{profSuccess}</div>
  {/if}
  {#if profError}
    <div class="px-3 py-2 text-xs text-danger-500">{profError}</div>
  {/if}

  <div class="flex-1 min-h-0">
    <DataTable
      data={profileRows}
      {columns}
      keyField="_rowKey"
      loading={profiles.loading}
      error={profiles.error && !profiles.loading ? profiles.error : null}
      emptyState={{ icon: 'file', title: 'No saved profiles' }}
    >
      {#snippet children(profile, col)}
        {#if col.key === '_name'}
          <span class="font-mono">{profile._name}</span>
        {:else if col.key === '_actions'}
          <div class="flex gap-2">
            <Button color="dark" size="xs" onclick={() => generateProfile(profile._configID, profile._name, profile._formatValue)}>Generate</Button>
            <Button color="dark" size="xs" onclick={() => generateStage(profile._name)}>Stage</Button>
            <Button color="red" size="xs" onclick={() => delProfile(profile._name)}>Delete</Button>
          </div>
        {:else}
          {profile[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
