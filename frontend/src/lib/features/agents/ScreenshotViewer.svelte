<script>
  import InlineImage from '$components/ui/InlineImage.svelte';
  import { TakeScreenshot } from '../../api/agents.js';
  import { addLootFile } from '../../api/server.js';
  import { sessions } from '$stores/resources/sessions.svelte.js';
  import { beacons } from '$stores/resources/beacons.svelte.js';
  import { errorMessage } from '../../utils/errors.js';
  import Button from '$components/ui/Button.svelte';

  let { sessionID = "", staticBase64 = "" } = $props();

  const shotBySession = new Map();

  let screenshotBase64 = $state("");
  let loading = $state(false);
  let error = $state("");
  let saving = $state(false);
  let savedMessage = $state("");

  let lastSession = null;
  $effect(() => {
    if (sessionID !== lastSession) {
      lastSession = sessionID;
      screenshotBase64 = staticBase64 || shotBySession.get(sessionID) || "";
      error = "";
      savedMessage = "";
    }
  });

  $effect(() => {
    if (staticBase64) {
      screenshotBase64 = staticBase64;
      shotBySession.set(sessionID, staticBase64);
    }
  });

  let currentAgent = $derived(
    sessions.data.find((s) => s.ID === sessionID) ?? beacons.data.find((b) => b.ID === sessionID),
  );

  function defaultLootName() {
    const base = currentAgent?.Name || currentAgent?.Hostname || `session-${sessionID.slice(0, 8)}`
    const ts = new Date().toISOString().replace(/[^0-9]/g, '').slice(0, 14)
    return `${base}-${ts}.png`
  }

  async function takeScreenshot() {
    loading = true;
    error = "";
    savedMessage = "";
    try {
      screenshotBase64 = await TakeScreenshot(sessionID);
      shotBySession.set(sessionID, screenshotBase64);
    } catch (err) {
      error = errorMessage(err);
    } finally {
      loading = false;
    }
  }

  async function saveToLoot() {
    if (!screenshotBase64 || saving) return
    saving = true
    error = ""
    savedMessage = ""
    try {
      await addLootFile(defaultLootName(), screenshotBase64)
      savedMessage = `Saved to Loot as ${defaultLootName()}`
    } catch (err) {
      error = errorMessage(err, 'Save to Loot failed: ')
    } finally {
      saving = false
    }
  }
</script>

<div class="tab-wrapper">
  <div class="tab-header justify-between text-sm">
    <span>Screenshot</span>
    <div class="flex items-center gap-2">
      <span class="text-success-500">{savedMessage}</span>
      <Button color="dark" size="sm" icon="download" onclick={saveToLoot} disabled={!screenshotBase64 || saving || loading}>
        Save to Loot
      </Button>
      <Button color="dark" size="sm" onclick={takeScreenshot}>Retake</Button>
    </div>
  </div>
  
  <div class="tab-content p-2">
    {#if error}
      <div class="text-danger-500 p-4">{error}</div>
    {:else if screenshotBase64}
      <InlineImage src={"data:image/png;base64," + screenshotBase64} alt="Target Screenshot" maxHeight="100%" />
    {:else}
      <div class="flex items-center justify-center h-64">
        <Button color="primary" size="lg" onclick={takeScreenshot} disabled={loading}>Capture Screenshot</Button>
      </div>
    {/if}
  </div>
</div>
