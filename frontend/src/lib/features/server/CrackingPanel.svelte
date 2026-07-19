<script>
  import Button from '$components/ui/Button.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import PresetPicker from '$components/forms/PresetPicker.svelte'
  import TextArea from '$components/ui/TextArea.svelte'
  import Select from '$components/ui/Select.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { crackstations } from '$stores/resources/crackstations.svelte.js'
  import { crackFiles } from '$stores/resources/crackFiles.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(crackstations, crackFiles)
  import {
    CrackSubmitJob,
    CrackTaskByID,
    CrackTaskCancel,
    CrackFileUploadFromPath,
    CrackFileDelete,
  } from '../../api/crack.js'
  import { OpenFileDialog } from '../../api/runtime.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'

  let { embedded = false, onclose } = $props()

  let subTab = $state('jobs')
  let actionError = $state('')
  let jobResult = $state('')
  let trackedJobs = $state([])
  let pollingJobs = $state(false)

  let uploadFileType = $state(1)
  let hashType = $state(1000)
  let hashes = $state('')
  let submitting = $state(false)

  const BRUTE_FORCE_ATTACK_MODE = 3
  const CRACK_FILE_TYPE_OPTIONS = [
    { value: 1, label: 'Wordlist' },
    { value: 2, label: 'Rules' },
    { value: 3, label: 'Markov hcstat2' },
  ]
  const CRACK_FILE_TYPE_LABELS = Object.fromEntries(CRACK_FILE_TYPE_OPTIONS.map((option) => [option.value, option.label]))

  const HASH_OPTIONS = [
    { value: 1000, label: 'NTLM' },
    { value: 2100, label: 'Domain Cached Credentials (DCC)' },
    { value: 3000, label: 'LM' },
    { value: 3200, label: 'bcrypt' },
    { value: 400, label: 'phpass' },
    { value: 500, label: 'MD5' },
    { value: 100, label: 'SHA1' },
    { value: 1400, label: 'SHA256' },
    { value: 1700, label: 'SHA512' },
  ]

  let stations = $derived(crackstations.data || [])
  let files = $derived(crackFiles.data || [])
  let fileRows = $derived(files.map((file, index) => ({
    _rowKey: file.ID || file.id || file.Name || file.name || index,
    _id: file.ID || file.id,
    _name: file.Name || file.name || '-',
    _type: crackFileTypeLabel(file.Type ?? file.type),
    _size: file.UncompressedSize ?? file.uncompressedSize ?? '-',
  })))
  let stationRows = $derived(stations.map((station, index) => ({
    _rowKey: station.ID || station.id || station.Name || station.name || index,
    _name: station.Name || station.name || '-',
    _operator: station.OperatorName || station.operatorName || '-',
    _osArch: `${station.GOOS || station.goos || '-'}/${station.GOARCH || station.goarch || '-'}`,
    _hashcat: station.HashcatVersion || station.hashcatVersion || '-',
    _benchmarks: `${Object.keys(station.Benchmarks || station.benchmarks || {}).length} hashes`,
  })))
  let trackedJobRows = $derived(trackedJobs.map((job) => ({
    _rowKey: job.id,
    _raw: job,
    _id: shortID(job.id),
    _hashType: job.hashType || '-',
    _state: jobState(job),
    _created: fmtTime(job.createdAt),
    _started: fmtTime(job.startedAt),
    _completed: fmtTime(job.completedAt),
    _error: job.err || '',
  })))

  const fileColumns = [
    { key: '_name', label: 'Name' },
    { key: '_type', label: 'Type', width: 120 },
    { key: '_size', label: 'Size', width: 120 },
    { key: '_actions', label: 'Actions', width: 90, sortable: false },
  ]
  const stationColumns = [
    { key: '_name', label: 'Name' },
    { key: '_operator', label: 'Operator' },
    { key: '_osArch', label: 'OS/Arch', width: 120 },
    { key: '_hashcat', label: 'Hashcat', width: 120 },
    { key: '_benchmarks', label: 'Benchmarks', width: 130 },
  ]
  const jobColumns = [
    { key: '_id', label: 'Task', width: 110 },
    { key: '_hashType', label: 'Type', width: 90 },
    { key: '_state', label: 'State', width: 100 },
    { key: '_created', label: 'Created', width: 150 },
    { key: '_started', label: 'Started', width: 150 },
    { key: '_completed', label: 'Completed', width: 150 },
    { key: '_error', label: 'Error' },
    { key: '_actions', label: '', width: 130, sortable: false },
  ]

  $effect(() => {
    if (!trackedJobs.some((job) => !job.completedAt && !job.err)) return
    let cancelled = false
    const tick = () => {
      if (!cancelled) refreshTrackedJobs()
    }
    tick()
    const timer = setInterval(tick, 5000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  })

  async function submitJob() {
    if (!hashes.trim()) { actionError = 'Enter at least one hash'; return }
    submitting = true
    actionError = ''
    jobResult = ''
    try {
      const hashList = hashes.trim().split('\n').map(h => h.trim()).filter(Boolean)
      const resp = await CrackSubmitJob(BRUTE_FORCE_ATTACK_MODE, hashType, hashList, [])
      const job = resp?.Job || resp?.job
      if (job) trackJob(job)
      jobResult = job ? `Job submitted: ${job.ID || job.id}` : 'Job submitted (no ID returned)'
      hashes = ''
    } catch (err) {
      actionError = errorMessage(err, 'Submit failed: ')
    } finally {
      submitting = false
    }
  }

  function applyJobPreset(values) {
    if (values.hashType != null) hashType = Number(values.hashType)
    if (values.uploadFileType != null) uploadFileType = Number(values.uploadFileType)
  }

  async function uploadFile() {
    let path
    try { path = await OpenFileDialog('Select file to upload') } catch { return }
    if (!path) return
    try {
      await CrackFileUploadFromPath(path, uploadFileType)
      await crackFiles.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Upload failed: ')
    }
  }

  async function deleteFile(id) {
    if (!(await dialog.confirm('Delete this file?', 'Confirm'))) return
    try {
      await CrackFileDelete(id)
      await crackFiles.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Delete failed: ')
    }
  }

  async function refreshTrackedJobs() {
    if (pollingJobs) return
    const active = trackedJobs.filter((job) => job.id && !job.completedAt && !job.err)
    if (active.length === 0) return
    pollingJobs = true
    try {
      const updates = await Promise.all(active.map((job) => CrackTaskByID(job.id, job.hostUUID || '')))
      for (const update of updates) trackJob(update)
    } catch (err) {
      actionError = errorMessage(err, 'Task refresh failed: ')
    } finally {
      pollingJobs = false
    }
  }

  async function cancelJob(job) {
    if (!job?.id || !(await dialog.confirm('Cancel this crack task?', 'Confirm'))) return
    try {
      await CrackTaskCancel(job.id, job.hostUUID || '')
      await refreshTrackedJobs()
    } catch (err) {
      actionError = errorMessage(err, 'Cancel failed: ')
    }
  }

  function trackJob(job) {
    const normalized = normalizeJob(job)
    if (!normalized.id) return
    trackedJobs = [
      normalized,
      ...trackedJobs.filter((current) => current.id !== normalized.id),
    ].slice(0, 25)
  }

  function normalizeJob(job) {
    const command = job.Command || job.command || {}
    const completedAt = job.CompletedAt ?? job.completedAt ?? ''
    return {
      id: job.ID || job.id || '',
      hostUUID: job.HostUUID || job.hostUUID || '',
      createdAt: job.CreatedAt || job.createdAt || '',
      startedAt: job.StartedAt || job.startedAt || '',
      completedAt,
      err: job.Err || job.err || '',
      hashType: command.HashType || command.hashType || hashType,
      raw: job,
    }
  }

  function jobState(job) {
    if (job.err) return 'Failed'
    if (job.completedAt) return 'Completed'
    if (job.startedAt) return 'Running'
    return 'Queued'
  }

  function shortID(value) {
    return value ? String(value).slice(0, 8) : '-'
  }

  function fmtTime(value) {
    if (!value) return '-'
    const asNumber = Number(value)
    if (Number.isFinite(asNumber)) return new Date(asNumber * 1000).toLocaleString()
    return String(value)
  }

  function crackFileTypeLabel(fileType) {
    return CRACK_FILE_TYPE_LABELS[fileType] ?? fileType ?? '-'
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Cracking'} icon={embedded ? '' : 'zap'}>
  <Toolbar class="justify-between">
    <div class="flex gap-1">
      <Button color={subTab === 'jobs' ? 'primary' : 'dark'} size="sm" onclick={() => { subTab = 'jobs' }}>Jobs</Button>
      <Button color={subTab === 'files' ? 'primary' : 'dark'} size="sm" onclick={() => { subTab = 'files' }}>Files</Button>
      <Button color={subTab === 'workers' ? 'primary' : 'dark'} size="sm" onclick={() => { subTab = 'workers' }}>Workers</Button>
    </div>
    <div class="flex gap-1">
      {#if subTab === 'files'}
        <div class="w-36">
          <Select bind:value={uploadFileType} options={CRACK_FILE_TYPE_OPTIONS} size="sm" aria-label="Crack file type" />
        </div>
        <Button color="primary" size="sm" icon="upload" onclick={uploadFile}>Upload</Button>
      {/if}
      <Button color="dark" size="sm" onclick={() => { crackstations.refresh(); crackFiles.refresh() }}>Refresh</Button>
    </div>
  </Toolbar>

  <div class="flex-1 min-h-0 p-3 text-xs">
    {#if actionError}
      <div class="mb-3 rounded border border-danger-500 bg-danger-500/10 p-2 text-danger-500">{actionError}</div>
    {/if}

    {#if subTab === 'jobs'}
      <div class="grid gap-3">
        <section class="border border-line bg-panel rounded">
          <div class="flex items-center justify-between border-b border-line px-3 py-2">
            <span class="font-semibold">Submit Crack Job</span>
            <PresetPicker
              commandPath="cracking/job"
              currentValues={{ hashType, uploadFileType }}
              onapply={applyJobPreset}
            />
          </div>
          <div class="grid gap-2 p-3">
            <Select bind:value={hashType} options={HASH_OPTIONS} aria-label="Hash type" />
            <TextArea
              class="min-h-24 w-full font-mono text-xs resize-y"
              bind:value={hashes}
              rows={6}
              placeholder="One hash per line"
            />
            <div class="flex gap-2">
              <Button color="primary" size="sm" onclick={submitJob} disabled={submitting}>
                {submitting ? 'Submitting...' : 'Submit Job'}
              </Button>
            </div>
          </div>
        </section>
        {#if jobResult}
          <div class="rounded border border-success-500 bg-success-500/10 p-2">{jobResult}</div>
        {/if}
        <section class="min-h-48 border border-line bg-panel rounded">
          <div class="flex items-center justify-between border-b border-line px-3 py-2">
            <span class="font-semibold">Tracked Tasks</span>
            <Button color="dark" size="xs" onclick={refreshTrackedJobs} disabled={pollingJobs || trackedJobs.length === 0}>
              {pollingJobs ? 'Refreshing...' : 'Refresh'}
            </Button>
          </div>
          <div class="min-h-40">
            <DataTable
              data={trackedJobRows}
              columns={jobColumns}
              keyField="_rowKey"
              emptyState={{ icon: 'zap', title: 'No tracked tasks' }}
            >
              {#snippet children(job, col)}
                {#if col.key === '_actions'}
                  <div class="flex justify-end">
                    <Button
                      color="red"
                      size="xs"
                      disabled={job._raw.completedAt || job._raw.err}
                      onclick={() => cancelJob(job._raw)}
                    >
                      Cancel
                    </Button>
                  </div>
                {:else if col.key === '_id' || col.key === '_hashType' || col.key === '_created' || col.key === '_started' || col.key === '_completed'}
                  <span class="font-mono text-fg-muted">{job[col.key]}</span>
                {:else}
                  {job[col.key]}
                {/if}
              {/snippet}
            </DataTable>
          </div>
        </section>
      </div>

    {:else if subTab === 'files'}
      <div class="h-full min-h-0">
        <DataTable
          data={fileRows}
          columns={fileColumns}
          keyField="_rowKey"
          loading={crackFiles.loading}
          error={crackFiles.error && !crackFiles.loading ? crackFiles.error : null}
          emptyState={{ icon: 'file', title: 'No files' }}
        >
          {#snippet children(file, col)}
            {#if col.key === '_name' || col.key === '_size'}
              <span class="font-mono text-fg-muted">{file[col.key]}</span>
            {:else if col.key === '_actions'}
              <div class="flex justify-end">
                <IconButton icon="trash" label="Delete" tooltip="Delete" color="red" size="xs" onclick={() => deleteFile(file._id)} />
              </div>
            {:else}
              {file[col.key]}
            {/if}
          {/snippet}
        </DataTable>
      </div>

    {:else if subTab === 'workers'}
      <div class="h-full min-h-0">
        <DataTable
          data={stationRows}
          columns={stationColumns}
          keyField="_rowKey"
          loading={crackstations.loading}
          error={crackstations.error && !crackstations.loading ? crackstations.error : null}
          emptyState={{ icon: 'cpu', title: 'No registered workers' }}
        >
          {#snippet children(station, col)}
            {#if col.key === '_name' || col.key === '_osArch' || col.key === '_hashcat' || col.key === '_benchmarks'}
              <span class="font-mono text-fg-muted">{station[col.key]}</span>
            {:else}
              {station[col.key]}
            {/if}
          {/snippet}
        </DataTable>
      </div>
    {/if}
  </div>
</Panel>
