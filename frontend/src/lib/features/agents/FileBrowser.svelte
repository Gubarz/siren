<script>
  import Icon from '$components/ui/Icon.svelte'
  import FileViewerModal from './modals/FileViewerModal.svelte'
  import DownloadHistoryModal from './modals/DownloadHistoryModal.svelte'
  import { UploadFiles } from '../../api/agents.js';
  import { onFileDrop } from '../../api/runtime.js';
  import { createFileBrowserActions } from './fileBrowserActions.js';
  import { contextMenu } from '$stores/ui/contextMenu.svelte.js';
  import { dialog } from '$stores/ui/dialog.svelte.js';
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js';
  import { tagsModal } from '$stores/ui/tagsModal.svelte.js';
  import { errorMessage } from '../../utils/errors.js';
  import DataTable from '$components/patterns/DataTable.svelte';
  import Button from '$components/ui/Button.svelte';
  import Checkbox from '$components/ui/Checkbox.svelte';
  import EntityTagBadges from '$components/ui/EntityTagBadges.svelte';
  import TextInput from '$components/ui/TextInput.svelte';
  import { entityColors } from '$stores/resources/entityColors.svelte.js';
  import { useResource } from '$stores/lib/createResource.svelte.js';
  import { useFileBrowser } from '$stores/perAgent/fileBrowser.svelte.js';
  import { entityColorStyle } from '../../utils/entityTags.js';

  useResource(entityColors)

  let {
    sessionID = "",
    picker = false,
    onpick,
    startPath = '',
    staticData = null,
  } = $props();

  let currentPath = $state('');
  let dropZone = $state();
  let uploading = $state(false);
  let filterText = $state('');
  let viewerData = $state(null); // { filename, data, isBinary } or null
  let historyModalState = $state({ isOpen: false, remotePath: '', sessionID: '' });
  let selected = $state(new Set());

  function openFileHistory(file) {
    const name = file.Name || file.name;
    historyModalState = { isOpen: true, remotePath: joinPath(name), sessionID };
  }

  function openGlobalHistory() {
    historyModalState = { isOpen: true, remotePath: '', sessionID };
  }

  function toggleSelection(name) {
    const next = new Set(selected);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    selected = next;
  }

  function selectAll() {
    selected = new Set(filteredFiles.map((f) => f._name));
  }

  function clearSelection() {
    selected = new Set();
  }

  function getSelectedFileObjects() {
    return filteredFiles.filter((f) => selected.has(f._name)).map((f) => f.rawFile);
  }

  async function downloadSelectedTar() {
    const fileObjs = getSelectedFileObjects();
    if (fileObjs.length === 0) return;
    await actions.downloadMultipleTar(fileObjs);
  }

  let store = $derived(!staticData ? useFileBrowser(sessionID) : null);

  $effect(() => {
    if (store) {
      store.acquire();
      return () => store.release();
    }
  });
  $effect(() => {
    if (!staticData) currentPath = store.state.path;
    else currentPath = staticData.path || '';
    selected = new Set();
  });

  // If picker was opened with a suggested starting path (e.g. re-opening a
  // previously-picked value), navigate there once on mount.
  let seeded = false;
  $effect(() => {
    if (seeded || !picker || !startPath || staticData) return;
    seeded = true;
    store.refresh(startPath);
  });

  function emitPick(path) {
    onpick?.({ path });
  }

  // Context menu state

  let tableColumns = [
    { key: "_checkbox", label: "", width: 32, sortable: false },
    { key: "iconStr", label: "Name", width: 300 },
    { key: "_tags", label: "Tags", width: 108, sortable: false },
    { key: "_sortSize", label: "Size", width: 100 },
    { key: "typeStr", label: "Type", width: 100 },
    { key: "modTimeStr", label: "Last Modified", width: 250 }
  ];

  let normalizedFiles = $derived((staticData ? (staticData.files || []) : (store.state.files || [])).map(f => ({
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
    if (staticData) return;
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
    store: { refresh: (p) => staticData ? null : store.refresh(p), get: () => staticData ? { files: staticData.files || [], path: staticData.path || '' } : store.state },
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

  function openFileComments(file) {
    const name = file.Name || file.name || file._name;
    commentsModal.openComments('file', fileEntityID(file), name);
  }

  function openFileTags(file) {
    const name = file.Name || file.name || file._name;
    tagsModal.openTags('file', fileEntityID(file), name);
  }

  function openSelectedFileTags(file) {
    const name = fileName(file);
    if (selected.has(name) && selected.size > 1) {
      const targets = getSelectedFileObjects().map((target) => ({
        type: 'file',
        id: fileEntityID(target),
        label: fileName(target),
      }));
      tagsModal.openTagsForEntities(targets, `${targets.length} files`);
      return;
    }
    openFileTags(file);
  }

  function fileName(file) {
    return file.Name || file.name || file._name || '';
  }

  function fileEntityID(file) {
    const name = fileName(file);
    const fullPath = joinPath(name);
    return sessionID ? `${sessionID}:${fullPath}` : fullPath;
  }

  function handleRightClick(event, file) {
    const isDir = file.IsDir || file.isDir;
    const isBulk = selected.has(fileName(file)) && selected.size > 1;
    contextMenu.open({
      x: event.clientX, y: event.clientY,
      sections: [
        { items: [
          ...(isBulk
            ? [{ icon: 'download', label: `Tar & Download (${selected.size} items)`, on: () => downloadSelectedTar() }]
            : isDir
            ? [{ icon: 'download', label: 'Download (tar)', on: () => downloadDir(file) }]
            : [{ icon: 'download', label: 'Download', on: () => downloadFile(file) }]
          ),
          { icon: 'history', label: 'Download History', on: () => openFileHistory(file) },
          { icon: 'tag', label: isBulk ? `Tags / Color (${selected.size})…` : 'Tags / Color…', on: () => openSelectedFileTags(file) },
          { icon: 'message-square', label: 'Comments / Notes…', on: () => openFileComments(file) },
          ...(!isDir && !isBulk ? [{ icon: 'search', label: 'View', on: () => viewFile(file) }] : []),
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
    <span class="font-mono">{staticData ? (staticData.path || '') : store.state.path}</span>
  </div>
  <div class="flex items-center gap-2 px-3 py-2 border-b border-line bg-chrome text-sm">
    {#if staticData}
      <span class="text-xs text-fg-muted font-semibold">Snapshot of {staticData.path || ''} ({staticData.files?.length || 0} files)</span>
      <div class="flex-1 min-w-0">
        <TextInput size="sm" value={staticData.path || ''} class="font-mono" disabled />
      </div>
    {:else}
    <Button color="dark" size="sm" onclick={goUp} disabled={isAtRoot(currentPath)}>&uarr; Up</Button>
    <div class="flex-1 min-w-0">
      <TextInput size="sm" bind:value={currentPath} onkeydown={(e) => {if(e.key==='Enter') store.refresh(currentPath)}} class="font-mono" />
    </div>
    <Button color="dark" size="sm" onclick={() => store.refresh(currentPath)}>Go</Button>
    <div class="w-40">
      <TextInput size="sm" placeholder="Filter..." bind:value={filterText} class="font-mono" />
    </div>
    {#if !picker}
      {#if selected.size > 0}
        <Button color="primary" size="sm" icon="download" onclick={downloadSelectedTar}>
          Tar & Download ({selected.size})
        </Button>
        <Button color="dark" size="sm" onclick={clearSelection}>Deselect</Button>
      {:else}
        <Button color="dark" size="sm" onclick={selectAll} disabled={filteredFiles.length === 0}>Select All</Button>
      {/if}
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
    {/if}
  </div>
  
  <div class="flex-1 min-h-0 overflow-auto">
    <DataTable
      data={filteredFiles}
      columns={tableColumns}
      keyField="_name"
      selectable={picker ? 'none' : 'multiple'}
      bind:selected={selected}
      loading={staticData ? false : (store?.state?.loading ?? false)}
      error={staticData ? null : (store?.state?.error ?? null)}
      emptyState={{ title: 'No files found.' }}
      rowStyle={(item) => entityColorStyle(entityColors.data, 'file', fileEntityID(item.rawFile))}
      onRowDblClick={(item) => handleDoubleClick(item.rawFile)}
      onRowContextMenu={picker ? undefined : (item, e) => handleRightClick(e, item.rawFile)}
      onRowClick={picker ? (item) => handleRowClick(item.rawFile) : undefined}
    >
      {#snippet children(item, col)}
        {#if col.key === '_checkbox'}
          <Checkbox
            checked={selected.has(item._name)}
            onclick={(e) => {
              e.stopPropagation();
              toggleSelection(item._name);
            }}
          />
        {:else if col.key === 'iconStr'}
          <span class:text-brand={item.isDir}>{item.iconStr}</span>
        {:else if col.key === '_tags'}
          <EntityTagBadges entityType="file" entityID={fileEntityID(item.rawFile)} showEmpty />
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
