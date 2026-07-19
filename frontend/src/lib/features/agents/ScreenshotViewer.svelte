<script>
  import { TakeScreenshot } from '../../api/agents.js';
  import { errorMessage } from '../../utils/errors.js';
  import Button from '$components/ui/Button.svelte';

  let { sessionID = "" } = $props();

  // Keep each session's last screenshot so switching tabs and coming back shows
  // it again instead of a blank panel.
  const shotBySession = new Map();

  let screenshotBase64 = $state("");
  let loading = $state(false);
  let error = $state("");

  // Restore this session's screenshot (if we have one) when switching to it.
  let lastSession = null;
  $effect(() => {
    if (sessionID !== lastSession) {
      lastSession = sessionID;
      screenshotBase64 = shotBySession.get(sessionID) || "";
      error = "";
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
  
  <div class="tab-content text-center p-5">
    {#if error}
      <div class="text-danger-500">{error}</div>
    {:else if screenshotBase64}
      <img src="data:image/png;base64,{screenshotBase64}" alt="Target Screenshot" class="border border-line shadow-lg" />
    {:else}
      <div class="mt-24">
        <Button color="primary" size="lg" onclick={takeScreenshot} disabled={loading}>Capture Screenshot</Button>
      </div>
    {/if}
  </div>
</div>
