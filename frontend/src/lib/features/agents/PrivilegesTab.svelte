<script>
  import { onMount } from 'svelte'
  import Button from '$components/ui/Button.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import ErrorState from '$components/ui/ErrorState.svelte'
  import PrivilegeTable from '$components/ui/PrivilegeTable.svelte'
  import { integrityBadge } from '$utils/privileges.js'
  import { GetTokenPrivs, RevToSelfToken } from '../../api/token.js'
  import { errorMessage } from '../../utils/errors.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'

  let { sessionID = '' } = $props()

  let privData = $state(null)
  let loading = $state(false)
  let error = $state('')

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    privData = null
    try {
      privData = (await GetTokenPrivs(sessionID)) ?? null
    } catch (err) {
      error = errorMessage(err)
    } finally {
      loading = false
    }
  }

  async function revToSelf() {
    if (!await dialog.confirm('Revert to original process token? This will drop any impersonation.', 'Rev2Self')) return
    try {
      await RevToSelfToken(sessionID)
      await refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err), 'Rev2Self failed')
    }
  }

  let badge = $derived(integrityBadge(privData?.ProcessIntegrity))
</script>

<div class="flex flex-col h-full">
  <div class="tab-header flex items-center gap-2 py-2 pl-2">
    {#if privData}
      <span class="font-mono text-xs">{privData.ProcessName || 'Current Process'}</span>
      {#if badge.label}
        <Badge variant={badge.variant}>{badge.label}</Badge>
      {/if}
    {/if}
    <div class="flex-1"></div>
    <div class="flex items-center gap-2">
      <Button color="dark" size="xs" onclick={refresh} disabled={loading}>Refresh</Button>
      <Button color="warning" size="xs" onclick={revToSelf}>Rev2Self</Button>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto">
    {#if error}
      <ErrorState {error} title="Failed to load privileges" class="m-2" />
    {/if}
    <PrivilegeTable {privData} {loading} />
  </div>
</div>
