<script>
  import Button from '$components/ui/Button.svelte'
  import PresetPicker from '$components/forms/PresetPicker.svelte'

  let {
    commandPath,
    currentValues = {},
    onapply,
    primaryLabel = 'Execute',
    onprimary,
    primaryDisabled = false,
    oncancel,
  } = $props()
</script>

<div class="flex justify-between items-center">
  <PresetPicker {commandPath} {currentValues} {onapply} />
  <div class="flex gap-2">
    <Button color="dark" onclick={oncancel}>Cancel</Button>
    <Button color="primary" onclick={onprimary} disabled={primaryDisabled}>{primaryLabel}</Button>
  </div>
</div>
