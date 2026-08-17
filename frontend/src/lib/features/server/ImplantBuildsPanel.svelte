<script>
  import { implantBuilds } from '$stores/resources/implantBuilds.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(implantBuilds)
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import { DeleteImplantBuild, RegenerateImplant } from '../../api/server.js'
  import { StageImplantBuilds, UnstageImplantBuild } from '../../api/staging.js'
  import { implantFormat } from '../../utils/formats.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'
  import { overlays } from '$stores/ui/overlays.svelte.js'

  let {
    embedded = false,
    onclose,
  } = $props()

  let buildSearchQuery = $state('')
  let buildStatus = $state('')
  let buildError = $state('')

  let filtered = $derived(
    (implantBuilds.data || []).filter((b) =>
      b.name.toLowerCase().includes(buildSearchQuery.toLowerCase())
    )
  )
  let buildRows = $derived(filtered.map((build, index) => ({
    _rowKey: build.name || index,
    _name: build.name || '-',
    _osArch: `${build.GOOS || build.goos || '?'}/${build.GOARCH || build.goarch || '?'}`,
    _format: implantFormat(build.Format ?? build.format),
    _type: (build.IsBeacon ?? build.isBeacon) ? 'beacon' : 'session',
    _staged: Boolean(build.staged),
  })))

  const columns = [
    { key: '_name', label: 'Name' },
    { key: '_osArch', label: 'OS / Arch', width: 120 },
    { key: '_format', label: 'Format', width: 120 },
    { key: '_type', label: 'Type', width: 90 },
    { key: '_staged', label: 'Stage', width: 90 },
    { key: '_actions', label: '', width: 280, sortable: false },
  ]

  async function delBuild(name) {
    if (!(await dialog.confirm(`Delete build "${name}"?`, 'Confirm Delete'))) return
    try { await DeleteImplantBuild(name); implantBuilds.refresh() } catch {}
  }

  async function regen(name) {
    try { await RegenerateImplant(name) } catch {}
  }

  async function stageBuild(name) {
    buildStatus = ''
    buildError = ''
    try {
      await StageImplantBuilds([name])
      buildStatus = `${name} is staged`
      await implantBuilds.refresh()
    } catch (err) {
      buildError = errorMessage(err, 'Stage failed: ')
    }
  }

  async function unstageBuild(name) {
    buildStatus = ''
    buildError = ''
    try {
      await UnstageImplantBuild(name)
      buildStatus = `${name} unstaged`
      await implantBuilds.refresh()
    } catch (err) {
      buildError = errorMessage(err, 'Unstage failed: ')
    }
  }
</script>

<Panel {embedded} {onclose}>
  <Toolbar class="justify-end">
    <div class="w-50">
      <TextInput size="sm" placeholder="Search builds..." bind:value={buildSearchQuery} />
    </div>
    <Button color="dark" size="sm" onclick={() => implantBuilds.refresh()}>Refresh</Button>
    <Button color="primary" size="sm" icon="plus" onclick={() => overlays.open('generate')}>Generate</Button>
  </Toolbar>

  {#if buildStatus || buildError}
    <div class="flex items-center gap-3 border-b border-line px-3 py-2 text-xs">
      {#if buildStatus}<span class="text-success-500">{buildStatus}</span>{/if}
      {#if buildError}<span class="text-danger-500">{buildError}</span>{/if}
    </div>
  {/if}

  <div class="flex-1 min-h-0">
    <DataTable
      data={buildRows}
      {columns}
      keyField="_rowKey"
      loading={implantBuilds.loading}
      error={implantBuilds.error && !implantBuilds.loading ? implantBuilds.error : null}
      emptyState={{ icon: 'hammer', title: 'No builds yet', description: 'Click Generate to create one.' }}
    >
      {#snippet children(build, col)}
        {#if col.key === '_name'}
          <span class="font-mono">{build._name}</span>
        {:else if col.key === '_staged'}
          {#if build._staged}
            <Badge variant="success" size="xs">serving</Badge>
          {:else}
            <span class="text-fg-muted">-</span>
          {/if}
        {:else if col.key === '_actions'}
          <div class="flex gap-2">
            <Button color="dark" size="xs" onclick={() => regen(build._name)}>Download</Button>
            <Button color="dark" size="xs" onclick={() => stageBuild(build._name)} disabled={build._staged}>Stage</Button>
            <Button color="dark" size="xs" onclick={() => unstageBuild(build._name)} disabled={!build._staged}>Unstage</Button>
            <Button color="red" size="xs" onclick={() => delBuild(build._name)}>Delete</Button>
          </div>
        {:else}
          {build[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
