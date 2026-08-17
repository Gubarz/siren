<script>
  import { implantBuilds } from '$stores/resources/implantBuilds.svelte.js'
  import { jobs } from '$stores/resources/jobs.svelte.js'
  import { profiles } from '$stores/resources/profiles.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'
  useResource(implantBuilds, jobs, profiles)
  import {
    GenerateStage,
    StageImplantBuilds,
    StartTCPStagerListener,
    UnstageAllImplantBuilds,
    UnstageImplantBuild,
  } from '../../api/staging.js'
  import { KillJob } from '../../api/server.js'
  import { OpenFileDialog } from '../../api/runtime.js'
  import { stagedBuildRows, stagerListenerRows } from '../../utils/staging.js'
  import { errorMessage } from '../../utils/errors.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import Button from '$components/ui/Button.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import Select from '$components/ui/Select.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  let { embedded = false, onclose } = $props()
  let status = $state('')
  let error = $state('')
  let stageBuildName = $state('')
  let genProfile = $state('')
  let genName = $state('')
  let aesKey = $state('')
  let aesIv = $state('')
  let rc4Key = $state('')
  let compress = $state('')
  let genPrepend = $state(true)
  let listenerHost = $state('0.0.0.0')
  let listenerPort = $state(8080)
  let listenerProfile = $state('')
  let stagePath = $state('')
  let listenerPrepend = $state(true)
  const COMPRESS_OPTIONS = [
    { value: '', label: 'none' },
    { value: 'zlib', label: 'zlib' },
    { value: 'gzip', label: 'gzip' },
    { value: 'deflate', label: 'deflate' },
  ]
  let profileOptions = $derived(
    (profiles.data || [])
      .map((profile) => ({ value: profile.Name || profile.name, label: profile.Name || profile.name }))
      .filter((option) => option.value)
  )
  let stagedRows = $derived(stagedBuildRows(implantBuilds.data || []))
  let stageOptions = $derived(
    (implantBuilds.data || [])
      .filter((build) => !build.staged && build.name)
      .map((build) => ({ value: build.name, label: build.name }))
  )
  let stagerRows = $derived(stagerListenerRows(jobs.data || []))
  const buildColumns = [
    { key: '_name', label: 'Build' },
    { key: '_osArch', label: 'OS / Arch', width: 120 },
    { key: '_format', label: 'Format', width: 120 },
    { key: '_type', label: 'Type', width: 90 },
    { key: '_actions', label: '', width: 120, sortable: false },
  ]
  const stagerColumns = [
    { key: '_id', label: 'Job ID', width: 90 },
    { key: '_port', label: 'Port', width: 80 },
    { key: '_profile', label: 'Profile / Stage file' },
    { key: '_actions', label: '', width: 120, sortable: false },
  ]
  $effect(() => {
    if (!genProfile && profileOptions.length > 0) genProfile = profileOptions[0].value
    if (!listenerProfile && profileOptions.length > 0) listenerProfile = profileOptions[0].value
  })
  async function stageSelected() {
    status = ''
    error = ''
    if (!stageBuildName) {
      error = 'Pick a build to stage'
      return
    }
    try {
      await StageImplantBuilds([stageBuildName])
      status = `${stageBuildName} is staged`
      stageBuildName = ''
      await implantBuilds.refresh()
    } catch (err) {
      error = errorMessage(err, 'Stage failed: ')
    }
  }
  async function unstageOne(name) {
    status = ''
    error = ''
    try {
      await UnstageImplantBuild(name)
      status = `${name} unstaged`
      await implantBuilds.refresh()
    } catch (err) {
      error = errorMessage(err, 'Unstage failed: ')
    }
  }
  async function unstageAll() {
    if (!(await dialog.confirm('Unstage every staged build?', 'Unstage All'))) return
    status = ''
    error = ''
    try {
      await UnstageAllImplantBuilds()
      status = 'All builds unstaged'
      await implantBuilds.refresh()
    } catch (err) {
      error = errorMessage(err, 'Unstage failed: ')
    }
  }
  async function killStager(id) {
    status = ''
    error = ''
    try {
      await KillJob(id)
      status = `Job ${id} killed`
      await jobs.refresh()
    } catch (err) {
      error = errorMessage(err, 'Kill failed: ')
    }
  }
  async function generateStageFile() {
    status = ''
    error = ''
    if (!genProfile) {
      error = 'Pick a profile to generate a stage from'
      return
    }
    try {
      const path = await GenerateStage({
        profile: genProfile,
        name: genName,
        aesEncryptKey: aesKey,
        aesEncryptIv: aesIv,
        rc4EncryptKey: rc4Key,
        compress,
        prependSize: genPrepend,
      })
      status = path ? `Stage saved to ${path}` : ''
    } catch (err) {
      error = errorMessage(err, 'Stage generation failed: ')
    }
  }
  async function startListener() {
    status = ''
    error = ''
    if (!stagePath && !listenerProfile) {
      error = 'Pick an implant profile or local stage file'
      return
    }
    try {
      const listener = await StartTCPStagerListener({
        host: listenerHost,
        port: Number(listenerPort),
        profile: stagePath ? '' : listenerProfile,
        stagePath,
        prependSize: listenerPrepend,
      })
      status = `Job ${listener?.JobID ?? listener?.jobID ?? '?'} started`
      await jobs.refresh()
    } catch (err) {
      error = errorMessage(err, 'Listener failed: ')
    }
  }
  async function pickStageFile() {
    const path = await OpenFileDialog('Select stage file')
    if (path) stagePath = path
  }
</script>

<Panel {embedded} {onclose}>
  <Toolbar class="justify-end">
    <Button color="dark" size="sm" onclick={() => { implantBuilds.refresh(); jobs.refresh(); profiles.refresh() }}>Refresh</Button>
  </Toolbar>
  {#if status || error}
    <div class="flex flex-wrap items-center gap-3 border-b border-line px-3 py-2 text-xs">
      {#if status}<span class="text-success-500">{status}</span>{/if}
      {#if error}<span class="text-danger-500">{error}</span>{/if}
    </div>
  {/if}
  <div class="flex-1 min-h-0 overflow-auto">
    <div class="grid gap-4 p-3">
      <section>
        <h3 class="mb-2 text-sm font-semibold text-fg">Staged builds</h3>
        <div class="mb-2 flex flex-wrap items-center gap-2">
          <div class="w-56">
            <Select bind:value={stageBuildName} options={stageOptions} placeholder="Choose build to stage..." size="sm" />
          </div>
          <Button color="primary" size="sm" onclick={stageSelected} disabled={stageOptions.length === 0}>Stage</Button>
          <Button color="dark" size="sm" onclick={unstageAll} disabled={stagedRows.length === 0}>Unstage all</Button>
        </div>
        <DataTable
          data={stagedRows}
          columns={buildColumns}
          keyField="_rowKey"
          loading={implantBuilds.loading}
          error={implantBuilds.error && !implantBuilds.loading ? implantBuilds.error : null}
          emptyState={{ icon: 'factory', title: 'No staged builds', description: 'Stage a build above or from the Builds panel.' }}
        >
          {#snippet children(row, col)}
            {#if col.key === '_name'}
              <span class="font-mono">{row._name}</span>
            {:else if col.key === '_actions'}
              <div class="flex justify-end">
                <Button color="red" size="xs" onclick={() => unstageOne(row._name)}>Unstage</Button>
              </div>
            {:else}
              {row[col.key]}
            {/if}
          {/snippet}
        </DataTable>
      </section>
      <section>
        <h3 class="mb-2 text-sm font-semibold text-fg">Stager listeners</h3>
        <DataTable
          data={stagerRows}
          columns={stagerColumns}
          keyField="_rowKey"
          loading={jobs.loading}
          error={jobs.error && !jobs.loading ? jobs.error : null}
          emptyState={{ icon: 'headphones', title: 'No stager listeners', description: 'Start one below.' }}
        >
          {#snippet children(row, col)}
            {#if col.key === '_id' || col.key === '_port'}
              <span class="font-mono">{row[col.key]}</span>
            {:else if col.key === '_profile'}
              <span class="font-mono">{row._profile}</span>
            {:else if col.key === '_actions'}
              <div class="flex justify-end">
                <Button color="red" size="xs" onclick={() => killStager(row._id)}>Kill</Button>
              </div>
            {:else}
              {row[col.key]}
            {/if}
          {/snippet}
        </DataTable>
      </section>
      <section>
        <h3 class="mb-2 text-sm font-semibold text-fg">Generate stage file</h3>
        <div class="flex flex-wrap items-center gap-2">
          <div class="w-56">
            <Select bind:value={genProfile} options={profileOptions} placeholder="Profile" size="sm" />
          </div>
          <div class="w-40">
            <TextInput size="sm" bind:value={genName} placeholder="Name (optional)" />
          </div>
          <div class="w-56">
            <Select bind:value={compress} options={COMPRESS_OPTIONS} placeholder="Compression" size="sm" />
          </div>
          <Checkbox bind:checked={genPrepend} label="Prepend size" />
        </div>
        <div class="flex flex-wrap items-center gap-2 pt-2">
          <div class="w-40"><TextInput size="sm" bind:value={aesKey} placeholder="AES key" class="font-mono" /></div>
          <div class="w-40"><TextInput size="sm" bind:value={aesIv} placeholder="AES IV" class="font-mono" /></div>
          <div class="w-40"><TextInput size="sm" bind:value={rc4Key} placeholder="RC4 key" class="font-mono" /></div>
          <Button color="primary" size="sm" onclick={generateStageFile}>Generate</Button>
        </div>
      </section>
      <section>
        <h3 class="mb-2 text-sm font-semibold text-fg">Start stager listener</h3>
        <div class="flex flex-wrap items-center gap-2">
          <div class="w-40"><TextInput size="sm" bind:value={listenerHost} placeholder="Host" class="font-mono" /></div>
          <div class="w-24"><TextInput type="number" size="sm" bind:value={listenerPort} placeholder="Port" class="font-mono" /></div>
          <div class="w-56">
            <Select bind:value={listenerProfile} options={profileOptions} disabled={Boolean(stagePath)} placeholder="Profile" size="sm" />
          </div>
          <Button color="dark" size="sm" onclick={pickStageFile}>Pick file</Button>
          {#if stagePath}
            <Button color="dark" size="sm" onclick={() => { stagePath = '' }}>Use profile</Button>
          {/if}
          <Checkbox bind:checked={listenerPrepend} label="Prepend size" />
          <Button color="primary" size="sm" onclick={startListener}>Start</Button>
        </div>
        {#if stagePath}
          <p class="break-all pt-2 text-xs text-fg-muted">Stage file: {stagePath}</p>
        {/if}
      </section>
    </div>
  </div>
</Panel>
