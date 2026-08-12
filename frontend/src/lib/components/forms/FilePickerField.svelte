<script>
  import { onFileDrop, OpenFileDialog } from '../../api/runtime.js'
  import Icon from '../ui/Icon.svelte'

  let { value = $bindable(''), label = 'File', description = '', accept = '' } = $props()

  let draggedOver = $state(false)
  let fileInput
  let dropZone

  const inWails = () => typeof window !== 'undefined' && window._wails?.flags != null

  // Drop subscription — cleanup lives on the effect return.
  // v3 only fires for drops on data-file-drop-target elements; guard that
  // the drop actually landed on this field's zone.
  $effect(() => onFileDrop((x, y, paths) => {
    const target = document.elementFromPoint(x, y)
    if (!dropZone?.contains(target)) return
    if (paths && paths.length > 0) {
      value = String(paths[0])
      draggedOver = false
    }
  }))

  function handleBrowse() {
    if (inWails()) {
      OpenFileDialog(label || 'Select file').then((path) => {
        if (path) value = path
      })
    }
  }

  function handleFileInputChange(e) {
    const file = e.target?.files?.[0]
    if (file) {
      if (file.path) { value = file.path }
      else if (file.name) { value = file.name }
    }
  }

  function handleClick() {
    if (inWails()) {
      handleBrowse()
      return
    }
    fileInput?.click()
  }
</script>

<div
  class="mb-2"
>
  <!-- hidden native input for click-to-browse + a11y -->
  <input
    bind:this={fileInput}
    type="file"
    {accept}
    onchange={handleFileInputChange}
    class="sr-only"
    tabindex="-1"
  />

  <div
    bind:this={dropZone}
    data-file-drop-target
    class={`flex flex-col items-center justify-center gap-2 px-4 py-5 border-2 border-dashed rounded-md cursor-pointer transition-colors outline-none text-fg-muted hover:border-brand hover:bg-brand/10 focus-visible:border-brand focus-visible:bg-brand/10 ${draggedOver ? 'border-brand bg-brand/10' : 'border-line'}`}
    role="button"
    tabindex="0"
    onclick={handleClick}
    onkeydown={(e) => e.key === 'Enter' && fileInput?.click()}
    ondragover={(e) => { e.preventDefault(); draggedOver = true }}
    ondragleave={() => draggedOver = false}
    ondrop={(e) => { e.preventDefault(); draggedOver = false }}
  >
    <Icon name="upload" size={24} />
    <span class="text-sm font-medium">{value || 'Drop file or click to browse'}</span>
  </div>

  {#if description}
    <span class="block text-xs mt-1 text-fg-muted">{description}</span>
  {/if}
</div>

{#if value}
  <div class="text-xs px-1 pt-1 truncate text-fg-muted">{value}</div>
{/if}
