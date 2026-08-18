<script>
  import { AnsiUp } from 'ansi_up';
  import { CancelBeaconTask, GetBeaconTaskOutput } from '../../api/agents.js';
  import { dialog } from '../../stores/ui/dialog.svelte.js';
  import { errorMessage } from '../../utils/errors.js';
  import { formatBytes, formatDateTime } from '../../utils/formats.js';
  import { shortAgentID } from '../../utils/agents.js';
  import { useBeaconTasks } from '$stores/perAgent/beaconTasks.svelte.js';
  import { agentTabs } from '$stores/agentTabs.svelte.js';
  import InlineImage from '$components/ui/InlineImage.svelte';
  import DataTable from '$components/patterns/DataTable.svelte';
  import Button from '$components/ui/Button.svelte';

  let { beaconID = '' } = $props();

  let expandedTask = $state('');
  let taskOutput = $state(null);
  let cancelingTask = $state('');
  let openingTask = $state('');
  const ansiUp = new AnsiUp();
  ansiUp.use_classes = false;

  let taskStore = $derived(useBeaconTasks(beaconID));

  $effect(() => {
    taskStore.acquire();
    return () => taskStore.release();
  });

  async function toggleTask(task) {
    if (expandedTask === task.ID) { expandedTask = ''; return; }
    expandedTask = task.ID;
    taskOutput = null;
    try {
      taskOutput = await GetBeaconTaskOutput(task.ID);
    } catch (cause) {
      taskOutput = { type: "text", textOutput: errorMessage(cause, 'Failed to load task: ') };
    }
  }

  async function cancelTask(event, task) {
    event.stopPropagation();
    const confirmed = await dialog.confirm(`Cancel pending task ${shortAgentID(task.ID)}?`);
    if (!confirmed) return;
    cancelingTask = task.ID;
    try {
      await CancelBeaconTask(task.ID);
    } catch (cause) {
      await dialog.alert(errorMessage(cause, 'Could not cancel task: '));
    } finally {
      cancelingTask = '';
    }
  }

  function taskState(value) {
    return String(value || '').toLowerCase();
  }

  function taskStateClass(value) {
    const state = taskState(value);
    if (state === 'completed') return 'text-success-500';
    if (state === 'sent' || state === 'pending') return 'text-warning-500';
    return 'text-fg-muted';
  }

  function taskKind(task) {
    const desc = String(task.Description || '').toLowerCase()
    if (desc.startsWith('ps') || desc.startsWith('psreq')) return 'processes'
    if (desc.startsWith('screenshot')) return 'screenshot'
    if (desc.startsWith('netstat')) return 'netstat'
    if (desc.startsWith('ifconfig')) return 'ifconfig'
    if (desc.startsWith('services') || desc.startsWith('servicesreq')) return 'services'
    if (desc.startsWith('env')) return 'env'
    if (desc.startsWith('grep')) return 'grep'
    if (desc.startsWith('ls') || desc.startsWith('pwd') || desc.startsWith('lsreq')) return 'files'
    return 'raw'
  }

  async function openTaskInTab(event, task, kind) {
    event.stopPropagation()
    openingTask = task.ID
    try {
      const output = await GetBeaconTaskOutput(task.ID)
      if (!output) return
      if (kind === 'processes' && output.processes) {
        agentTabs.openOrUpdateTab(beaconID, 'processExplorer', { staticData: output.processes })
      } else if (kind === 'netstat') {
        agentTabs.openOrUpdateTab(beaconID, 'netstat', { staticOutput: output.textOutput })
      } else if (kind === 'screenshot' && output.imageData) {
        agentTabs.openOrUpdateTab(beaconID, 'screenshot', { staticBase64: output.imageData })
      } else if (kind === 'services' && output.services) {
        agentTabs.openOrUpdateTab(beaconID, 'services', { staticServices: output.services })
      } else if (kind === 'files' && output.files) {
        agentTabs.openOrUpdateTab(beaconID, 'fileBrowser', { staticData: { files: output.files, path: output.path || '' } })
      } else if (kind === 'env' && output.envVars) {
        agentTabs.openOrUpdateTab(beaconID, 'env', { staticData: output.envVars })
      } else if (kind === 'ifconfig') {
        agentTabs.openOrUpdateTab(beaconID, 'ifconfig', { staticOutput: output.textOutput || '' })
      } else if (kind === 'grep') {
        agentTabs.openOrUpdateTab(beaconID, 'grep', { staticOutput: output.textOutput || '' })
      } else {
        toggleTask(task)
      }
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Failed to open task: '))
    } finally {
      openingTask = ''
    }
  }

  const columns = [
    { key: '_shortID', label: 'ID', width: 72 },
    { key: 'Description', label: 'Message Type', width: 130 },
    { key: '_stateClass', label: 'State', width: 70, sortable: false },
    { key: 'CreatedAt', label: 'Created', width: 130 },
    { key: 'SentAt', label: 'Sent', width: 130 },
    { key: 'CompletedAt', label: 'Completed', width: 130 },
    { key: '_actions', label: '', width: 150, sortable: false },
  ]

  const psColumns = [
    { key: 'Pid', label: 'PID', width: 80 },
    { key: 'Ppid', label: 'PPID', width: 80 },
    { key: 'Executable', label: 'Executable', width: 200 },
    { key: 'Owner', label: 'Owner', width: 150 },
  ]

  const netstatColumns = [
    { key: 'Protocol', label: 'Proto', width: 60 },
    { key: 'localAddr', label: 'Local Address', width: 160 },
    { key: 'remoteAddr', label: 'Remote Address', width: 160 },
    { key: 'State', label: 'State', width: 120 },
  ]

  const svcColumns = [
    { key: 'name', label: 'Name', width: 220 },
    { key: 'display', label: 'Display Name', width: 250 },
    { key: 'status', label: 'Status', width: 100 },
    { key: 'startup', label: 'Startup Type', width: 100 },
  ]

  const fileColumns = [
    { key: 'iconStr', label: 'Name', width: 300 },
    { key: 'sizeStr', label: 'Size', width: 100 },
    { key: 'modTimeStr', label: 'Last Modified', width: 200 },
  ]

  const envColumns = [
    { key: 'Key', label: 'Key', width: 200 },
    { key: 'Value', label: 'Value', width: 400 },
  ]

  function translateServiceStatus(v) {
    const m = { 1: 'Stopped', 2: 'Start Pending', 3: 'Stop Pending', 4: 'Running', 5: 'Continue Pending', 6: 'Pause Pending', 7: 'Paused' }
    return m[v] || `Unknown (${v})`
  }

  function translateServiceStartup(v) {
    const m = { 0: 'Boot', 1: 'System', 2: 'Automatic', 3: 'Manual', 4: 'Disabled' }
    return m[v] || `Unknown (${v})`
  }

  let netstatRows = $derived(
    (taskOutput?.netstatEntries || []).map((e, idx) => ({
      ...e,
      _key: `${e.LocalAddr?.Ip || ''}:${e.LocalAddr?.Port || ''}-${e.RemoteAddr?.Ip || ''}:${e.RemoteAddr?.Port || ''}-${idx}`,
      localAddr: `${e.LocalAddr?.Ip || ''}:${e.LocalAddr?.Port || ''}`,
      remoteAddr: `${e.RemoteAddr?.Ip || ''}:${e.RemoteAddr?.Port || ''}`,
      State: e.SkState || '',
    }))
  )

  let tableData = $derived(taskStore.state.tasks.map((t) => ({
    ...t,
    _shortID: shortAgentID(t.ID),
    _stateClass: taskStateClass(t.State),
    _kind: taskKind(t),
  })))

  function openFullTab(beaconID, tabType, meta) {
    agentTabs.openOrUpdateTab(beaconID, tabType, meta)
  }

  function handleRowClick(item) {
    expandedTask = item.ID;
  }

  function taskRowClass(item) {
    return expandedTask === item.ID ? 'bg-brand/10' : ''
  }
</script>

<div class="flex flex-col h-full bg-canvas text-fg">
  <div class="flex items-center justify-between gap-2 px-3 py-1 border-b border-line bg-chrome shrink-0">
    <span class="text-xs font-medium text-fg-muted">Beacon Tasks</span>
    <Button color="dark" size="sm" onclick={() => taskStore?.refresh()}>Refresh</Button>
  </div>

  <div class="flex-1 overflow-auto flex flex-col">
    <div class="flex-1 overflow-auto">
      <DataTable
        data={tableData}
        {columns}
        keyField="ID"
        defaultSort={{ key: 'CreatedAt', dir: 'desc' }}
        loading={taskStore.state.loading && taskStore.state.tasks.length === 0}
        error={taskStore.state.error || null}
        emptyState={{ title: 'No tasks have been queued for this beacon.' }}
        onRowClick={handleRowClick}
        rowClass={taskRowClass}
      >
        {#snippet children(item, col)}
          {#if col.key === '_shortID'}
            <span class="font-mono">{item._shortID}</span>
          {:else if col.key === '_stateClass'}
            <span class="font-semibold {item._stateClass}">{item.State || '-'}</span>
          {:else if col.key === 'CreatedAt'}
            <span class="font-mono">{formatDateTime(item.CreatedAt)}</span>
          {:else if col.key === 'SentAt'}
            <span class="font-mono">{formatDateTime(item.SentAt)}</span>
          {:else if col.key === 'CompletedAt'}
            <span class="font-mono">{formatDateTime(item.CompletedAt)}</span>
          {:else if col.key === '_actions'}
            <div class="flex items-center gap-1 min-h-6">
              {#if taskState(item.State) === 'pending'}
                <Button
                  color="red"
                  size="xs"
                  disabled={cancelingTask === item.ID}
                  onclick={(event) => cancelTask(event, item)}
                >
                  {cancelingTask === item.ID ? '...' : 'Cancel'}
                </Button>
              {/if}
              {#if item._kind && item._kind !== 'raw' && taskState(item.State) === 'completed'}
                <Button
                  color="dark"
                  size="xs"
                  disabled={openingTask === item.ID}
                  onclick={(event) => openTaskInTab(event, item, item._kind)}
                >
                  {openingTask === item.ID ? '...' : 'Open'}
                </Button>
              {/if}
            </div>
          {:else if col.key === 'Description'}
            {String(item.Description || '').replace(/Req$/, '') || '-'}
          {:else}
            {item[col.key] ?? '-'}
          {/if}
        {/snippet}
      </DataTable>
    </div>

    {#if expandedTask}
      <div class="shrink-0 border-t-2 border-brand max-h-80 overflow-y-auto bg-chrome">
        <div class="flex items-center justify-between px-3 py-1 border-b border-line bg-surface-50 sticky top-0 z-10">
          <span class="text-xs font-medium">Task {shortAgentID(expandedTask)}</span>
          <Button color="dark" size="xs" onclick={() => expandedTask = ''}>Close</Button>
        </div>
        {#if !taskOutput}
          <div class="p-4 text-fg-muted text-xs">Loading...</div>
        {:else if taskOutput.type === "image"}
          <InlineImage src={"data:image/png;base64," + taskOutput.imageData} alt="Screenshot" maxHeight="16rem" />
        {:else if taskOutput.type === "processes"}
          <div class="flex flex-col">
            <div class="flex items-center justify-between px-3 py-1 border-b border-line bg-surface-50">
              <span class="text-xs text-fg-muted">{taskOutput.processes?.length || 0} processes</span>
              <Button color="dark" size="xs" onclick={() => openFullTab(beaconID, "processExplorer", {staticData: taskOutput.processes})}>Open Process Explorer</Button>
            </div>
            <div class="max-h-64 overflow-y-auto">
              <DataTable data={taskOutput.processes || []} columns={psColumns} keyField="Pid" emptyState={{ title: "No processes found." }} />
            </div>
          </div>
        {:else if taskOutput.type === "netstat"}
          <div class="flex flex-col">
            <div class="flex items-center justify-between px-3 py-1 border-b border-line bg-surface-50">
              <span class="text-xs text-fg-muted">{taskOutput.netstatEntries?.length || 0} connections</span>
              <Button color="dark" size="xs" onclick={() => openFullTab(beaconID, 'networkConnections', {staticEntries: taskOutput.netstatEntries})}>Open Network Tab</Button>
            </div>
            <div class="max-h-64 overflow-y-auto">
              <DataTable
                data={netstatRows}
                columns={netstatColumns}
                keyField="_key"
                emptyState={{ title: "No network connections found." }}
              />
            </div>
          </div>
         {:else if taskOutput.type === "services"}
          <div class="flex flex-col">
            <div class="flex items-center justify-between px-3 py-1 border-b border-line bg-surface-50">
              <span class="text-xs text-fg-muted">{taskOutput.services?.length || 0} services</span>
              <Button color="dark" size="xs" onclick={() => openFullTab(beaconID, 'services', {staticServices: taskOutput.services})}>Open Services</Button>
            </div>
            <div class="max-h-64 overflow-y-auto">
              <DataTable
                data={(taskOutput.services || []).map((s) => ({
                  name: s.Name || s.name || '',
                  display: s.DisplayName || s.displayName || '',
                  status: translateServiceStatus(s.Status ?? s.status),
                  startup: translateServiceStartup(s.StartupType ?? s.startupType),
                }))}
                columns={svcColumns}
                keyField="name"
                emptyState={{ title: "No services found." }}
              />
            </div>
          </div>
        {:else if taskOutput.type === "filelist"}
          <div class="flex flex-col">
            <div class="flex items-center justify-between px-3 py-1 border-b border-line bg-surface-50">
              <span class="text-xs text-fg-muted">{taskOutput.files?.length || 0} files in {taskOutput.path || ''}</span>
              <Button color="dark" size="xs" onclick={() => openFullTab(beaconID, 'fileBrowser', {staticData: {files: taskOutput.files, path: taskOutput.path || ''}})}>Open File Browser</Button>
            </div>
            <div class="max-h-64 overflow-y-auto">
              <DataTable
                data={(taskOutput.files || []).map((f) => ({
                  _name: f.Name || f.name || '',
                  iconStr: (f.IsDir || f.isDir) ? `📁 ${f.Name || f.name}` : `📄 ${f.Name || f.name}`,
                  sizeStr: (f.IsDir || f.isDir) ? '-' : formatBytes(f.Size || f.size),
                  modTimeStr: formatDateTime(f.ModTime || f.modTime || 0),
                }))}
                columns={fileColumns}
                keyField="_name"
                emptyState={{ title: "No files found." }}
              />
            </div>
          </div>
        {:else if taskOutput.type === "env"}
          <div class="flex flex-col">
            <div class="flex items-center justify-between px-3 py-1 border-b border-line bg-surface-50">
              <span class="text-xs text-fg-muted">{taskOutput.envVars?.length || 0} variables</span>
              <Button color="dark" size="xs" onclick={() => openFullTab(beaconID, 'env', {staticData: taskOutput.envVars})}>Open Env</Button>
            </div>
            <div class="max-h-64 overflow-y-auto">
              <DataTable
                data={taskOutput.envVars || []}
                columns={envColumns}
                keyField="Key"
                emptyState={{ title: "No env vars found." }}
              />
            </div>
          </div>
        {:else}
          <pre class="max-h-80 m-0 p-4 overflow-auto whitespace-pre-wrap break-words text-fg text-sm font-mono">{@html ansiUp.ansi_to_html(taskOutput.textOutput)}</pre>
        {/if}
      </div>
    {/if}
  </div>
</div>
