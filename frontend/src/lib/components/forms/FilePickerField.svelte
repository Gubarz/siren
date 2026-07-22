<script>
  import { onFileDrop } from '../../api/runtime.js'
  import Icon from '../ui/Icon.svelte'

  let { value = $bindable(''), label = 'File', description = '', accept = '' } = $props()

  let draggedOver = $state(false)
  let fileInput

  // Drop subscription — cleanup lives on the effect return.
  $effect(() => onFileDrop((x, y, paths) => {
    if (paths && paths.length > 0) {
      value = String(paths[0])
      draggedOver = false
    }
  }))

  function handleBrowse() {
    if (typeof window.go?.gui?.App?.OpenFileDialog === 'function') {
      window.go.gui.App.OpenFileDialog(label || 'Select file').then((path) => {
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
    if (typeof window.go?.gui?.App?.OpenFileDialog === 'function') {
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
