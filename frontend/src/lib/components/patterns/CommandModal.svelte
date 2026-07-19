<script>
  import Modal from './Modal.svelte'
  import Button from '../ui/Button.svelte'
  import PresetPicker from '../forms/PresetPicker.svelte'

  let {
    open = $bindable(false),
    title = '',
    size = '2xl',
    description = '',
    cmdPreview = '',
    presetPath = '',
    presetValues = {},
    onpresetapply,
    onexecute,
    onclose,
    disabled = false,
    executeLabel = 'Execute',
    executeColor = 'primary',
    children,
  } = $props()

  function handleCancel() {
    open = false
    onclose?.()
  }
</script>

<Modal bind:open {title} {size} {onclose}>
  {#if description}
    <p class="text-fg-muted text-sm mb-4">{description}</p>
  {/if}

  {@render children?.()}

  {#if cmdPreview}
    <div class="mt-4 mb-4">
      <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
      <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview}</code>
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex justify-between items-center w-full">
      <div>
        {#if presetPath}
          <PresetPicker
            commandPath={presetPath}
            currentValues={presetValues}
            onapply={onpresetapply}
          />
        {/if}
      </div>
      <div class="flex gap-2">
        <Button color="dark" onclick={handleCancel}>Cancel</Button>
        <Button color={executeColor} onclick={onexecute} {disabled}>{executeLabel}</Button>
      </div>
    </div>
  {/snippet}
</Modal>
