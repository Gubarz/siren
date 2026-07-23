<script>
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import {
    getEnvInfo,
    setDataDirOverride,
    clearDataDirOverride,
    setLogDirOverride,
    clearLogDirOverride,
  } from '$api/envvars.js'

  let info = $state(null)
  let error = $state('')
  let successMsg = $state('')
  let dataDirInput = $state('')
  let logDirInput = $state('')

  async function load() {
    try {
      info = await getEnvInfo()
      dataDirInput = info.effective_data_dir
      logDirInput = info.effective_log_dir
    } catch (e) {
      error = e?.message || String(e)
    }
  }
  load()

  async function saveDataDir() {
    error = ''
    successMsg = ''
    try {
      info = await setDataDirOverride(dataDirInput)
      successMsg = 'Data directory updated. Restart console sessions for effect.'
    } catch (e) {
      error = e?.message || String(e)
    }
  }

  async function clearDataDir() {
    error = ''
    successMsg = ''
    try {
      info = await clearDataDirOverride()
      dataDirInput = info.effective_data_dir
      successMsg = 'Data directory override cleared.'
    } catch (e) {
      error = e?.message || String(e)
    }
  }

  async function saveLogDir() {
    error = ''
    successMsg = ''
    try {
      info = await setLogDirOverride(logDirInput)
      successMsg = 'Log directory updated.'
    } catch (e) {
      error = e?.message || String(e)
    }
  }

  async function clearLogDir() {
    error = ''
    successMsg = ''
    try {
      info = await clearLogDirOverride()
      logDirInput = info.effective_log_dir
      successMsg = 'Log directory override cleared.'
    } catch (e) {
      error = e?.message || String(e)
    }
  }
</script>

<div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
  <h3 class="m-0 mb-1 text-fg text-base">Environment</h3>
  <p class="text-xs mt-1 mb-4 text-fg-muted">
    Sliver client directories. Overrides persist in <code class="font-mono text-xs px-1 py-px rounded bg-chrome border border-line">gui-settings.yaml</code>.
  </p>

  {#if error}
    <div class="mb-3 p-2 rounded bg-red-900/20 border border-red-800/40 text-red-300 text-xs">{error}</div>
  {/if}
  {#if successMsg}
    <div class="mb-3 p-2 rounded bg-green-900/20 border border-green-800/40 text-green-300 text-xs">{successMsg}</div>
  {/if}

  <div class="mb-4">
    <label class="block text-sm font-medium text-fg mb-1">Data Directory</label>
    <p class="text-xs text-fg-muted mb-1">Where tags, comments, automation rules, cases, and discovery cache are stored.</p>
    <div class="flex items-center gap-2">
      <TextInput value={dataDirInput} oninput={(e) => dataDirInput = e.target.value} class="flex-1" />
      <Button color="primary" size="sm" onclick={saveDataDir}>Save</Button>
      <Button color="alternative" size="sm" onclick={clearDataDir}>Reset</Button>
    </div>
  </div>

  <div class="mb-4">
    <label class="block text-sm font-medium text-fg mb-1">Log Directory</label>
    <p class="text-xs text-fg-muted mb-1">Where client and console logs are stored.</p>
    <div class="flex items-center gap-2">
      <TextInput value={logDirInput} oninput={(e) => logDirInput = e.target.value} class="flex-1" />
      <Button color="primary" size="sm" onclick={saveLogDir}>Save</Button>
      <Button color="alternative" size="sm" onclick={clearLogDir}>Reset</Button>
    </div>
  </div>

  {#if info}
    <div class="mt-4 pt-3 border-t border-line">
      <h4 class="text-sm font-medium text-fg mb-2">Effective Paths</h4>
      <div class="grid grid-cols-1 gap-1 text-xs font-mono">
        <div class="flex justify-between">
          <span class="text-fg-muted">Configs:</span>
          <span class="text-fg truncate ml-2">{info.config_dir}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-fg-muted">Root:</span>
          <span class="text-fg truncate ml-2">{info.root_dir}</span>
        </div>
      </div>
      <h4 class="text-sm font-medium text-fg mt-3 mb-1">Active Env Vars</h4>
      <div class="grid grid-cols-1 gap-1 text-xs font-mono">
        {#each info.active_vars as v}
          <div class="flex justify-between">
            <span class="text-fg-muted">{v.name}:</span>
            <span class="text-fg truncate ml-2">{v.set ? v.value : '(not set)'}</span>
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>
