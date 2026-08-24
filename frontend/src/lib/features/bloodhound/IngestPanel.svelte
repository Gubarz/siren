<script>
  import { onMount } from 'svelte';
  import Button from '$components/ui/Button.svelte';
  import Badge from '$components/ui/Badge.svelte';
  import Panel from '$components/patterns/Panel.svelte';
  import { onFileDrop, OpenFileDialog } from '$api/runtime.js';
  import { getBloodHoundIngestJob, ingestBloodHoundLocalFile } from '$api/bloodhound.js';
  import { bloodhoundStore, subscribeBloodhound, refreshIngestJobs } from './bloodhound.svelte.js';
  import { errorMessage } from '$utils/errors.js';
  import { toast } from '$stores/ui/toast.svelte.js';

  let { onclose = () => {} } = $props();

  let expandedID = $state(null);
  let expandedFiles = $state([]);
  let loadingFiles = $state(false);
  let uploading = $state(false);
  let draggedOver = $state(false);
  let dropZone;

  onMount(() => {
    subscribeBloodhound();
    refreshIngestJobs();
  });

  $effect(() => onFileDrop((x, y, paths) => {
    const target = document.elementFromPoint(x, y);
    if (!dropZone?.contains(target)) return;
    draggedOver = false;
    const path = paths?.[0];
    if (path) upload(path);
  }));

  function statusVariant(status) {
    return {
      complete: 'success',
      partially_complete: 'warning',
      failed: 'danger',
      canceled: 'danger',
      timed_out: 'danger',
      ingesting: 'primary',
      analyzing: 'primary',
      running: 'primary',
      ready: 'default',
    }[status] || 'default';
  }

  async function upload(path) {
    uploading = true;
    try {
      const job = await ingestBloodHoundLocalFile(path);
      toast.push({ variant: 'success', message: `Uploaded to BloodHound (job ${job?.id ?? '?'})` });
      refreshIngestJobs();
    } catch (e) {
      toast.push({ variant: 'error', message: `Ingest failed: ${errorMessage(e)}` });
    } finally {
      uploading = false;
    }
  }

  function browse() {
    OpenFileDialog('Select BloodHound collection file')
      .then((path) => { if (path) upload(path); })
      .catch(() => {});
  }

  async function toggleExpand(id) {
    if (expandedID === id) {
      expandedID = null;
      expandedFiles = [];
      return;
    }
    expandedID = id;
    loadingFiles = true;
    try {
      const job = await getBloodHoundIngestJob(id);
      expandedFiles = job?.files ?? [];
    } catch (e) {
      toast.push({ variant: 'error', message: `Job detail failed: ${errorMessage(e)}` });
      expandedFiles = [];
    } finally {
      loadingFiles = false;
    }
  }
</script>

<Panel {onclose}>
  <div class="flex flex-col gap-4 p-4">
    <div class="flex items-center justify-between gap-3">
      <h3 class="m-0 text-fg text-base">BloodHound Ingest</h3>
      <div class="flex items-center gap-2">
        <Button size="sm" color="alternative" onclick={refreshIngestJobs}>Refresh</Button>
        <Button size="sm" color="primary" loading={uploading} onclick={browse}>Upload collection…</Button>
      </div>
    </div>

    <div
      bind:this={dropZone}
      role="region"
      aria-label="Collection file drop zone"
      data-file-drop-target
      class="flex items-center justify-center rounded-lg border border-dashed border-line px-4 py-6 {draggedOver ? 'border-brand' : ''}"
      ondragover={() => (draggedOver = true)}
      ondragleave={() => (draggedOver = false)}
    >
      <p class="text-xs text-fg-muted m-0">
        Drop a SharpHound/AzureHound .zip or .json here, or use Upload.
      </p>
    </div>

    {#if bloodhoundStore.ingestJobs.length === 0}
      <p class="text-xs text-fg-muted m-0">No ingest jobs. Run a collection or drop a file.</p>
    {:else}
      <div class="flex flex-col gap-2">
        {#each bloodhoundStore.ingestJobs as job (job.id)}
          <div class="bg-panel border border-panel-border rounded-lg px-4 py-3">
            <Button
              color="alternative"
              size="sm"
              fullWidth
              class="!justify-start !border-0 !bg-transparent !p-0 hover:!bg-transparent"
              onclick={() => toggleExpand(job.id)}
            >
              <span class="text-xs font-mono text-fg-muted">#{job.id}</span>
              <Badge variant={statusVariant(job.status)}>{job.status || 'unknown'}</Badge>
              <span class="flex-1 min-w-0 text-xs text-fg-muted truncate">{job.message || ''}</span>
              <span class="text-xs text-fg-muted shrink-0">
                {job.failedFiles ? `${job.failedFiles} failed / ` : ''}{job.totalFiles ?? 0} files
              </span>
            </Button>
            {#if expandedID === job.id}
              <div class="mt-2 border-t border-line pt-2">
                {#if loadingFiles}
                  <p class="text-xs text-fg-muted m-0">Loading file results…</p>
                {:else if expandedFiles.length === 0}
                  <p class="text-xs text-fg-muted m-0">No per-file results yet.</p>
                {:else}
                  {#each expandedFiles as file (file.name)}
                    <div class="flex items-center gap-2 py-1">
                      <span class="flex-1 min-w-0 text-xs truncate">{file.name}</span>
                      {#if file.errors?.length}
                        <span class="text-xs text-danger-500 truncate" title={file.errors.join('; ')}>
                          {file.errors.join('; ')}
                        </span>
                      {:else}
                        <Badge variant="success">ok</Badge>
                      {/if}
                    </div>
                  {/each}
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
</Panel>
