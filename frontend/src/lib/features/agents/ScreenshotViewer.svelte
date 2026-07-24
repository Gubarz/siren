<script>
  import InlineImage from '$components/ui/InlineImage.svelte';
  import { TakeScreenshot } from '../../api/agents.js';
  import { errorMessage } from '../../utils/errors.js';
  import Button from '$components/ui/Button.svelte';

  let { sessionID = "", staticBase64 = "" } = $props();

  const shotBySession = new Map();

  let screenshotBase64 = $state("");
  let loading = $state(false);
  let error = $state("");

  let lastSession = null;
  $effect(() => {
    if (sessionID !== lastSession) {
      lastSession = sessionID;
      screenshotBase64 = staticBase64 || shotBySession.get(sessionID) || "";
      error = "";
    }
  });

  $effect(() => {
    if (staticBase64) {
      screenshotBase64 = staticBase64;
      shotBySession.set(sessionID, staticBase64);
    }
  });

  async function takeScreenshot() {
    loading = true;
    error = "";
    try {
      screenshotBase64 = await TakeScreenshot(sessionID);
      shotBySession.set(sessionID, screenshotBase64);
    } catch (err) {
      error = errorMessage(err);
    } finally {
      loading = false;
    }
  }
</script>

<div class="tab-wrapper">
  <div class="tab-header justify-between text-sm">
    <span>Screenshot</span>
    <Button color="dark" size="sm" onclick={takeScreenshot}>Retake</Button>
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
