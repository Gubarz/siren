<script>
  import { onMount } from 'svelte'
  import { GrepFiles, Pwd } from '../../api/agents.js'
  import { errorMessage } from '../../utils/errors.js'
  import Button from '$components/ui/Button.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import ErrorState from '$components/ui/ErrorState.svelte'
  import EmptyState from '$components/ui/EmptyState.svelte'
  import LoadingState from '$components/ui/LoadingState.svelte'

  let { sessionID = '' } = $props()

  let pattern = $state('')
  let path = $state('.')
  let recursive = $state(true)
  let beforeLines = $state(0)
  let afterLines = $state(0)
  let results = $state(null)
  let loading = $state(false)
  let cancelled = $state(false)
  let error = $state('')

  onMount(async () => {
    try {
      const cwd = await Pwd(sessionID)
      if (cwd) path = cwd
    } catch {}
  })

  async function search() {
    if (!pattern.trim() || loading) return
    loading = true
    cancelled = false
    error = ''
    results = null
    try {
      const text = await GrepFiles(sessionID, pattern.trim(), path, recursive, beforeLines, afterLines)
      if (text) results = { text }
      else results = { text: '' }
    } catch (err) {
      if (!cancelled) error = errorMessage(err, 'Grep failed')
    } finally {
      if (!cancelled) loading = false
    }
  }

  function stop() {
    cancelled = true
    loading = false
  }

</script>

<div class="flex flex-col h-full">
  <div class="tab-header">
    <div class="w-50">
      <TextInput size="sm" placeholder="Path..." bind:value={path} class="font-mono" />
    </div>
    <div class="flex-1 min-w-38">
      <TextInput size="sm" placeholder="Search pattern..." bind:value={pattern} onkeydown={(e) => { if (e.key === 'Enter') search() }} class="font-mono" />
    </div>
    {#if loading}
      <Button color="red" size="xs" onclick={stop}>Stop</Button>
    {:else}
      <Button size="xs" onclick={search}>Search</Button>
    {/if}
    <Checkbox bind:checked={recursive} label="Recursive" />
    <label class="flex items-center gap-1 text-xs whitespace-nowrap text-fg-muted">
      <span class="whitespace-nowrap">Before:</span>
      <div class="w-15">
        <TextInput type="number" size="sm" bind:value={beforeLines} min="0" max="20" />
      </div>
    </label>
    <label class="flex items-center gap-1 text-xs whitespace-nowrap text-fg-muted">
      <span class="whitespace-nowrap">After:</span>
      <div class="w-15">
        <TextInput type="number" size="sm" bind:value={afterLines} min="0" max="20" />
      </div>
    </label>
  </div>

  <div class="flex-1 overflow-y-auto p-2 flex flex-col">
    {#if error}
      <ErrorState {error} title="Search failed" class="m-2" />
    {/if}

    {#if loading}
      <LoadingState description="Searching remote files..." />
    {:else}
      {#if results}
        {#if results.text.trim()}
          <pre class="p-2 font-mono text-xs leading-6 whitespace-pre-wrap break-all m-0 text-fg overflow-y-auto flex-1">{results.text}</pre>
        {:else}
          <EmptyState icon="search" title="No matches found" description="The search pattern did not match any content in the specified path." />
        {/if}
      {:else}
        <EmptyState icon="search" title="Start a search" description="Enter a pattern and path, and click Search." />
      {/if}
    {/if}
  </div>
</div>
