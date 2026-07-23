<script>
  import { onMount } from 'svelte'
  import Modal from '$components/patterns/Modal.svelte'
  import Button from '$components/ui/Button.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import PrivilegeTable from '$components/ui/PrivilegeTable.svelte'
  import { integrityBadge } from '$utils/privileges.js'
  import { GetTokenPrivs, RevToSelfToken } from '../../../../api/token.js'
  import { errorMessage } from '../../../../utils/errors.js'

  let {
    open = $bindable(false),
    firstSessionID = '',
    onclose,
  } = $props()

  let privData = $state(null)
  let loading = $state(false)
  let error = $state('')

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      privData = (await GetTokenPrivs(firstSessionID)) ?? null
    } catch (err) {
      error = errorMessage(err)
    } finally {
      loading = false
    }
  }

  async function revToSelf() {
    try {
      await RevToSelfToken(firstSessionID)
      await refresh()
    } catch (err) {
      error = errorMessage(err, 'Rev2Self failed: ')
    }
  }

  let badge = $derived(integrityBadge(privData?.ProcessIntegrity))
</script>

<Modal bind:open title="Get Privileges" size="3xl" {onclose}>
  {#if loading}
    <p class="text-fg-muted text-sm">Loading privileges...</p>
  {:else if error}
    <p class="text-danger-500 text-sm mb-4">{error}</p>
    <div class="flex justify-end gap-2">
      <Button color="dark" onclick={refresh}>Retry</Button>
    </div>
  {:else if privData}
    <div class="flex items-center gap-2 mb-4">
      <span class="text-sm font-mono">{privData.ProcessName || 'Current Process'}</span>
      {#if badge.label}
        <Badge variant={badge.variant}>{badge.label}</Badge>
      {/if}
    </div>

    <div class="max-h-96 overflow-auto">
      <PrivilegeTable {privData} />
    </div>
  {:else}
    <p class="text-fg-muted text-sm">No data received.</p>
  {/if}

  {#snippet footer()}
    <div class="flex justify-between items-center">
      <Button color="warning" size="xs" onclick={revToSelf}>Rev2Self</Button>
      <div class="flex gap-2">
        {#if privData}
          <Button color="dark" size="xs" onclick={refresh}>Refresh</Button>
        {/if}
        <Button color="dark" onclick={() => open = false}>Close</Button>
      </div>
    </div>
  {/snippet}
</Modal>
