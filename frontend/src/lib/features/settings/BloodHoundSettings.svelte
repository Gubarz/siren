<script>
  import { onMount } from 'svelte'
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import { toast } from '$stores/ui/toast.svelte.js'
  import { errorMessage } from '$utils/errors.js'
  import {
    getBloodHoundConfig,
    saveBloodHoundConfig,
    testBloodHoundConnection,
  } from '$api/bloodhound.js'

  let serverUrl = $state('')
  let tokenId = $state('')
  let tokenKey = $state('')
  let insecureTls = $state(false)
  let hasSavedKey = $state(false)
  let busy = $state(false)
  let error = $state('')

  onMount(async () => {
    try {
      const cfg = await getBloodHoundConfig()
      if (cfg) {
        serverUrl = cfg.serverUrl
        tokenId = cfg.tokenId
        hasSavedKey = Boolean(cfg.hasTokenKey)
        insecureTls = Boolean(cfg.insecureTls)
      }
    } catch (e) {
      error = e?.message || String(e)
    }
  })

  function currentConfig() {
    return {
      serverUrl: serverUrl.trim(),
      tokenId: tokenId.trim(),
      tokenKey: tokenKey.trim(),
      insecureTls,
    }
  }

  async function onSave() {
    error = ''
    busy = true
    try {
      await saveBloodHoundConfig(currentConfig())
      hasSavedKey = hasSavedKey || tokenKey.trim() !== ''
      tokenKey = ''
      toast.push({ variant: 'success', message: 'BloodHound settings saved' })
    } catch (e) {
      error = e?.message || String(e)
      toast.push({ variant: 'error', message: `Save failed: ${errorMessage(e)}` })
    } finally {
      busy = false
    }
  }

  async function onTest() {
    error = ''
    busy = true
    try {
      const attempt = currentConfig()
      if (!attempt.tokenKey && !hasSavedKey) {
        toast.push({ variant: 'warning', message: 'Enter a token key to test' })
        return
      }
      await testBloodHoundConnection(attempt)
      toast.push({ variant: 'success', message: 'BloodHound connection OK' })
    } catch (e) {
      error = e?.message || String(e)
      toast.push({ variant: 'error', message: `Test failed: ${errorMessage(e)}` })
    } finally {
      busy = false
    }
  }
</script>

<div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
  <h3 class="m-0 mb-1 text-fg text-base">BloodHound</h3>
  <p class="text-xs mt-1 mb-4 text-fg-muted">
    HMAC API-token access to a BloodHound CE server. The token key is stored locally and never displayed again.
  </p>

  {#if error}
    <div class="mb-3 p-2 rounded bg-red-900/20 border border-red-800/40 text-red-300 text-xs">{error}</div>
  {/if}

  <form class="grid gap-4" onsubmit={(e) => { e.preventDefault(); onSave(); }}>
    <div>
      <label class="block text-sm font-medium text-fg mb-1" for="bh-server-url">Server URL</label>
      <TextInput
        id="bh-server-url"
        type="url"
        placeholder="https://bloodhound.example.com"
        value={serverUrl}
        oninput={(e) => serverUrl = e.target.value}
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-fg mb-1" for="bh-token-id">API token ID</label>
      <TextInput
        id="bh-token-id"
        value={tokenId}
        oninput={(e) => tokenId = e.target.value}
      />
    </div>

    <div>
      <label class="block text-sm font-medium text-fg mb-1" for="bh-token-key">
        API token key{hasSavedKey ? ' (saved)' : ''}
      </label>
      <p class="text-xs text-fg-muted mb-1">Leave blank to keep the saved key.</p>
      <TextInput
        id="bh-token-key"
        type="password"
        placeholder={hasSavedKey ? 'Leave blank to keep saved key' : ''}
        value={tokenKey}
        oninput={(e) => tokenKey = e.target.value}
      />
    </div>

    <label class="flex items-center gap-2 text-sm">
      <Checkbox checked={insecureTls} onchange={() => insecureTls = !insecureTls} />
      Skip TLS verification (self-signed certs)
    </label>

    <div class="flex items-center justify-end gap-2 pt-2 border-t border-line">
      <Button color="alternative" size="sm" disabled={busy} onclick={onTest}>Test connection</Button>
      <Button color="primary" size="sm" type="submit" disabled={busy}>Save</Button>
    </div>
  </form>
</div>
