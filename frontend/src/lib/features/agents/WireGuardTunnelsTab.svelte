<script>
  import Button from '$components/ui/Button.svelte'
  import PanelBody from '$components/patterns/PanelBody.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { WGStartSocks, WGStopSocks, WGStartPortForward, WGStopPortForward } from '../../api/wireguard.js'
  import { errorMessage } from '../../utils/errors.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'
  import { useWGTunnels } from '$stores/perAgent/wgTunnels.svelte.js'

  let { sessionID = '' } = $props()

  let store = $derived(useWGTunnels(sessionID))

  $effect(() => {
    store.acquire()
    return () => store.release()
  })

  let localError = $state('')
  let error = $derived(localError || store.state.error || '')

  async function startSocks() {
    const port = await dialog.prompt('Local SOCKS5 port:', 'WG Start SOCKS', '1080')
    if (!port) return
    try {
      await WGStartSocks(sessionID, parseInt(port))
      await store.refresh()
    } catch (err) {
      localError = errorMessage(err, 'SOCKS failed: ')
    }
  }

  async function stopSocks(id) {
    try {
      await WGStopSocks(sessionID, id)
      await store.refresh()
    } catch (err) {
      localError = errorMessage(err, 'Stop failed: ')
    }
  }

  async function startForward() {
    const local = await dialog.prompt('Local port:', 'WG Start Portfwd', '8080')
    if (!local) return
    const remote = await dialog.prompt('Remote address (host:port):', 'Remote Target', '')
    if (!remote) return
    try {
      await WGStartPortForward(sessionID, parseInt(local), remote)
      await store.refresh()
    } catch (err) {
      localError = errorMessage(err, 'Portfwd failed: ')
    }
  }

  async function stopForward(id) {
    try {
      await WGStopPortForward(sessionID, id)
      await store.refresh()
    } catch (err) {
      localError = errorMessage(err, 'Stop failed: ')
    }
  }
</script>

<div class="flex flex-col h-full">
  <Toolbar class="justify-end gap-1">
    <Button color="primary" size="xs" icon="plug" onclick={startSocks}>SOCKS</Button>
    <Button color="primary" size="xs" icon="arrow-right" onclick={startForward}>Portfwd</Button>
    <Button color="dark" size="xs" onclick={() => store.refresh()} disabled={store.state.loading}>Refresh</Button>
  </Toolbar>

  <PanelBody error={error || null}>
    <div class="p-2 space-y-3">
      <section>
        <h3 class="text-xs font-semibold mb-1">SOCKS Servers</h3>
        <table class="w-full border-collapse text-xs">
          <thead>
            <tr class="border-b border-line bg-table-header text-left text-fg-muted">
              <th class="px-3 py-2 font-medium">ID</th>
              <th class="px-3 py-2 font-medium">Local Address</th>
              <th class="px-3 py-2 text-right font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each store.state.socksServers as s}
              <tr class="border-b border-line hover:bg-row-hover">
                <td class="px-3 py-2 font-mono">{s.ID ?? s.id}</td>
                <td class="px-3 py-2 font-mono">{s.LocalAddr ?? s.localAddr}</td>
                <td class="px-3 py-2 text-right">
                  <Button color="red" size="xs" onclick={() => stopSocks(s.ID ?? s.id)}>Stop</Button>
                </td>
              </tr>
            {:else}
              <tr><td colspan="3" class="px-3 py-2 text-center text-fg-muted">No SOCKS servers.</td></tr>
            {/each}
          </tbody>
        </table>
      </section>

      <section>
        <h3 class="text-xs font-semibold mb-1">Port Forwarders</h3>
        <table class="w-full border-collapse text-xs">
          <thead>
            <tr class="border-b border-line bg-table-header text-left text-fg-muted">
              <th class="px-3 py-2 font-medium">ID</th>
              <th class="px-3 py-2 font-medium">Local</th>
              <th class="px-3 py-2 font-medium">Remote</th>
              <th class="px-3 py-2 text-right font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each store.state.forwarders as f}
              <tr class="border-b border-line hover:bg-row-hover">
                <td class="px-3 py-2 font-mono">{f.ID ?? f.id}</td>
                <td class="px-3 py-2 font-mono">{f.LocalAddr ?? f.localAddr}</td>
                <td class="px-3 py-2 font-mono">{f.RemoteAddr ?? f.remoteAddr}</td>
                <td class="px-3 py-2 text-right">
                  <Button color="red" size="xs" onclick={() => stopForward(f.ID ?? f.id)}>Stop</Button>
                </td>
              </tr>
            {:else}
              <tr><td colspan="4" class="px-3 py-2 text-center text-fg-muted">No port forwarders.</td></tr>
            {/each}
          </tbody>
        </table>
      </section>
    </div>
  </PanelBody>
</div>
