<script>
  import Icon from '$components/ui/Icon.svelte'
  import FileViewerModal from './modals/FileViewerModal.svelte'
  import DownloadHistoryModal from './modals/DownloadHistoryModal.svelte'
  import { UploadFiles } from '../../api/agents.js';
  import { onFileDrop } from '../../api/runtime.js';
  import { createFileBrowserActions } from './fileBrowserActions.js';
  import { dialog } from '../../stores/ui/dialog.svelte.js';
  import { contextMenu } from '$stores/ui/contextMenu.svelte.js';
  import { errorMessage } from '../../utils/errors.js';
  import DataTable from '$components/patterns/DataTable.svelte';
  import Button from '$components/ui/Button.svelte';
  import TextInput from '$components/ui/TextInput.svelte';
  import { useFileBrowser } from '$stores/perAgent/fileBrowser.svelte.js';

  let {
    sessionID = "",
    // picker mode: false | 'file' | 'dir'
    //   'file' — single-click a file to emit { path }
    //   'dir'  — header shows "Select this folder" button that emits current path
    picker = false,
    onpick,
    startPath = '',
  } = $props();

  let currentPath = $state('');
  let dropZone = $state();
  let uploading = $state(false);
  let filterText = $state('');
  let viewerData = $state(null); // { filename, data, isBinary } or null
  let historyModalState = $state({ isOpen: false, remotePath: '', sessionID: '' });

  function openFileHistory(file) {
    const name = file.Name || file.name;
    historyModalState = { isOpen: true, remotePath: joinPath(name), sessionID };
  }

  function openGlobalHistory() {
    historyModalState = { isOpen: true, remotePath: '', sessionID };
  }

  let store = $derived(useFileBrowser(sessionID));

  $effect(() => {
    store.acquire();
    return () => store.release();
  });
  $effect(() => {
    currentPath = store.state.path;
  });

  // If picker was opened with a suggested starting path (e.g. re-opening a
  // previously-picked value), navigate there once on mount.
  let seeded = false;
  $effect(() => {
    if (seeded || !picker || !startPath) return;
    seeded = true;
    store.refresh(startPath);
  });

  function emitPick(path) {
    onpick?.({ path });
  }

  // Context menu state

  let tableColumns = [
    { key: "iconStr", label: "Name", width: 300 },
    { key: "_sortSize", label: "Size", width: 100 },
    { key: "typeStr", label: "Type", width: 100 },
    { key: "modTimeStr", label: "Last Modified", width: 250 }
  ];

  let normalizedFiles = $derived((store.state.files || []).map(f => ({
    ...f,
    rawFile: f,
    _name: f.Name || f.name,
    isDir: f.IsDir || f.isDir,
    iconStr: (f.IsDir || f.isDir) ? `📁 ${f.Name || f.name}` : `📄 ${f.Name || f.name}`,
    _sortSize: f.IsDir || f.isDir ? '' : String(f.Size || f.size || 0).padStart(20, '0'),
    sizeStr: (f.IsDir || f.isDir) ? '-' : formatSize(f.Size || f.size),
    typeStr: (f.IsDir || f.isDir) ? 'Directory' : 'File',
    modTimeStr: new Date((f.ModTime || f.modTime || 0) * 1000).toLocaleString()
  })));

  let filteredFiles = $derived(
    !filterText
      ? normalizedFiles
      : normalizedFiles.filter((f) => f._name.toLowerCase().includes(filterText.toLowerCase()))
  );

  onFileDrop(async (x, y, paths) => {
    if (picker) return; // picker mode is read-only — no accidental uploads
    const target = document.elementFromPoint(x, y);
    if (!dropZone?.contains(target) || !paths?.length || uploading) return;

    uploading = true;
    try {
      await UploadFiles(sessionID, store.state.path, paths);
      store.refresh(store.state.path);
    } catch (err) {
      await dialog.alert(errorMessage(err, "Upload failed: "), 'Upload Error');
    } finally {
      uploading = false;
    }
  });

  function formatSize(bytes) {
    if (bytes === 0 || !bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  function handleDoubleClick(file) {
    const isDir = file.IsDir || file.isDir;
    const name = file.Name || file.name;
    if (isDir) {
      let sep = currentPath.includes('\\') ? '\\' : '/';
      let nextPath = currentPath.endsWith(sep) ? currentPath + name : currentPath + sep + name;
      store.refresh(nextPath);
    } else if (picker === 'file') {
      // In file-picker mode, double-click a file also confirms the selection.
      emitPick(joinPath(name));
    }
  }

  function handleRowClick(file) {
    if (!picker) return;
    const isDir = file.IsDir || file.isDir;
    const name = file.Name || file.name;
    // File picker mode: single-click a file confirms. Folders still need a
    // double-click to navigate (avoids "picked the wrong thing" surprises).
    if (picker === 'file' && !isDir) {
      emitPick(joinPath(name));
    }
  }

  // A drive root ("C:\") or filesystem root ("/") — going up from here is invalid
  // and the implant returns an RPC error, so we no-op.
  function isAtRoot(path) {
    return path === "" || path === "/" || /^[A-Za-z]:\\?$/.test(path);
  }

  function goUp() {
    if (isAtRoot(currentPath)) return;

    const isWindows = currentPath.includes('\\') || /^[A-Za-z]:/.test(currentPath);
    const sep = isWindows ? '\\' : '/';

    const parts = currentPath.split(sep);
    if (parts[parts.length - 1] === '') parts.pop(); // trailing separator
    parts.pop();
    let nextPath = parts.join(sep);

    if (isWindows) {
      // Keep the drive root as "C:\" rather than "C:".
      if (/^[A-Za-z]:$/.test(nextPath)) nextPath += '\\';
      else if (nextPath === '') return;
    } else if (nextPath === '') {
      nextPath = '/';
    }

    store.refresh(nextPath);
  }

  const actions = $derived(createFileBrowserActions({
    sessionID, dialog,
    store: { refresh: (p) => store.refresh(p), get: () => store.state },
    getCurrentPath: () => currentPath,
    setViewerData: (v) => { viewerData = v },
  }));
  const downloadDir = (f) => actions.downloadDir(f);
  const downloadFile = (f) => actions.downloadFile(f);
  const uploadFile = () => actions.uploadFile();
  const newFolder = () => actions.newFolder();
  const deleteFile = (f) => actions.deleteFile(f);
  const renameFile = (f) => actions.renameFile(f);
  const viewFile = (f) => actions.viewFile(f);
  const editPermissions = (f) => actions.editPermissions(f);
  const copyFile = (f) => actions.copyFile(f);
  const moveFile = (f) => actions.moveFile(f);
  const joinPath = (name) => actions.joinPath(name);

  function handleRightClick(event, file) {
    const isDir = file.IsDir || file.isDir;
    contextMenu.open({
      x: event.clientX, y: event.clientY,
      sections: [
        { items: [
          ...(isDir
            ? [{ icon: 'download', label: 'Download (tar)', on: () => downloadDir(file) }]
            : [{ icon: 'download', label: 'Download', on: () => downloadFile(file) }]
          ),
          { icon: 'history', label: 'Download History', on: () => openFileHistory(file) },
          ...(!isDir ? [{ icon: 'search', label: 'View', on: () => viewFile(file) }] : []),
        ]},
        { items: [
          { icon: 'pen', label: 'Rename', on: () => renameFile(file) },
          { icon: 'copy', label: 'Copy to…', on: () => copyFile(file) },
          { icon: 'arrow-right', label: 'Move to…', on: () => moveFile(file) },
        ]},
        { items: [
          { icon: 'lock', label: 'Permissions…', on: () => editPermissions(file) },
        ]},
        { items: [{ icon: 'trash', label: 'Delete', danger: true, on: async () => deleteFile(file) }] },
      ],
    })
  }
</script>

<div
  bind:this={dropZone}
  class="group relative border-2 border-transparent"
  style="--wails-drop-target: drop"
>
  <div class="absolute inset-0 hidden flex-col items-center justify-center z-50 bg-black/80 text-brand group-[.wails-drop-target-active]:flex">
    <Icon name="upload" size={32} />
    <h2 class="mt-5 text-white text-xl font-semibold">Drop files to upload to:</h2>
    <span class="font-mono">{store.state.path}</span>
  </div>
  <div class="flex items-center gap-2 px-3 py-2 border-b border-line bg-chrome text-sm">
    <Button color="dark" size="sm" onclick={goUp} disabled={isAtRoot(currentPath)}>&uarr; Up</Button>
    <div class="flex-1 min-w-0">
      <TextInput size="sm" bind:value={currentPath} onkeydown={(e) => {if(e.key==='Enter') store.refresh(currentPath)}} class="font-mono" />
    </div>
    <Button color="dark" size="sm" onclick={() => store.refresh(currentPath)}>Go</Button>
    <div class="w-40">
      <TextInput size="sm" placeholder="Filter..." bind:value={filterText} class="font-mono" />
    </div>
    {#if !picker}
      <Button color="dark" size="sm" onclick={openGlobalHistory} title="View download history">History</Button>
      <Button color="dark" size="sm" onclick={newFolder}>New Folder</Button>
      <Button color="primary" size="sm" onclick={uploadFile} disabled={uploading}>
        {uploading ? 'Uploading...' : 'Upload'}
      </Button>
    {:else if picker === 'dir'}
      <Button color="primary" size="sm" onclick={() => emitPick(currentPath)} disabled={!currentPath}>
        Select this folder
      </Button>
    {:else if picker === 'file'}
      <span class="text-xs text-fg-muted self-center">Single-click a file to select it</span>
    {/if}
  </div>
  
  <div class="flex-1 min-h-0 overflow-auto">
    <DataTable
      data={filteredFiles}
      columns={tableColumns}
      keyField="_name"
      loading={store.state.loading}
      error={store.state.error}
      emptyState={{ title: 'No files found.' }}
      onRowDblClick={(item) => handleDoubleClick(item.rawFile)}
      onRowContextMenu={picker ? undefined : (item, e) => handleRightClick(e, item.rawFile)}
      onRowClick={picker ? (item) => handleRowClick(item.rawFile) : undefined}
    >
      {#snippet children(item, col)}
        {#if col.key === 'iconStr'}
          <span class:text-brand={item.isDir}>{item.iconStr}</span>
        {:else if col.key === '_sortSize'}
          <span class="font-mono">{item.sizeStr}</span>
        {:else if col.key === 'modTimeStr'}
          <span class="font-mono">{item.modTimeStr}</span>
        {:else}
          {item[col.key] ?? ''}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</div>

<FileViewerModal {viewerData} onclose={() => { viewerData = null }} />
<DownloadHistoryModal
  isOpen={historyModalState.isOpen}
  remotePath={historyModalState.remotePath}
  {sessionID}
  onclose={() => { historyModalState = { ...historyModalState, isOpen: false } }}
/>
