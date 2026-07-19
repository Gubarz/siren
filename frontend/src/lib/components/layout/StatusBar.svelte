<script>
  import ServerProfileMenu from '../../features/connection/ServerProfileMenu.svelte'
  import { connection } from '$stores/connection.svelte.js'

  let { serverVersion = '', sessionCount = 0, beaconCount = 0 } = $props();
  let menuOpen = $state(false)
</script>

<div class="flex items-center bg-chrome-header px-4 py-1 text-xs text-brand border-t border-black">
  <button
    type="button"
    class="flex items-center gap-1 bg-transparent border-0 cursor-pointer text-brand hover:text-brand hover:underline p-0"
    title="Manage teamserver profiles"
    onclick={() => menuOpen = true}
  >
    <span class="text-success-500">●</span>
    <span>Connected</span>
    {#if connection.profile}
      <span class="text-fg-muted">·</span>
      <span class="text-fg">{connection.profile}</span>
    {/if}
  </button>
  {#if serverVersion}<span class="mx-2">·</span>Server v{serverVersion}{/if}
  <span class="mx-2">·</span>{sessionCount} session{sessionCount === 1 ? '' : 's'}
  <span class="mx-2">·</span>{beaconCount} beacon{beaconCount === 1 ? '' : 's'}
</div>

<ServerProfileMenu bind:open={menuOpen} />
