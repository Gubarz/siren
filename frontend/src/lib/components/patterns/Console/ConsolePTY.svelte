<script>
  import { onMount } from 'svelte'
  import { acquireConsolePty } from './consolePtySession.js'

  let { sessionID = '', onshell = null } = $props()

  let hostEl

  onMount(() => {
    const pty = acquireConsolePty(sessionID, hostEl, onshell)

    const ro = new ResizeObserver(() => {
      pty.resize()
    })
    ro.observe(hostEl)
    pty.resize()

    return () => {
      ro.disconnect()
      pty.release()
    }
  })
</script>

<div bind:this={hostEl} class="h-full min-h-0 overflow-hidden"></div>
