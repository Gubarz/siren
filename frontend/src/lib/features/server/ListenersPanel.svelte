<script>
  import { KillJob, StartListener } from '../../api/server.js'
  import { isStagerJob } from '../../utils/staging.js'
  import { StartWGListener, GenerateUniqueWGIP, GenerateWGClientConfig } from '../../api/wireguard.js'
  import { jobs } from '$stores/resources/jobs.svelte.js'
  import { entityColors } from '$stores/resources/entityColors.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(jobs, entityColors)
  import { errorMessage } from '../../utils/errors.js'
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js'
  import { tagsModal } from '$stores/ui/tagsModal.svelte.js'
  import Select from '$components/ui/Select.svelte'
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import EntityTagBadges from '$components/ui/EntityTagBadges.svelte'
  import { entityColorStyle } from '../../utils/entityTags.js'

  let {
    embedded = false,
    onclose,
  } = $props()

  let proto = $state('mtls')
  let host = $state('0.0.0.0')
  let port = $state(443)
  let domains = $state('')
  let starting = $state(false)
  let listenerError = $state('')
  let listenerStatus = $state('')
  let wgTunIP = $state('')
  let wgNPort = $state(1337)
  let wgKeyPort = $state(1338)
  let wgClientConfig = $state(null)
  let jobRows = $derived((jobs.data || []).map((job, index) => ({
    _rowKey: job.ID ?? job.id ?? index,
    _id: job.ID ?? job.id ?? '-',
    _name: job.Name ?? job.name ?? '-',
    _protocol: job.Protocol ?? job.protocol ?? '-',
    _port: job.Port ?? job.port ?? '-',
    _description: job.Description ?? job.description ?? '-',
    _isStager: isStagerJob(job),
    _profile: job.ProfileName ?? job.profileName ?? '',
  })))
  const DEFAULT_PORTS = { mtls: 443, http: 80, https: 443, dns: 53, wireguard: 1337 }
  const PROTO_OPTIONS = ['mtls', 'http', 'https', 'dns', 'wireguard'].map((p) => ({ value: p, label: p }))
  const columns = [
    { key: '_id', label: 'ID', width: 80 },
    { key: '_name', label: 'Name' },
    { key: '_tags', label: 'Tags', width: 108, sortable: false },
    { key: '_protocol', label: 'Protocol', width: 100 },
    { key: '_port', label: 'Port', width: 80 },
    { key: '_profile', label: 'Profile', width: 150 },
    { key: '_description', label: 'Description' },
    { key: '_actions', label: '', width: 220, sortable: false },
  ]

  let isDNS = $derived(proto === 'dns')
  let isWG = $derived(proto === 'wireguard')

  function onProtoChange() {
    port = DEFAULT_PORTS[proto] ?? port
    listenerError = ''
    listenerStatus = ''
  }

  async function start() {
    starting = true
    listenerError = ''
    listenerStatus = ''
    try {
      if (isWG) await startWG()
      else await StartListener(proto, host, Number(port), domains)
      await jobs.refresh()
    } catch (err) {
      listenerError = errorMessage(err, 'Listener failed: ')
    } finally {
      starting = false
    }
  }

  async function startWG() {
    if (!wgTunIP) {
      const ip = await GenerateUniqueWGIP()
      wgTunIP = ip?.IP || ip?.ip || ''
    }
    await StartWGListener(host, Number(port), Number(wgNPort), Number(wgKeyPort), wgTunIP)
    listenerStatus = 'WireGuard listener started'
  }

  async function downloadWGConfig() {
    try {
      wgClientConfig = await GenerateWGClientConfig()
    } catch (err) {
      listenerError = errorMessage(err, 'Config failed: ')
    }
  }

  async function kill(id) {
    try { await KillJob(id); await jobs.refresh() } catch (err) { listenerError = errorMessage(err, 'Kill failed: ') }
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Listeners'} icon={embedded ? '' : 'headphones'} size={embedded ? 'lg' : '3xl'}>
  <div class="flex gap-2 flex-wrap px-3 py-2 border-b border-line">
    <div class="w-25">
      <Select bind:value={proto} options={PROTO_OPTIONS} onchange={onProtoChange} />
    </div>
    <div class="flex-1 min-w-30">
      <TextInput size="sm" aria-label="Listener host" bind:value={host} placeholder="Host" class="font-mono" />
    </div>
    <div class="w-22">
      <TextInput type="number" size="sm" aria-label="Listener port" bind:value={port} placeholder="Port" class="font-mono" />
    </div>
    {#if isDNS}
      <div class="flex-1 min-w-40">
        <TextInput size="sm" aria-label="DNS domains" bind:value={domains} placeholder="domains (comma-separated)" class="font-mono" />
      </div>
    {/if}
    {#if isWG}
      <TextInput size="sm" aria-label="WG tun IP" bind:value={wgTunIP} placeholder="Tun IP (auto)" class="font-mono w-30" />
      <TextInput type="number" size="sm" aria-label="WG nport" bind:value={wgNPort} placeholder="NPort" class="font-mono w-15" />
      <TextInput type="number" size="sm" aria-label="WG key port" bind:value={wgKeyPort} placeholder="KeyPort" class="font-mono w-15" />
    {/if}
    <Button color="primary" size="sm" onclick={start} disabled={starting}>
      {starting ? 'Starting…' : 'Start Listener'}
    </Button>
  </div>

  {#if listenerError || listenerStatus || (isWG && wgTunIP)}
    <div class="flex flex-wrap items-center gap-3 border-b border-line px-3 py-2 text-xs">
      {#if isWG && wgTunIP && !wgClientConfig}
        <Button color="dark" size="xs" onclick={downloadWGConfig}>Download Client Config</Button>
      {/if}
      {#if listenerStatus}<span class="text-success-500">{listenerStatus}</span>{/if}
      {#if listenerError}<span class="text-danger-500">{listenerError}</span>{/if}
    </div>
  {/if}
  {#if wgClientConfig}
    <div class="border-b border-line px-3 py-2 text-xs">
      <div class="font-semibold mb-1">WireGuard Client Config</div>
      <pre class="bg-canvas p-2 rounded overflow-x-auto text-xs">{JSON.stringify(wgClientConfig, null, 2)}</pre>
    </div>
  {/if}

  <div class="flex-1 min-h-0">
    <DataTable
      data={jobRows}
      {columns}
      keyField="_rowKey"
      loading={jobs.loading}
      error={jobs.error && !jobs.loading ? jobs.error : null}
      emptyState={{ icon: 'headphones', title: 'No active jobs' }}
      rowStyle={(job) => entityColorStyle(entityColors.data, 'listener', String(job._id))}
    >
      {#snippet children(job, col)}
        {#if col.key === '_id' || col.key === '_port'}
          <span class="font-mono">{job[col.key]}</span>
        {:else if col.key === '_name'}
          <span class="flex items-center gap-2">
            {job._name}
            {#if job._isStager}<Badge variant="warning" size="xs">stager</Badge>{/if}
          </span>
        {:else if col.key === '_profile'}
          {#if job._profile}<span class="font-mono">{job._profile}</span>{:else}<span class="text-fg-muted">-</span>{/if}
        {:else if col.key === '_tags'}
          <EntityTagBadges entityType="listener" entityID={String(job._id)} showEmpty />
        {:else if col.key === '_actions'}
          <div class="flex gap-2 justify-end">
            <Button color="dark" size="xs" icon="tag" onclick={() => tagsModal.openTags('listener', String(job._id), job._name || `Job #${job._id}`)}>Tags</Button>
            <Button color="dark" size="xs" icon="message-square" onclick={() => commentsModal.openComments('listener', String(job._id), job._name || `Job #${job._id}`)}>Comments</Button>
            <Button color="red" size="xs" onclick={() => kill(job._id)}>Kill</Button>
          </div>
        {:else}
          {job[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
