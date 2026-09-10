<script>
  import { getTaskCallPayloads, listSessionTasks } from '../../api/sessionTasks.js';
  import { errorMessage } from '../../utils/errors.js';
  import { formatBytes, formatDateTime } from '../../utils/formats.js';
  import DataTable from '$components/patterns/DataTable.svelte';
  import Button from '$components/ui/Button.svelte';
  import InlineImage from '$components/ui/InlineImage.svelte';
  import { decodeTextPayload, inferKind, normalizeKeys, parsePayload, requestSummary, shortMethod, statusChip } from './sessionTaskRender.js';

  let { sessionID = '', active = true } = $props();

  let tasks = $state([]);
  let loading = $state(false);
  let error = $state('');
  let expandedChain = $state('');
  let payloads = $state({});
  let payloadErrors = $state({});
  let loadingChain = $state('');
  let loadSeq = 0;

  async function load() {
    if (!sessionID) return;
    const seq = ++loadSeq;
    loading = true;
    error = '';
    try {
      const nextTasks = await listSessionTasks(sessionID, 100);
      if (seq !== loadSeq) return;
      tasks = nextTasks;
      if (expandedChain && !tasks.some((t) => t.chainRef === expandedChain)) expandedChain = '';
    } catch (err) {
      if (seq !== loadSeq) return;
      error = errorMessage(err);
      tasks = [];
      expandedChain = '';
    } finally {
      if (seq === loadSeq) loading = false;
    }
  }

  $effect(() => {
    if (sessionID) {
      loadSeq++;
      tasks = [];
      error = '';
      expandedChain = '';
      payloads = {};
      payloadErrors = {};
      loadingChain = '';
    }
  });

  $effect(() => {
    if (active && sessionID) load();
  });

  async function toggle(task) {
    if (expandedChain === task.chainRef) {
      expandedChain = '';
      return;
    }
    expandedChain = task.chainRef;
    if (payloads[task.chainRef]) return;
    const nextErrors = { ...payloadErrors };
    delete nextErrors[task.chainRef];
    payloadErrors = nextErrors;
    loadingChain = task.chainRef;
    try {
      const rows = await getTaskCallPayloads(task.chainRef);
      payloads = { ...payloads, [task.chainRef]: rows };
    } catch (err) {
      payloadErrors = { ...payloadErrors, [task.chainRef]: errorMessage(err) };
    } finally {
      if (loadingChain === task.chainRef) loadingChain = '';
    }
  }

  function fallbackText(base64) {
    if (!base64) return '(no payload)';
    if (base64.length <= 2048) return base64;
    return `${base64.slice(0, 2048)}\n… truncated (${base64.length} base64 chars total)`;
  }

  const columns = [
    { key: '_time', label: 'Time', width: 150, sortable: false },
    { key: '_method', label: 'Method', width: 200 },
    { key: '_status', label: 'Status', width: 110, sortable: false },
    { key: '_bytes', label: 'Bytes', width: 90, sortable: false },
    { key: '_stage', label: 'Run', width: 160 },
    { key: '_actions', label: '', width: 90, sortable: false },
  ];

  let rows = $derived(tasks.map((task) => ({
    ...task,
    _time: formatDateTime(task.ts),
    _method: shortMethod(task.method),
    _status: statusChip(task),
    _bytes: formatBytes((task.requestBytes || 0) + (task.responseBytes || 0)),
    _stage: task.runId || task.stageId || '-',
  })));

  const processColumns = [
    { key: 'pid', label: 'PID', width: 80 },
    { key: 'ppid', label: 'PPID', width: 80 },
    { key: 'executable', label: 'Executable', width: 220 },
    { key: 'owner', label: 'Owner', width: 150 },
  ];
  const fileColumns = [
    { key: 'name', label: 'Name', width: 300 },
    { key: 'size', label: 'Size', width: 100 },
    { key: 'mode', label: 'Mode', width: 100 },
    { key: 'modTime', label: 'Modified', width: 180 },
  ];
  const envColumns = [
    { key: 'key', label: 'Key', width: 200 },
    { key: 'value', label: 'Value', width: 400 },
  ];
  const serviceColumns = [
    { key: 'name', label: 'Name', width: 200 },
    { key: 'displayname', label: 'Display name', width: 260 },
    { key: 'status', label: 'Status', width: 90 },
    { key: 'startup', label: 'Startup', width: 110 },
  ];
  const netstatColumns = [
    { key: 'protocol', label: 'Protocol', width: 90 },
    { key: 'localaddr', label: 'Local', width: 200 },
    { key: 'remoteaddr', label: 'Remote', width: 200 },
    { key: 'state', label: 'State', width: 110 },
  ];

  function formatSockAddr(addr) {
    if (!addr || typeof addr !== 'object') return addr || '-';
    const ip = addr.ip || '';
    const port = addr.port ?? '';
    return port === '' ? ip : `${ip}:${port}`;
  }

  let expandedMethod = $derived(tasks.find((t) => t.chainRef === expandedChain)?.method || '');
  let decodedPayloads = $derived((payloads[expandedChain] || []).map((row, index) => {
    const raw = parsePayload(row.json);
    const data = raw ? normalizeKeys(raw) : null;
    const kind = row.direction === 'response' ? (row.kind || inferKind(expandedMethod, data || {})) : 'json';
    if (kind === 'files' && data) {
      data.files = (data.files || []).map((f, i) => ({
        ...f,
        _key: String(i),
        size: formatBytes(f.size),
        modTime: formatDateTime(Number(f.modTime) * 1000),
      }));
    }
    if (kind === 'processes' && data) {
      data.processes = (data.processes || []).map((proc, i) => ({ ...proc, _key: String(i) }));
    }
    if (kind === 'env' && data) {
      data.variables = (data.variables || []).map((v, i) => ({ ...v, _key: String(i) }));
    }
    if (kind === 'services' && data) {
      data.details = (data.details || []).map((svc, i) => ({ ...svc, _key: String(i) }));
    }
    if (kind === 'netstat' && data) {
      data.entries = (data.entries || []).map((entry, i) => ({
        ...entry,
        _key: String(i),
        localaddr: formatSockAddr(entry.localaddr),
        remoteaddr: formatSockAddr(entry.remoteaddr),
        state: entry.skstate,
      }));
    }
    return { row, kind, data, index, summary: row.direction === 'request' ? requestSummary(data) : [] };
  }));
</script>

<div class="flex-1 flex flex-col overflow-hidden">
  <div class="flex items-center justify-between px-3 py-1 border-b border-line">
    <span class="text-xs font-medium">Session tasks</span>
    <Button color="dark" size="xs" disabled={loading} onclick={load}>{loading ? '...' : 'Refresh'}</Button>
  </div>

  {#if tasks.length >= 100}
    <div class="px-3 py-1 text-xs text-fg-muted">Showing the latest 100 calls.</div>
  {/if}

  <div class="flex-1 overflow-auto">
    <DataTable
      data={rows}
      {columns}
      keyField="id"
      defaultSort={{ key: 'ts', dir: 'desc' }}
      {loading}
      error={error || null}
      emptyState={{ title: 'No recorded calls for this session yet.' }}
    >
      {#snippet children(item, col)}
        {#if col.key === '_status'}
          <span class="font-semibold {item._status.class}">{item._status.label}</span>
        {:else if col.key === '_actions'}
          <Button color="dark" size="xs" onclick={() => toggle(item)}>
            {expandedChain === item.chainRef ? 'Hide' : 'View'}
          </Button>
        {:else}
          {item[col.key] ?? '-'}
        {/if}
      {/snippet}
    </DataTable>
  </div>

  {#if expandedChain}
    <div class="shrink-0 border-t-2 border-brand max-h-96 overflow-y-auto bg-chrome">
      <div class="flex items-center justify-between px-3 py-1 border-b border-line bg-surface-50 sticky top-0 z-10">
        <span class="text-xs font-medium font-mono">{shortMethod(expandedMethod)} · {expandedChain.slice(0, 8)}</span>
        <Button color="dark" size="xs" onclick={() => (expandedChain = '')}>Close</Button>
      </div>

      {#if loadingChain === expandedChain}
        <div class="p-3 text-xs text-fg-muted">Loading…</div>
      {:else if payloadErrors[expandedChain]}
        <div class="p-3 text-xs text-danger-500">{payloadErrors[expandedChain]}</div>
      {:else if decodedPayloads.length === 0}
        <div class="p-3 text-xs text-fg-muted">(no payload)</div>
      {:else}
        {#each decodedPayloads as { row, kind, data, index, summary } (index)}
          <div class="border-b border-line">
            <div class="px-3 py-1 text-xs font-medium text-fg-muted">
              {row.direction === 'request' ? 'Request' : 'Response'}
            </div>
            <div class="p-3">
              {#if row.direction === 'request'}
                {#if summary.length === 0}
                  <div class="text-xs text-fg-muted">(no parameters)</div>
                {:else}
                  <!-- eslint-disable-next-line local/no-arbitrary-tailwind-value -->
                  <dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                    {#each summary as item, itemIndex (itemIndex)}
                      <dt class="text-fg-muted">{item.label}</dt>
                      <dd class="font-mono break-all">{item.value}</dd>
                    {/each}
                  </dl>
                {/if}
              {:else if data && kind === 'image' && data.data}
                <InlineImage src={'data:image/png;base64,' + data.data} alt="Screenshot" maxHeight="16rem" />
              {:else if data && kind === 'processes'}
                <DataTable data={data.processes || []} columns={processColumns} keyField="_key" />
              {:else if data && kind === 'files'}
                <DataTable data={data.files || []} columns={fileColumns} keyField="_key" />
              {:else if data && kind === 'env'}
                <DataTable data={data.variables || []} columns={envColumns} keyField="_key" />
              {:else if data && kind === 'services'}
                <DataTable data={data.details || []} columns={serviceColumns} keyField="_key" />
              {:else if data && kind === 'netstat'}
                <DataTable data={data.entries || []} columns={netstatColumns} keyField="_key" />
              {:else if data && kind === 'text'}
                {@const decoded = decodeTextPayload(data)}
                <pre class="text-xs whitespace-pre-wrap break-all">{decoded.text}</pre>
                {#if decoded.err}
                  <div class="text-xs text-danger-500">{decoded.err}</div>
                {/if}
              {:else if data}
                <pre class="text-xs whitespace-pre-wrap break-all">{JSON.stringify(data, null, 2)}</pre>
              {:else}
                <pre class="text-xs whitespace-pre-wrap break-all">{fallbackText(row.base64)}</pre>
              {/if}
            </div>
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>
