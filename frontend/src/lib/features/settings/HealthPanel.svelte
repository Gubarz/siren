<script>
  import { onMount } from 'svelte'
  import Icon from '$components/ui/Icon.svelte'
  import { HealthSnapshot } from '../../api/health.js'

  // Live connection-quality view. Polls App.HealthSnapshot every 5s while
  // mounted — small enough round-trip that we don't need a shared store,
  // big enough that we don't want a 1s cadence saturating the RPC.

  const POLL_MS = 5000
  const HISTORY = 24

  let snap = $state(null)
  let history = $state([])
  let error = $state('')

  onMount(() => {
    tick()
    const timer = setInterval(tick, POLL_MS)
    return () => clearInterval(timer)
  })

  async function tick() {
    try {
      const s = await HealthSnapshot()
      snap = s
      error = ''
      history = [...history.slice(-HISTORY + 1), { t: s.checkedAt, latency: s.rpcLatencyMs, ok: !s.rpcError }]
    } catch (err) {
      error = String(err)
    }
  }

  let maxLatency = $derived(Math.max(1, ...history.map((h) => h.latency)))
  let statusLabel = $derived(!snap ? 'Checking…' :
    !snap.connected ? 'Disconnected' :
    snap.rpcError ? 'Errors' :
    snap.rpcLatencyMs > 500 ? 'Degraded' : 'Healthy')
  let statusVariant = $derived(!snap?.connected ? 'danger' :
    snap.rpcError ? 'warning' :
    snap.rpcLatencyMs > 500 ? 'warning' : 'success')
</script>

<div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
  <div class="flex items-center gap-2 mb-2">
    <h3 class="m-0 text-fg text-base flex-1">Connection health</h3>
    <span
      class="text-xs px-2 py-1 rounded font-mono"
      class:bg-success-500={statusVariant === 'success'}
      class:bg-warning-500={statusVariant === 'warning'}
      class:bg-danger-500={statusVariant === 'danger'}
    >
      {statusLabel}
    </span>
  </div>
  <p class="text-xs mt-1 mb-4 text-fg-muted">
    Live snapshot updated every {POLL_MS / 1000}s. Round-trip is a
    `GetVersion` ping — anything under 100ms is fast, over 500ms warrants
    a closer look.
  </p>

  {#if error}
    <div class="text-sm text-danger-500 mb-3">Snapshot failed: {error}</div>
  {/if}

  {#if snap}
    <div class="grid grid-cols-2 gap-3 mb-4">
      <div class="border border-line rounded px-3 py-2">
        <div class="text-xs uppercase tracking-wider text-fg-muted">RPC latency</div>
        <div class="text-2xl font-mono">{snap.rpcLatencyMs}<span class="text-sm text-fg-muted"> ms</span></div>
      </div>
      <div class="border border-line rounded px-3 py-2">
        <div class="text-xs uppercase tracking-wider text-fg-muted">Sessions / Beacons</div>
        <div class="text-2xl font-mono">{snap.sessionCount} <span class="text-fg-muted">/</span> {snap.beaconCount}</div>
      </div>
      <div class="border border-line rounded px-3 py-2">
        <div class="text-xs uppercase tracking-wider text-fg-muted">Active jobs</div>
        <div class="text-2xl font-mono">{snap.jobCount}</div>
      </div>
      <div class="border border-line rounded px-3 py-2">
        <div class="text-xs uppercase tracking-wider text-fg-muted">Tunnels</div>
        <div class="text-2xl font-mono">{snap.tunnelCount}</div>
        <div class="text-xs text-fg-muted">
          SOCKS {snap.socksCount} · portfwd {snap.portfwdCount} · rportfwd {snap.rportfwdCount}
        </div>
      </div>
    </div>

    {#if history.length > 1}
      <div class="border border-line rounded px-3 py-3">
        <div class="text-xs uppercase tracking-wider text-fg-muted mb-2">RPC latency history</div>
        <div class="flex items-end h-16 gap-1">
          {#each history as h}
            {@const pct = Math.max(4, Math.round((h.latency / maxLatency) * 100))}
            <div class="flex-1 min-w-1 rounded-t" style="height: {pct}%"
              class:bg-success-500={h.ok && h.latency < 250}
              class:bg-warning-500={h.ok && h.latency >= 250 && h.latency < 500}
              class:bg-danger-500={!h.ok || h.latency >= 500}
              title="{h.latency}ms"
            ></div>
          {/each}
        </div>
        <div class="mt-1 flex justify-between text-xs text-fg-muted">
          <span>{history.length} samples</span>
          <span>peak {maxLatency}ms</span>
        </div>
      </div>
    {/if}

    {#if snap.rpcError}
      <div class="mt-3 text-xs text-danger-500 flex items-start gap-2">
        <Icon name="alert-triangle" size={14} class="shrink-0 mt-1" />
        <div>{snap.rpcError}</div>
      </div>
    {/if}
  {/if}
</div>
