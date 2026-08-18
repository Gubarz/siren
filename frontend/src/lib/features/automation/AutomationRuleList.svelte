<script>
  import Icon from '$components/ui/Icon.svelte'
  import Button from '$components/ui/Button.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import EmptyState from '$components/ui/EmptyState.svelte'
  import { formatDateTime } from '../../utils/formats.js'

  let {
    rules = [],
    selectedID = '',
    selectedRun = null,
    activePanel = '',
    history = [],
    onnew,
    onselect,
    ontoggle,
    onhistory,
    onhistoryselect,
    onclearhistory,
    onexport,
    onimport,
    onstarters,
  } = $props()

  let ruleItems = $derived(Array.isArray(rules) ? rules : [])
  let historyItems = $derived(Array.isArray(history) ? history : [])

  function triggerLabel(value) {
    const map = {
      'session-connected': 'Session connected',
      'beacon-registered': 'Beacon registered',
      'beacon-checkin': 'Beacon check-in',
      interval: 'Recurring interval',
      manual: 'Manual only',
    }
    return map[value] || value
  }

  function runIcon(status) {
    if (status === 'completed') return 'check'
    if (status === 'running') return 'loader'
    return 'x'
  }
</script>

<aside class="w-72 shrink-0 overflow-hidden flex flex-col border-r border-line bg-chrome">
  {#if activePanel === 'history'}
    <div class="flex items-center justify-between gap-3 p-5 border-b border-line">
      <div class="min-w-0">
        <h2 class="m-0 text-lg">Execution history</h2>
        <span class="text-xs text-fg-muted">{historyItems.length} run{historyItems.length === 1 ? '' : 's'}</span>
      </div>
      <Button color="alternative" size="sm" icon="trash" onclick={onclearhistory} disabled={historyItems.length === 0} title="Clear history" />
    </div>

    <div class="flex-1 overflow-auto p-2">
      {#if historyItems.length === 0}
        <EmptyState title="" description="No automation runs recorded." icon="" class="py-6" />
      {/if}
      {#each historyItems as run (run.id)}
        <Button
          color="alternative"
          class={`w-full! min-w-0! max-w-full! overflow-hidden! whitespace-normal! flex! items-start! justify-start! gap-2 px-2! py-3! border! rounded! bg-transparent! text-fg! text-left ${selectedRun?.id === run.id ? 'border-brand! bg-panel!' : 'border-transparent! hover:bg-row-hover!'}`}
          onclick={() => onhistoryselect?.(run)}
        >
          <Badge variant={run.status} size="xs" class="mt-1 shrink-0">
            <Icon name={runIcon(run.status)} size={10} />
          </Badge>
          <span class="flex min-w-0 flex-1 flex-col gap-1 overflow-hidden">
            <strong class="block min-w-0 truncate">{run.ruleName}</strong>
            <small class="block min-w-0 truncate text-fg-muted">{run.targetName || 'No target'} &middot; {run.trigger}</small>
            <time class="block min-w-0 truncate text-xs text-fg-muted">{formatDateTime(run.startedAt)}</time>
          </span>
        </Button>
      {/each}
    </div>
  {:else}
    <div class="flex items-center justify-between gap-3 p-5 border-b border-line">
      <div class="min-w-0">
        <h2 class="m-0 text-lg">Automation</h2>
        <span class="text-xs text-fg-muted">{ruleItems.length} rule{ruleItems.length === 1 ? '' : 's'}</span>
      </div>
      <Button color="primary" size="sm" icon="plus" onclick={onnew}>New</Button>
    </div>

    <div class="grid grid-cols-3 gap-1 px-3 py-2 border-b border-line">
      <Button color="alternative" size="sm" icon="package" class="min-w-0!" onclick={onstarters} title="Starter library">Starters</Button>
      <Button color="alternative" size="sm" icon="upload" class="min-w-0!" onclick={onimport} title="Import rules">Import</Button>
      <Button color="alternative" size="sm" icon="download" class="min-w-0!" onclick={onexport} title="Export all rules" disabled={ruleItems.length === 0}>Export</Button>
    </div>

    <div class="flex-1 overflow-auto p-2">
      {#if ruleItems.length === 0}
        <EmptyState title="" description="No automation rules yet." icon="" class="py-6" />
      {/if}
      {#each ruleItems as rule (rule.id)}
        <Button
          color="alternative"
          class={`w-full! min-w-0! max-w-full! overflow-hidden! whitespace-normal! flex! items-center! justify-start! gap-2 px-2! py-3! border! rounded! bg-transparent! text-fg! text-left ${selectedID === rule.id ? 'border-brand! bg-panel!' : 'border-transparent! hover:bg-row-hover!'}`}
          onclick={() => onselect?.(rule)}
        >
          <span
          class={`w-2 h-2 flex-none rounded-full border border-line ${rule.enabled ? 'bg-success-500' : 'bg-fg-muted'}`}
          role="switch"
          tabindex="0"
          aria-checked={rule.enabled}
          title={rule.enabled ? 'Disable rule' : 'Enable rule'}
          onclick={(e) => { e.stopPropagation(); ontoggle?.(rule) }}
          onkeydown={(e) => { e.stopPropagation(); if (e.key === 'Enter' || e.key === ' ') ontoggle?.(rule) }}
          ></span>
          <span class="flex min-w-0 flex-1 flex-col gap-1 overflow-hidden">
            <strong class="block min-w-0 truncate">{rule.name}</strong>
            <small class="block min-w-0 truncate text-fg-muted">{triggerLabel(rule.trigger)} &middot; {rule.targetKind || 'any'}</small>
          </span>
        </Button>
      {/each}
    </div>
  {/if}

  {#if activePanel === 'history'}
    <Button
      color="alternative"
      class="w-full! min-w-0! overflow-hidden! flex! items-center! justify-start! gap-2 px-5! py-3! border-0! border-t! border-line! rounded-none! bg-transparent! text-fg-muted hover:text-brand! hover:bg-panel! text-left"
      onclick={() => onhistory?.('rule')}
    >
      <Icon name="arrow-left" />
      <span class="min-w-0 truncate">Back to rules</span>
    </Button>
  {:else}
    <Button
      color="alternative"
      class={`w-full! min-w-0! overflow-hidden! flex! items-center! justify-start! gap-2 px-5! py-3! border-0! border-t! border-line! rounded-none! bg-transparent! text-left ${activePanel === 'history' ? 'text-brand! bg-panel!' : 'text-fg-muted hover:text-brand! hover:bg-panel!'}`}
      onclick={() => onhistory?.('history')}
    >
      <Icon name="history" />
      <span class="min-w-0 truncate">Run history</span>
      <strong class="ml-auto">{historyItems.length}</strong>
    </Button>
  {/if}
</aside>
