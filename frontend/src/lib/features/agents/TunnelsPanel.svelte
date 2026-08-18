<script>
  import {
    StartSocks,
    StopSocks,
    StartPortfwd,
    StopPortfwd,
    StartRportfwd,
    StopRportfwd,
  } from '../../api/agents.js';
  import { errorMessage } from '../../utils/errors.js';
  import { formatBytes } from '../../utils/formats.js';
  import { dialog } from '$stores/ui/dialog.svelte.js';
  import { tunnels } from '$stores/resources/tunnels.svelte.js';
  import { useResource } from '$stores/lib/createResource.svelte.js';
  import Button from '$components/ui/Button.svelte';

  let { sessionID = '' } = $props();

  useResource(tunnels);

  let proxies = $derived(tunnels.data?.proxies || []);
  let rportfwds = $derived(tunnels.data?.rportfwds || []);

  async function doStartSocks() {
    const port = await dialog.prompt('Local SOCKS port:', 'Start SOCKS Proxy', '1080');
    if (!port) return;
    try {
      await StartSocks(sessionID, '127.0.0.1:' + port, '', '');
      await tunnels.refresh();
    } catch (err) {
      await dialog.alert(errorMessage(err, 'SOCKS failed'), 'Error');
    }
  }

  async function doStopSocks(id) {
    try {
      await StopSocks(id);
      await tunnels.refresh();
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Stop failed'), 'Error');
    }
  }

  async function doStartPortfwd() {
    const bind = await dialog.prompt('Local bind address (e.g. :8080):', 'Start Portfwd', ':8080');
    if (!bind) return;
    const remote = await dialog.prompt('Remote target (e.g. 10.0.0.5:80):', 'Remote Address');
    if (!remote) return;
    try {
      await StartPortfwd(sessionID, bind, remote);
      await tunnels.refresh();
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Portfwd failed'), 'Error');
    }
  }

  async function doStopPortfwd(id) {
    try {
      await StopPortfwd(id);
      await tunnels.refresh();
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Stop failed'), 'Error');
    }
  }

  async function doStartRportfwd() {
    const bindPort = await dialog.prompt('Port for implant to listen on:', 'Reverse Portfwd', '4444');
    if (!bindPort) return;
    const fwdPort = await dialog.prompt('Forward to local port:', 'Forward Port', bindPort);
    if (!fwdPort) return;
    try {
      await StartRportfwd(sessionID, '0.0.0.0', parseInt(bindPort), '127.0.0.1', parseInt(fwdPort));
      await tunnels.refresh();
    } catch (err) {
      await dialog.alert(errorMessage(err, 'RPortfwd failed'), 'Error');
    }
  }

  async function doStopRportfwd(id, sessionId) {
    try {
      await StopRportfwd(id, sessionId);
      await tunnels.refresh();
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Stop failed'), 'Error');
    }
  }
</script>

<div class="flex flex-col h-full">
  <div class="tab-header flex justify-end gap-2">
    <Button color="primary" size="xs" icon="plug" onclick={doStartSocks}>SOCKS</Button>
    <Button color="primary" size="xs" icon="arrow-right" onclick={doStartPortfwd}>Portfwd</Button>
    <Button color="primary" size="xs" icon="arrow-left" onclick={doStartRportfwd}>RPortfwd</Button>
    <Button color="dark" size="xs" onclick={() => tunnels.refresh()}>Refresh</Button>
  </div>

  <div class="flex-1 overflow-y-auto p-2">
    {#if tunnels.error}
      <div class="text-danger-500 p-2 text-xs">{tunnels.error}</div>
    {/if}
    <h3 class="text-fg text-xs font-semibold mt-0 mb-1">SOCKS Proxies</h3>
    <table class="w-full border-collapse text-xs [&_th]:text-left [&_th]:px-2 [&_th]:py-1 [&_th]:font-medium [&_th]:border-b [&_th]:border-line [&_td]:px-2 [&_td]:py-1 [&_td]:border-b [&_td]:border-line [&_tr:hover]:bg-row-hover">
      <thead class="text-fg-muted"><tr><th>Agent</th><th>Bind</th><th>User</th><th>In</th><th>Out</th><th></th></tr></thead>
      <tbody>
        {#each proxies.filter((p) => p.kind === 'socks') as p}
          <tr>
            <td class="font-mono">{p.sessionId?.slice(0, 8)}</td>
            <td class="font-mono">{p.bindAddr}</td>
            <td>{p.username || '-'}</td>
            <td class="font-mono">{formatBytes(p.bytesIn)}</td>
            <td class="font-mono">{formatBytes(p.bytesOut)}</td>
            <td><Button color="red" size="xs" onclick={() => doStopSocks(p.id)}>Stop</Button></td>
          </tr>
        {:else}
          <tr><td colspan="6" class="text-center p-3 text-fg-muted">No SOCKS proxies active.</td></tr>
        {/each}
      </tbody>
    </table>

    <h3 class="text-fg text-xs font-semibold mt-3 mb-1">Port Forwards</h3>
    <table class="w-full border-collapse text-xs [&_th]:text-left [&_th]:px-2 [&_th]:py-1 [&_th]:font-medium [&_th]:border-b [&_th]:border-line [&_td]:px-2 [&_td]:py-1 [&_td]:border-b [&_td]:border-line [&_tr:hover]:bg-row-hover">
      <thead class="text-fg-muted"><tr><th>Agent</th><th>Local</th><th>Remote</th><th></th></tr></thead>
      <tbody>
        {#each proxies.filter((p) => p.kind === 'portfwd') as p}
          <tr>
            <td class="font-mono">{p.sessionId?.slice(0, 8)}</td>
            <td class="font-mono">{p.bindAddr}</td>
            <td class="font-mono">{p.remoteAddr}</td>
            <td><Button color="red" size="xs" onclick={() => doStopPortfwd(p.id)}>Stop</Button></td>
          </tr>
        {:else}
          <tr><td colspan="4" class="text-center p-3 text-fg-muted">No port forwards active.</td></tr>
        {/each}
      </tbody>
    </table>

    <h3 class="text-fg text-xs font-semibold mt-3 mb-1">Reverse Port Forwards</h3>
    <table class="w-full border-collapse text-xs [&_th]:text-left [&_th]:px-2 [&_th]:py-1 [&_th]:font-medium [&_th]:border-b [&_th]:border-line [&_td]:px-2 [&_td]:py-1 [&_td]:border-b [&_td]:border-line [&_tr:hover]:bg-row-hover">
      <thead class="text-fg-muted"><tr><th>Agent</th><th>Implant Bind</th><th>Forwards To</th><th></th></tr></thead>
      <tbody>
        {#each rportfwds as p}
          <tr>
            <td class="font-mono">{p.sessionId?.slice(0, 8)}</td>
            <td class="font-mono">{p.bindAddr}</td>
            <td class="font-mono">{p.remoteAddr}</td>
            <td><Button color="red" size="xs" onclick={() => doStopRportfwd(p.id, p.sessionId)}>Stop</Button></td>
          </tr>
        {:else}
          <tr><td colspan="4" class="text-center p-3 text-fg-muted">No reverse port forwards active.</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
