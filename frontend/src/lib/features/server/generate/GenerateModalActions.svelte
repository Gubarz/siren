<script>
  import PresetPicker from '$components/forms/PresetPicker.svelte'
  import Button from '$components/ui/Button.svelte'

  let {
    presetValues = {},
    onapply,
    oncancel,
    onsave,
    ongenerate,
    savingProfile = false,
    generating = false,
    canGenerate = false,
    generateLabel = 'Generate',
  } = $props()
</script>

<div class="flex justify-between items-center mt-5">
  <PresetPicker
    commandPath="generate"
    currentValues={presetValues}
    {onapply}
  />
  <div class="flex gap-2">
    <Button color="dark" onclick={oncancel}>Cancel</Button>
    <Button color="dark" onclick={onsave} disabled={savingProfile || generating || !canGenerate}>
      {savingProfile ? 'Saving...' : 'Save as Profile'}
    </Button>
    <Button color="primary" onclick={ongenerate} disabled={generating || savingProfile || !canGenerate}>
      {generating ? 'Building... (may take ~1 min)' : generateLabel}
    </Button>
  </div>
</div>
