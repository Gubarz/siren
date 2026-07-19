<script>
  import Modal from '$components/patterns/Modal.svelte'
  import Button from '$components/ui/Button.svelte'
  import LoadingState from '$components/ui/LoadingState.svelte'
  import ErrorState from '$components/ui/ErrorState.svelte'
  import { GetStarterAutomationRules, ImportStarterAutomationRule } from '../../api/automation.js'
  import { errorMessage } from '../../utils/errors.js'
  import { toast } from '$stores/ui/toast.svelte.js'

  let { open = $bindable(false), onimported } = $props()

  let rules = $state([])
  let loading = $state(false)
  let error = $state('')
  let busyID = $state('')

  $effect(() => {
    if (open) load()
  })

  async function load() {
    loading = true
    error = ''
    try {
      rules = await GetStarterAutomationRules()
    } catch (e) {
      error = errorMessage(e)
      rules = []
    } finally {
      loading = false
    }
  }

  async function importOne(rule) {
    busyID = rule.id
    try {
      await ImportStarterAutomationRule(rule.id)
      toast.push({ variant: 'success', message: `Imported "${rule.name}" (disabled — review before enabling)` })
      onimported?.()
    } catch (e) {
      toast.push({ variant: 'error', message: `Import failed: ${errorMessage(e)}` })
    } finally {
      busyID = ''
    }
  }

  function triggerLabel(v) {
    return ({
      'session-connected': 'Session connected',
      'beacon-registered': 'Beacon registered',
      'beacon-checkin': 'Beacon check-in',
      interval: 'Recurring interval',
      manual: 'Manual only',
    })[v] || v
  }
</script>

<Modal bind:open title="Starter automation library" icon="package" size="2xl">
  <p class="text-sm text-fg-muted mb-3">
    Read-only starter rules. Importing copies a rule into your library
    (disabled by default) — review filters/commands before enabling.
  </p>
  {#if loading}
    <LoadingState />
  {:else if error}
    <ErrorState error={error} title="Could not load starter library" />
  {:else}
    <div class="flex flex-col gap-2">
      {#each rules as rule (rule.id)}
        <div class="border border-line rounded p-3 flex gap-3 items-start">
          <div class="flex-1 min-w-0">
            <div class="font-medium">{rule.name}</div>
            <div class="text-xs text-fg-muted mb-1">
              {triggerLabel(rule.trigger)} &middot; {rule.targetKind || 'any'}
              {#if rule.filter?.os}&middot; os={rule.filter.os}{/if}
              {#if rule.filter?.name}&middot; name={rule.filter.name}{/if}
            </div>
            <div class="text-sm">{rule.description}</div>
          </div>
          <Button
            size="sm"
            color="primary"
            disabled={busyID === rule.id}
            onclick={() => importOne(rule)}
          >{busyID === rule.id ? 'Importing…' : 'Import'}</Button>
        </div>
      {/each}
    </div>
  {/if}
</Modal>
