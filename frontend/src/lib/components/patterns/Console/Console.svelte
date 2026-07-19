<script>
  import ConsolePTY from './ConsolePTY.svelte'

  let { sessionID = '', onshell = null } = $props()
</script>

<div class="flex flex-col h-full min-h-0 w-full border border-line rounded overflow-hidden bg-panel">
  {#key sessionID}
    <ConsolePTY {sessionID} {onshell} />
  {/key}
</div>
