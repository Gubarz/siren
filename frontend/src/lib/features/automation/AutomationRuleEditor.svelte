<script>
  import Button from '$components/ui/Button.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import Select from '$components/ui/Select.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import TextArea from '$components/ui/TextArea.svelte'
  import EmptyState from '$components/ui/EmptyState.svelte'
  import Field from '$components/ui/Field.svelte'
  import Tabs from '$components/patterns/Tabs.svelte'
  import { sessions } from '$stores/resources/sessions.svelte.js'
  import { beacons } from '$stores/resources/beacons.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'
  import { matchesAutomationTarget } from '../../utils/automation.js'
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js'

  useResource(sessions, beacons)

  let {
    draft = $bindable(null),
    manualTarget = $bindable(''),
    busy = false,
    onnew,
    ondirty,
    onfilter,
    oncommands,
    onsave,
    ondelete,
    onrun,
  } = $props()

  let activeTab = $state('trigger')

  let targets = $derived([
    ...(Array.isArray(sessions.data) ? sessions.data : []).map((t) => ({ ...t, _kind: 'session' })),
    ...(Array.isArray(beacons.data) ? beacons.data : []).map((t) => ({ ...t, _kind: 'beacon' })),
  ])
  let matchingTargets = $derived(draft ? targets.filter((t) => matchesAutomationTarget(t, draft)) : targets)

  const triggerOptions = [
    { value: 'session-connected', label: 'Session connected' },
    { value: 'beacon-registered', label: 'Beacon registered' },
    { value: 'beacon-checkin', label: 'Beacon check-in' },
    { value: 'interval', label: 'Recurring interval' },
    { value: 'manual', label: 'Manual only' },
  ]

  const targetKindOptions = [
    { value: 'any', label: 'Sessions & beacons' },
    { value: 'session', label: 'Sessions only' },
    { value: 'beacon', label: 'Beacons only' },
  ]

  const modeOptions = [
    { value: 'commands', label: 'Commands' },
    { value: 'javascript', label: 'JavaScript' },
  ]

  let targetOptions = $derived([
    { value: '', label: 'All matching targets' },
    ...matchingTargets.map((t) => ({ value: t.ID, label: `${t.Name || t.Hostname || t.ID} (${t._kind})` })),
  ])

  let tabs = $derived([
    { id: 'trigger', label: 'Trigger', icon: 'bolt', badge: matchingTargets.length || undefined },
    { id: 'workflow', label: 'Workflow', icon: 'terminal' },
    { id: 'advanced', label: 'Advanced', icon: 'sliders' },
  ])

  function updateFilter(key, value) { onfilter?.({ key, value }) }
  function updateCommands(value) { oncommands?.(value) }
</script>

{#if !draft}
  <EmptyState
    icon="workflow"
    title="Build repeatable agent workflows"
    description="Choose a rule from the list or create a new one. Rules run commands or Sobek JavaScript against matching sessions and beacons."
    class="m-4"
  >
    <Button color="primary" onclick={onnew}>Create automation</Button>
  </EmptyState>
{:else}
  <div class="flex flex-col h-full min-h-0 bg-canvas">
    <header class="flex items-center gap-3 px-5 py-3 border-b border-line bg-chrome-header shrink-0">
      <Checkbox bind:checked={draft.enabled} onchange={ondirty} />
      <TextInput
        class="flex-1! text-base! font-medium!"
        bind:value={draft.name}
        oninput={ondirty}
        placeholder="Rule name"
      />
      <span class="text-xs text-fg-muted whitespace-nowrap">
        {draft.runCount || 0} run{draft.runCount === 1 ? '' : 's'}
      </span>
      {#if draft.id}
        <Button color="alternative" size="sm" icon="message-square" onclick={() => commentsModal.openComments('automation', draft.id, draft.name || draft.id)} title="Comments" />
        <Button color="alternative" size="sm" icon="trash" onclick={ondelete} disabled={busy} title="Delete rule" />
        <div class="w-px h-5 bg-line mx-1"></div>
        <Select bind:value={manualTarget} options={targetOptions} class="max-w-56!" />
        <Button size="sm" onclick={onrun} disabled={busy || matchingTargets.length === 0} icon="play">Run now</Button>
      {/if}
      <Button color="primary" size="sm" onclick={onsave} disabled={busy || !draft.__dirty}>
        {busy ? 'Working…' : draft.__dirty ? 'Save' : 'Saved'}
      </Button>
    </header>

    <div class="px-5 pt-2 shrink-0">
      <Tabs {tabs} bind:active={activeTab} />
    </div>

    <div class="flex-1 min-h-0 overflow-auto px-5 py-4">
      {#if activeTab === 'trigger'}
        <div class="flex flex-col gap-4 max-w-3xl">
          <div class="grid grid-cols-2 gap-3">
            <Field label="Trigger">
              <Select bind:value={draft.trigger} options={triggerOptions} onchange={ondirty} />
            </Field>
            <Field label="Target type">
              <Select bind:value={draft.targetKind} options={targetKindOptions} onchange={ondirty} />
            </Field>
          </div>

          {#if draft.trigger === 'interval'}
            <Field label="Interval (seconds)" hint="Minimum 10 seconds.">
              <TextInput type="number" bind:value={draft.intervalSeconds} oninput={ondirty} />
            </Field>
          {/if}

          <Field label="Description" hint="Optional — shows in the rule list.">
            <TextInput bind:value={draft.description} oninput={ondirty} placeholder="What this workflow does" />
          </Field>

          <div>
            <div class="flex items-center justify-between mb-2">
              <h3 class="text-sm font-medium m-0">Filters</h3>
              <span class="text-xs text-fg-muted">Comma-separated glob patterns. Blank = match everything.</span>
            </div>
            <div class="grid grid-cols-2 gap-2">
              <TextInput value={draft.filter.os} oninput={(e) => updateFilter('os', e.currentTarget.value)} placeholder="OS (e.g. windows,linux)" />
              <TextInput value={draft.filter.arch} oninput={(e) => updateFilter('arch', e.currentTarget.value)} placeholder="Arch (e.g. amd64)" />
              <TextInput value={draft.filter.hostname} oninput={(e) => updateFilter('hostname', e.currentTarget.value)} placeholder="Hostname (e.g. prod-*)" />
              <TextInput value={draft.filter.username} oninput={(e) => updateFilter('username', e.currentTarget.value)} placeholder="Username (e.g. admin*)" />
              <TextInput class="col-span-2!" value={draft.filter.name} oninput={(e) => updateFilter('name', e.currentTarget.value)} placeholder="Agent name" />
            </div>
          </div>

          <div class="flex items-center gap-2 text-xs text-fg-muted mt-1">
            <Icon name="bullseye" size={14} class="text-brand" />
            <span><strong class="text-fg">{matchingTargets.length}</strong> current target{matchingTargets.length === 1 ? '' : 's'} match this rule.</span>
          </div>
        </div>

      {:else if activeTab === 'workflow'}
        <div class="flex flex-col gap-3 h-full">
          <div class="flex items-center justify-between gap-3">
            <div class="text-xs text-fg-muted">
              {#if draft.executionMode === 'javascript'}
                API: <code>sliver.run(cmd)</code>, <code>sliver.log(…)</code>, <code>sliver.sleep(ms)</code>. Context: <code>target</code>, <code>trigger</code>.
              {:else}
                One command per line. Templates: <code>{'{{name}}'}</code>, <code>{'{{hostname}}'}</code>, <code>{'{{os}}'}</code>, <code>{'{{arch}}'}</code>.
              {/if}
            </div>
            <Select bind:value={draft.executionMode} options={modeOptions} onchange={ondirty} class="max-w-40!" />
          </div>

          {#if draft.executionMode === 'javascript'}
            <TextArea class="font-mono! w-full! flex-1! min-h-80!" bind:value={draft.script} oninput={ondirty} spellcheck="false" />
          {:else}
            <TextArea
              class="font-mono! w-full! flex-1! min-h-80!"
              value={draft.commands.join('\n')}
              oninput={(e) => updateCommands(e.currentTarget.value)}
              placeholder="info&#10;whoami&#10;ps"
              spellcheck="false"
            />
          {/if}
        </div>

      {:else}
        <div class="flex flex-col gap-4 max-w-3xl">
          <div class="grid grid-cols-2 gap-3">
            <Field label="Run timeout (s)" hint="Includes waiting for beacon tasks.">
              <TextInput type="number" bind:value={draft.timeoutSeconds} oninput={ondirty} />
            </Field>
            <Field label="Per-target cooldown (s)" hint="Skip re-runs against the same target for this long.">
              <TextInput type="number" bind:value={draft.cooldownSeconds} oninput={ondirty} />
            </Field>
            {#if draft.executionMode !== 'javascript'}
              <Field label="Delay between steps (s)">
                <TextInput type="number" bind:value={draft.delaySeconds} oninput={ondirty} />
              </Field>
            {/if}
            <Field label="Maximum runs" hint="0 = unlimited.">
              <TextInput type="number" bind:value={draft.maxRuns} oninput={ondirty} />
            </Field>
          </div>
          {#if draft.executionMode === 'commands'}
            <Checkbox bind:checked={draft.continueOnError} onchange={ondirty} label="Continue after command errors" />
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/if}
