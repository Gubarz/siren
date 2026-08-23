<script>
  import Button from '$components/ui/Button.svelte';
  import Checkbox from '$components/ui/Checkbox.svelte';
  import TextInput from '$components/ui/TextInput.svelte';
  import Modal from '$components/patterns/Modal.svelte';
  import { COLLECTION_METHODS, buildCollectionOptions } from './collectionOptions.js';
  import { startBloodHoundCollection } from '$api/bloodhound.js';
  import { errorMessage } from '$utils/errors.js';
  import { toast } from '$stores/ui/toast.svelte.js';

  let { agent = null, onclose = () => {} } = $props();
  let open = $state(false);

  let methods = $state(['Default']);
  let flags = $state('');
  let domain = $state('');
  let timeoutMinutes = $state(15);
  let autoIngest = $state(true);
  let archiveLoot = $state(true);
  let submitting = $state(false);
  let error = $state('');

  function toggleMethod(method) {
    methods = methods.includes(method)
      ? methods.filter((m) => m !== method)
      : [...methods, method];
  }

  async function onStart() {
    error = '';
    submitting = true;
    try {
      const opts = buildCollectionOptions({
        methods, flags, domain, timeoutMinutes, ingest: autoIngest, loot: archiveLoot,
      });
      const id = await startBloodHoundCollection(
        agent.ID, agent._kind === 'beacon' ? 'beacon' : 'session', agent.OS ?? 'windows', opts,
      );
      toast.push({ variant: 'success', message: `Collection started (${id.slice(0, 8)})` });
      onclose();
    } catch (e) {
      error = errorMessage(e);
    } finally {
      submitting = false;
    }
  }
</script>

<Modal bind:open title={`Collect BloodHound data — ${agent?.Hostname || agent?.ID || ''}`} {onclose}>
  {#if error}
    <div class="mb-3 p-2 rounded bg-red-900/20 border border-red-800/40 text-red-300 text-xs">{error}</div>
  {/if}

  <div class="mb-3">
    <p class="text-xs font-medium text-fg mb-1">Collection methods</p>
    <div class="flex flex-wrap gap-2">
      {#each COLLECTION_METHODS as method (method)}
        <label class="flex items-center gap-1 text-xs cursor-pointer">
          <Checkbox checked={methods.includes(method)} onchange={() => toggleMethod(method)} />
          {method}
        </label>
      {/each}
    </div>
  </div>

  <div class="mb-3">
    <label class="block text-xs font-medium text-fg mb-1" for="bh-flags">Extra flags (space-separated)</label>
    <TextInput id="bh-flags" value={flags} oninput={(e) => (flags = e.target.value)} placeholder="--Stealth" />
  </div>

  <div class="mb-3">
    <label class="block text-xs font-medium text-fg mb-1" for="bh-domain">Domain (optional)</label>
    <TextInput id="bh-domain" value={domain} oninput={(e) => (domain = e.target.value)} placeholder="corp.local" />
  </div>

  <div class="mb-3">
    <label class="block text-xs font-medium text-fg mb-1" for="bh-timeout">Timeout (minutes)</label>
    <TextInput id="bh-timeout" value={timeoutMinutes} oninput={(e) => (timeoutMinutes = Number(e.target.value))} />
  </div>

  <label class="flex items-center gap-2 text-xs mb-2">
    <Checkbox checked={autoIngest} onchange={() => (autoIngest = !autoIngest)} />
    Ingest into BloodHound after download
  </label>
  <label class="flex items-center gap-2 text-xs mb-4">
    <Checkbox checked={archiveLoot} onchange={() => (archiveLoot = !archiveLoot)} />
    Archive artifact to loot
  </label>

  <div class="flex justify-end gap-2 pt-2 border-t border-line">
    <Button color="alternative" size="sm" disabled={submitting} onclick={onclose}>Cancel</Button>
    <Button color="primary" size="sm" loading={submitting} onclick={onStart}>Start collection</Button>
  </div>
</Modal>
