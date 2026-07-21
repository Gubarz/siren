<script>
  import { onMount } from 'svelte'
  import { AnsiUp } from 'ansi_up'
  import { RunSessionCommand } from '../../api/console.js'
  import { errorMessage } from '../../utils/errors.js'
  import Button from '$components/ui/Button.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import EmptyState from '$components/ui/EmptyState.svelte'
  import ErrorState from '$components/ui/ErrorState.svelte'
  import LoadingState from '$components/ui/LoadingState.svelte'

  let { sessionID = '', command = 'ifconfig' } = $props()

  let output = $state('')
  let loading = $state(false)
  let error = $state('')
  let showAllInterfaces = $state(false)
  let includeUdp = $state(false)
  let listeningOnly = $state(false)
  let ip4Only = $state(false)
  let ip6Only = $state(false)

  const ansiUp = new AnsiUp()
  ansiUp.use_classes = false

  const isIfconfig = $derived(command === 'ifconfig')
  const title = $derived(isIfconfig ? 'Ifconfig' : 'Netstat')
  const renderedOutput = $derived(ansiUp.ansi_to_html(output || ''))

  const commandLine = $derived.by(() => {
    const parts = [command]
    if (isIfconfig) {
      if (showAllInterfaces) parts.push('--all')
      return parts.join(' ')
    }
    if (includeUdp) parts.push('--udp')
    if (listeningOnly) parts.push('--listen')
    if (ip4Only && !ip6Only) parts.push('--ip4')
    if (ip6Only && !ip4Only) parts.push('--ip6')
    return parts.join(' ')
  })

  onMount(() => refresh())

  async function refresh() {
    if (!sessionID || loading) return
    loading = true
    error = ''
    try {
      output = await RunSessionCommand(sessionID, commandLine)
    } catch (err) {
      error = errorMessage(err, `${title} failed`)
    } finally {
      loading = false
    }
  }

  function handleEnter(event) {
    if (event.key === 'Enter') refresh()
  }
</script>

<div class="flex h-full min-h-0 flex-col bg-canvas">
  <div class="tab-header">
    <code class="min-w-0 max-w-full overflow-hidden text-ellipsis whitespace-nowrap rounded border border-line bg-chrome px-2 py-1 text-xs text-fg-muted">
      {commandLine}
    </code>
    <div class="ml-auto flex shrink-0 items-center gap-2">
      {#if isIfconfig}
        <Checkbox bind:checked={showAllInterfaces} label="All" onchange={refresh} onkeydown={handleEnter} />
      {:else}
        <Checkbox bind:checked={includeUdp} label="UDP" onchange={refresh} onkeydown={handleEnter} />
        <Checkbox bind:checked={listeningOnly} label="Listen" onchange={refresh} onkeydown={handleEnter} />
        <Checkbox bind:checked={ip4Only} label="IPv4" disabled={ip6Only} onchange={refresh} onkeydown={handleEnter} />
        <Checkbox bind:checked={ip6Only} label="IPv6" disabled={ip4Only} onchange={refresh} onkeydown={handleEnter} />
      {/if}
      <Button color="primary" size="xs" icon="refresh" loading={loading} onclick={refresh}>Refresh</Button>
    </div>
  </div>

  <div class="min-h-0 flex-1 overflow-auto p-2">
    {#if error}
      <ErrorState {error} title={`${title} failed`} class="m-2" />
    {:else if loading && !output}
      <LoadingState description={`Loading ${command}...`} />
    {:else if output.trim()}
      <pre class="m-0 min-h-full whitespace-pre-wrap break-words rounded border border-line bg-panel p-3 font-mono text-xs leading-5 text-fg">{@html renderedOutput}</pre>
    {:else}
      <EmptyState icon={isIfconfig ? 'network-wired' : 'list'} title={`No ${command} output`} />
    {/if}
  </div>
</div>
