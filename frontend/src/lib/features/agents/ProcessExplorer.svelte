<script>
  import { KillProcess } from "../../api/agents.js";
  import { dialog } from "../../stores/ui/dialog.svelte.js";
  import { errorMessage } from "../../utils/errors.js";
  import { contextMenu } from "$stores/ui/contextMenu.svelte.js";
  import { commandModal } from "$stores/ui/commandModal.svelte.js";
  import DataTable from "$components/patterns/DataTable.svelte";
  import Button from "$components/ui/Button.svelte";
  import EntityTagBadges from "$components/ui/EntityTagBadges.svelte";
  import { entityColors } from "$stores/resources/entityColors.svelte.js";
  import { useResource } from "$stores/lib/createResource.svelte.js";
  import { useProcessList } from "$stores/perAgent/processList.svelte.js";
  import { buildProcessTree, buildProcessContextSections } from "./processExplorerHelpers.js";
  import { entityColorStyle } from "../../utils/entityTags.js";

  let { sessionID = "", picker = false, onpick, staticData = null } = $props();

  useResource(entityColors)

  let store = $derived(!staticData ? useProcessList(sessionID) : null);

  $effect(() => {
    if (store) {
      store.acquire();
      return () => store.release();
    }
  });
  let isTreeView = $derived(store?.state?.isTreeView ?? false);
  let isFullView = $derived(store?.state?.isFullView ?? false);
  let isSnapshotTreeView = $state(false);
  let effectiveTreeView = $derived(staticData ? isSnapshotTreeView : isTreeView);

  const baseColumns = [
    { key: "PidStr", label: "PID", width: 120 },
    { key: "PpidStr", label: "PPID", width: 80 },
    { key: "ExecutableStr", label: "Executable", width: 250 },
    { key: "_tags", label: "Tags", width: 108, sortable: false },
  ];
  const fullColumns = [
    ...baseColumns.map((c) => ({ ...c })),
    { key: "OwnerStr", label: "Owner", width: 170 },
    { key: "ArchStr", label: "Arch", width: 80 },
    { key: "SessionStr", label: "Session", width: 80 },
  ];
  // Full View reveals deeper (and noisier) process detail.
  let tableColumns = $derived(isFullView ? fullColumns : baseColumns);

  let normalizedProcesses = $derived(
    staticData
      ? (staticData || []).map((p) => ({
          ...p,
          PidStr: String(p.Pid ?? p.pid ?? 0),
          PpidStr: String(p.Ppid ?? p.ppid ?? 0),
          ExecutableStr: p.Executable ?? p.executable ?? "",
          OwnerStr: p.Owner ?? p.owner ?? "",
          ArchStr: p.Architecture ?? p.architecture ?? "",
          SessionStr: String(p.SessionID ?? p.sessionID ?? ""),
        }))
      : (store.state.processes || []).map((p) => ({
          ...p,
          PidStr: String(p.Pid ?? p.pid ?? 0),
          PpidStr: String(p.Ppid ?? p.ppid ?? 0),
          ExecutableStr: p.Executable ?? p.executable ?? "",
          OwnerStr: p.Owner ?? p.owner ?? "",
          ArchStr: p.Architecture ?? p.architecture ?? "",
          SessionStr: String(p.SessionID ?? p.sessionID ?? ""),
        })),
  );

  let displayProcesses = $derived(
    effectiveTreeView ? buildProcessTree(normalizedProcesses) : normalizedProcesses,
  );

  // Full View surfaces owner/arch/session — deeper enumeration that isn't
  // opsec-safe, so confirm before enabling. Turning it back off is instant.
  async function toggleFullView() {
    if (isFullView) {
      store.setFullView(false);
      return;
    }
    const sid = sessionID; // capture: the user may switch agents during the dialog
    const ok = await dialog.confirm(
      "Full View performs deeper process enumeration (owner, architecture, session) " +
        "on this agent. This is NOT opsec-safe and may trigger EDR. Continue?",
      "Opsec Warning",
    );
    if (!ok) return;
    if (sid !== sessionID) return; // switched away
    store.setFullView(true);
  }

  async function killProcess(pid) {
    if (!store) return;
    if (
      !(await dialog.confirm(
        `Are you sure you want to kill PID ${pid}?`,
        "Confirm Kill",
      ))
    )
      return;
    try {
      await KillProcess(sessionID, pid);
      store.refresh(isFullView);
    } catch (err) {
      await dialog.alert(
        errorMessage(err, "Failed to kill process: "),
        "Kill Error",
      );
    }
  }

  function handleRowClick(item) {
    if (picker) {
      onpick?.({
        pid: item.Pid ?? item.pid ?? 0,
        name: item.Executable ?? item.executable ?? "",
      });
    }
  }

  function handleRightClick(event, proc) {
    if (picker) return;
    const pid = proc.Pid || proc.pid || 0;
    const procName = proc.Executable || proc.executable || proc.ExecutableStr || "";
    contextMenu.open({
      x: event.clientX, y: event.clientY,
      sections: buildProcessContextSections({
        pid, procName, commandModal,
        killProcess: staticData ? () => dialog.alert('Process kill unavailable for snapshot data.', 'Snapshot') : killProcess,
      }),
    });
  }
</script>

<div class="rounded-sm">
  <div class="tab-header flex justify-between gap-2">
    {#if staticData}
      <span class="text-xs text-fg-muted font-semibold">Beacon Task Snapshot ({staticData.length} processes)</span>
    {/if}
    <div class="flex gap-2 {staticData ? '' : 'ml-auto'}">
      {#if !staticData}
        <Button
          size="xs"
          color={isFullView ? "secondary" : "danger"}
          title="Reveals owner/arch/session via deeper enumeration (not opsec-safe)"
          onclick={toggleFullView}
        >
          {isFullView ? "Basic View" : "Full View"}
        </Button>
      {/if}
      {#if staticData}
        <Button size="xs" color="dark" onclick={() => isSnapshotTreeView = !isSnapshotTreeView}>
          {isSnapshotTreeView ? "List View" : "Tree View"}
        </Button>
      {:else}
        <Button size="xs" color="dark" onclick={() => store.setTreeView(!isTreeView)}>
          {isTreeView ? "List View" : "Tree View"}
        </Button>
      {/if}
      {#if !staticData}
        <Button size="xs" color="dark" onclick={() => store.refresh()}>Refresh</Button>
      {/if}
    </div>
  </div>

  <div>
    <DataTable
      data={displayProcesses}
      columns={tableColumns}
      keyField="PidStr"
      loading={store?.state?.loading ?? false}
      error={store?.state?.error ?? null}
      emptyState={{ title: "No processes found." }}
      rowStyle={(item) => entityColorStyle(entityColors.data, 'process', item.PidStr)}
      onRowContextMenu={picker
        ? undefined
        : (item, e) => handleRightClick(e, item)}
      onRowClick={picker ? (item) => handleRowClick(item) : undefined}
    >
      {#snippet children(item, col)}
        {#if col.key === "PidStr"}
          {#if effectiveTreeView}
            <span
              class="font-mono"
              style="display: inline-block; margin-left: {(item._depth || 0) *
                20}px;"
            >
              {#if (item._depth || 0) > 0}<span
                  style="color: var(--color-fg-muted); margin-right: 5px;">&rdsh;</span
                >{/if}
              {item.PidStr}
            </span>
          {:else}
            <span class="font-mono">{item.PidStr}</span>
          {/if}
        {:else if col.key === "PpidStr" || col.key === "ArchStr" || col.key === "SessionStr"}
          <span class="font-mono">{item[col.key]}</span>
        {:else if col.key === "_tags"}
          <EntityTagBadges entityType="process" entityID={item.PidStr} showEmpty />
        {:else}
          {item[col.key] ?? ""}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</div>
