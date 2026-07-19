<script>
  import { AnsiUp } from 'ansi_up';
  import { CancelBeaconTask, GetBeaconTaskOutput } from '../../api/agents.js';
  import { dialog } from '../../stores/ui/dialog.svelte.js';
  import { errorMessage } from '../../utils/errors.js';
  import { useBeaconTasks } from '$stores/perAgent/beaconTasks.svelte.js';
  import Button from '$components/ui/Button.svelte';

  let { beaconID = '' } = $props();

  let expandedTask = $state('');
  let expandedContent = $state('');
  let cancelingTask = $state('');
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
    expandedContent = 'Loading...';
    try {
      expandedContent = await GetBeaconTaskOutput(task.ID);
    } catch (cause) {
      expandedContent = errorMessage(cause, 'Failed to load task: ');
    }
  }

  async function cancelTask(event, task) {
    event.stopPropagation();
    const confirmed = await dialog.confirm(`Cancel pending task ${shortID(task.ID)}?`);
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

  function shortID(id) {
    return String(id || '').split('-')[0];
  }

  function formatTime(value) {
    return value ? new Date(value * 1000).toLocaleString() : '-';
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
</script>

<div class="flex flex-col h-full bg-canvas">
  <div class="flex items-center justify-between gap-2 px-3 py-2 border-b border-line bg-chrome">
    <span>Beacon Tasks</span>
    <Button color="dark" size="sm" onclick={() => taskStore?.refresh()}>Refresh</Button>
  </div>

  <div class="flex-1 overflow-auto">
    {#if taskStore.state.error}
      <div class="p-4 text-danger-500">{taskStore.state.error}</div>
    {:else if taskStore.state.loading && taskStore.state.tasks.length === 0}
      <div class="p-4 text-fg-muted">Loading tasks...</div>
    {:else if taskStore.state.tasks.length === 0}
      <div class="p-4 text-fg-muted">No tasks have been queued for this beacon.</div>
    {:else}
      <table class="w-full border-collapse text-xs">
        <thead>
          <tr>
            <th>ID</th><th>Message Type</th><th>State</th><th>Created</th><th>Sent</th><th>Completed</th><th></th>
          </tr>
        </thead>
        <tbody>
          {#each taskStore.state.tasks as task (task.ID)}
            <tr class="cursor-pointer" class:selected={expandedTask === task.ID} onclick={() => toggleTask(task)}>
              <td class="font-mono">{shortID(task.ID)}</td>
              <td>{String(task.Description || '').replace(/Req$/, '') || '-'}</td>
              <td class={`font-semibold ${taskStateClass(task.State)}`}>{task.State || '-'}</td>
              <td>{formatTime(task.CreatedAt)}</td>
              <td>{formatTime(task.SentAt)}</td>
              <td>{formatTime(task.CompletedAt)}</td>
              <td class="w-px whitespace-nowrap">
                {#if taskState(task.State) === 'pending'}
                  <Button
                    color="red"
                    size="xs"
                    disabled={cancelingTask === task.ID}
                    onclick={(event) => cancelTask(event, task)}
                  >
                    {cancelingTask === task.ID ? 'Canceling...' : 'Cancel'}
                  </Button>
                {/if}
              </td>
            </tr>
            {#if expandedTask === task.ID}
              <tr>
                <td colspan="7" class="p-0 bg-chrome border-b-2 border-brand">
                  <pre class="max-h-105 m-0 p-4 overflow-auto whitespace-pre-wrap break-words text-fg text-sm font-mono">{@html ansiUp.ansi_to_html(expandedContent)}</pre>
                </td>
              </tr>
            {/if}
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>
