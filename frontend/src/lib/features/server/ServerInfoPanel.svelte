<script>
  import { onMount } from 'svelte'
  import Button from '$components/ui/Button.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import PanelBody from '$components/patterns/PanelBody.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import {
    GetCertificateAuthorityInfo,
    GetCompiler,
    RestartJobs,
  } from '../../api/operatorControls.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'

  // Compact server introspection surface: cross-compile toolchains the
  // server can invoke, CA fingerprint, and a "restart all jobs" panic
  // button for when a listener wedges after a config change.

  let { embedded = false, onclose } = $props()

  let ca = $state(null)
  let compiler = $state(null)
  let loading = $state(false)
  let error = $state('')

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      const [caResp, compilerResp] = await Promise.all([
        GetCertificateAuthorityInfo(),
        GetCompiler(),
      ])
      ca = caResp
      compiler = compilerResp
    } catch (err) {
      error = errorMessage(err, 'Failed to load server info: ')
    } finally {
      loading = false
    }
  }

  async function restartAll() {
    if (!(await dialog.confirm(
      'Restart every registered job? Active listeners will drop and reopen; live sessions survive but beacons may miss a callback.',
      'Restart Jobs',
    ))) return
    try {
      await RestartJobs([])
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Restart failed: '), 'Restart Jobs')
    }
  }

  function targets() {
    const list = compiler?.Targets || compiler?.targets || []
    const seen = new Set()
    for (const t of list) {
      const key = `${t.GOOS || t.goos || '?'}/${t.GOARCH || t.goarch || '?'}`
      seen.add(key)
    }
    return [...seen].sort()
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Server'} icon={embedded ? '' : 'server'}>
  <Toolbar class="justify-end">
    <Button color="dark" size="sm" onclick={refresh} disabled={loading}>
      {loading ? 'Loading…' : 'Refresh'}
    </Button>
    <Button color="red" size="sm" onclick={restartAll}>Restart all jobs</Button>
  </Toolbar>

  <PanelBody error={error || null} empty={false}>
    <div class="grid gap-4 p-3 text-xs">
      <section>
        <h3 class="mb-2 text-sm font-semibold text-fg">Certificate Authority</h3>
        {#if ca}
          <div class="grid grid-cols-4 gap-x-3 gap-y-1 font-mono">
            <div class="text-fg-muted">Fingerprint</div>
            <div class="col-span-3 break-all">{ca.Fingerprint || ca.fingerprint || '—'}</div>
            <div class="text-fg-muted">Certificate</div>
            <pre class="col-span-3 whitespace-pre-wrap break-all text-fg-muted">{ca.Certificate || ca.certificate || ''}</pre>
          </div>
        {:else}
          <p class="text-fg-muted">—</p>
        {/if}
      </section>

      <section>
        <h3 class="mb-2 text-sm font-semibold text-fg">Compiler</h3>
        {#if compiler}
          <div class="grid grid-cols-4 gap-x-3 gap-y-1">
            <div class="text-fg-muted">Server GOOS/GOARCH</div>
            <div class="col-span-3 font-mono">{compiler.GOOS || compiler.goos || '?'} / {compiler.GOARCH || compiler.goarch || '?'}</div>
            <div class="text-fg-muted">Cross-compile matrix</div>
            <div class="col-span-3 flex flex-wrap gap-1 font-mono">
              {#each targets() as t}
                <span class="rounded border border-line bg-chrome px-2 py-1">{t}</span>
              {/each}
            </div>
          </div>
        {:else}
          <p class="text-fg-muted">—</p>
        {/if}
      </section>
    </div>
  </PanelBody>
</Panel>
