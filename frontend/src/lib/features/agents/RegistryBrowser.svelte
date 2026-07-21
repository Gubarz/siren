<script>
  import Icon from '$components/ui/Icon.svelte'
  import Button from '$components/ui/Button.svelte'
  import {
    CreateRegistryKey,
    DeleteRegistryEntry,
    WriteRegistryValue,
  } from '../../api/agents.js';
  import { dialog } from '$stores/ui/dialog.svelte.js';
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js';
  import { errorMessage } from '../../utils/errors.js';
  import { contextMenu } from '$stores/ui/contextMenu.svelte.js';
  import DataTable from '$components/patterns/DataTable.svelte';
  import SplitPane from '$components/patterns/SplitPane.svelte';
  import { useRegistry } from '$stores/perAgent/registry.svelte.js';

  let { sessionID = "" } = $props();

  const hives = ["HKLM", "HKCU", "HKU", "HKCR", "HKCC"];

  let currentHive = $state('HKLM')
  let currentPath = $state('')

  let store = $derived(useRegistry(sessionID))

  $effect(() => {
    store.acquire()
    return () => store.release()
  })
  
  $effect(() => {
    currentPath = store.state.path
  });
  $effect(() => {
    currentHive = store.state.hive || 'HKLM'
  });

  let tableColumns = [
    { key: "iconStr", label: "Name", width: 300 },
    { key: "typeStr", label: "Type", width: 150 },
    { key: "DataStr", label: "Data", width: 400 }
  ];

  let normalizedData = $derived([
    ...(store.state.keys || []).map(k => ({ _key: `k:${k}`, iconStr: `📁 ${k}`, typeStr: 'Key', DataStr: '', rawName: k, isKey: true })),
    ...(store.state.values || []).map(value => ({
      _key: `v:${value.name}`,
      iconStr: `📄 ${value.name}`,
      typeStr: value.type,
      DataStr: value.value,
      rawName: value.name,
      isKey: false,
      rawValue: value,
    }))
  ]);

  function handleHiveClick(hive) {
    currentHive = hive;
    currentPath = "";
    store.refresh(currentHive, currentPath);
  }

  function handleKeyDoubleClick(keyName) {
    let sep = "\\";
    const nextPath = currentPath === "" ? keyName : currentPath + sep + keyName;
    store.refresh(currentHive, nextPath);
  }

  function goUp() {
    if (currentPath === "") return;
    let sep = "\\";
    let parts = currentPath.split(sep);
    parts.pop();
    store.refresh(currentHive, parts.join(sep));
  }

  async function createKey() {
    const name = await dialog.prompt('Key name:', 'New Registry Key');
    if (!name) return;
    try {
      await CreateRegistryKey(sessionID, currentHive, currentPath, name);
      store.refresh(currentHive, store.state.path);
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Create key failed: '), 'Registry Error');
    }
  }

  async function writeValue(existing = null) {
    const name = existing?.rawName || await dialog.prompt('Value name:', 'Registry Value');
    if (!name) return;
    const type = await dialog.prompt(
      'Value type (string, dword, qword, binary):',
      'Registry Value Type',
      'string',
    );
    if (!type) return;
    const value = await dialog.prompt(
      type.toLowerCase() === 'binary' ? 'Hexadecimal value:' : 'Value:',
      existing ? 'Edit Registry Value' : 'New Registry Value',
      existing?.rawValue?.value || '',
    );
    if (value === null) return;
    try {
      await WriteRegistryValue(sessionID, currentHive, currentPath, name, type, value);
      store.refresh(currentHive, store.state.path);
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Write value failed: '), 'Registry Error');
    }
  }

  async function deleteEntry(row) {
    const kind = row.isKey ? 'key' : 'value';
    if (!(await dialog.confirm(`Delete registry ${kind} "${row.rawName}"?`, 'Confirm Delete'))) return;
    try {
      await DeleteRegistryEntry(sessionID, currentHive, currentPath, row.rawName);
      store.refresh(currentHive, store.state.path);
    } catch (err) {
      await dialog.alert(errorMessage(err, `Delete ${kind} failed: `), 'Registry Error');
    }
  }

  function showMenu(event, row = null) {
    contextMenu.open({
      x: event.clientX, y: event.clientY,
      sections: [
        { items: row
          ? row.isKey
            ? [
                { icon: 'folder-open', label: 'Open', on: () => handleKeyDoubleClick(row.rawName) },
                { icon: 'message-square', label: 'Comments / Notes…', on: () => commentsModal.openComments('registry', `${currentHive}\\${store.state.path}\\${row.rawName}`, `${currentHive}\\${row.rawName}`) },
              ]
            : [
                { icon: 'pen', label: 'Edit Value', on: () => writeValue(row) },
                { icon: 'message-square', label: 'Comments / Notes…', on: () => commentsModal.openComments('registry', `${currentHive}\\${store.state.path}\\${row.rawName}`, `${currentHive}\\${row.rawName}`) },
              ]
          : [
              { icon: 'folder-plus', label: 'New Key', on: () => createKey() },
              { icon: 'plus', label: 'New Value', on: () => writeValue() },
            ],
        },
        ...(row ? [{ items: [{ icon: 'trash', label: row.isKey ? 'Delete Key' : 'Delete Value', danger: true, on: () => deleteEntry(row) }] }] : []),
      ],
    })
  }
</script>

<SplitPane size={25}>
  {#snippet left()}
    <div class="flex h-full flex-col bg-panel border-r border-line">
      <div class="flex items-center px-3 py-2 border-b border-line bg-chrome text-sm">
        <span><strong>Registry Hives</strong></span>
      </div>
      <div class="flex-1 min-h-0 overflow-auto">
        <ul class="list-none p-0 m-0">
          {#each hives as hive}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
            <li
              class={`px-5 py-3 cursor-pointer border-b border-line transition-colors hover:bg-row-hover border-l-4 ${hive === currentHive ? 'bg-brand text-canvas border-l-canvas' : 'border-l-transparent'}`}
              onclick={() => handleHiveClick(hive)}
            >
              <Icon name="database" class="mr-2" /> {hive}
            </li>
          {/each}
        </ul>
      </div>
    </div>
  {/snippet}

  {#snippet right()}
    <div class="flex h-full flex-col min-w-0">
      <div class="flex items-center gap-2 px-3 py-2 border-b border-line bg-chrome text-sm">
        <Button size="sm" disabled={currentPath === ""} onclick={goUp}>↑ Up</Button>
        <div class="flex-1 bg-canvas border border-line text-fg px-2 py-1 rounded overflow-hidden text-ellipsis whitespace-nowrap font-mono">
          {currentHive}\{currentPath}
        </div>
        <Button size="sm" onclick={() => store.refresh(currentHive, currentPath)}>Refresh</Button>
        <Button size="sm" onclick={createKey}>New Key</Button>
        <Button size="sm" color="primary" onclick={() => writeValue()}>New Value</Button>
      </div>

      <div class="flex-1 min-h-0 overflow-auto" role="region" oncontextmenu={(event) => { event.preventDefault(); showMenu(event); }}>
        <DataTable
          data={normalizedData}
          columns={tableColumns}
          keyField="_key"
          loading={store.state.loading}
          error={store.state.error}
          emptyState={{ title: 'No keys or values found.' }}
          onRowDblClick={(item) => item.isKey ? handleKeyDoubleClick(item.rawName) : writeValue(item)}
          onRowContextMenu={(item, e) => { e.stopPropagation(); showMenu(e, item) }}
        >
          {#snippet children(item, col)}
            {#if col.key === 'iconStr'}
              <span class:text-brand={item.isKey}>{item.iconStr}</span>
            {:else if col.key === 'DataStr'}
              <span class="font-mono">{item.DataStr}</span>
            {:else}
              {item[col.key] ?? ''}
            {/if}
          {/snippet}
        </DataTable>
      </div>
    </div>
  {/snippet}
</SplitPane>
