<script>
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import Modal from '../patterns/Modal.svelte'

  let isOpen = $state(false)
  let inputEl = $state(null)

  $effect(() => { isOpen = dialog.isOpen })

  function close(result) {
    if (dialog.resolve) dialog.resolve(result)
    dialog.isOpen = false
    isOpen = false
  }

  function handleCancel() {
    close(dialog.type === 'prompt' ? null : false)
  }

  function handleConfirm() {
    close(dialog.type === 'prompt' ? dialog.inputValue : true)
  }

  function handleKeydown(e) {
    if (isOpen) {
      if (e.key === 'Enter') handleConfirm()
      if (e.key === 'Escape') handleCancel()
    }
  }

  $effect(() => {
    if (isOpen && dialog.type === 'prompt') inputEl?.focus()
  })
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isOpen}
  <Modal bind:open={isOpen} title={dialog.title} size="sm" zIndex={10200} onclose={handleCancel}>
    <p class="text-fg text-sm whitespace-pre-wrap wrap-break-word mb-4">{dialog.message}</p>

    {#if dialog.type === 'prompt'}
      <input
        bind:this={inputEl}
        type="text"
        bind:value={dialog.inputValue}
        class="w-full p-2 mb-4 border border-line rounded bg-chrome text-fg text-sm outline-none focus:border-brand"
      />
    {/if}

    <div class="flex justify-end gap-2">
      {#if dialog.type === 'confirm' || dialog.type === 'prompt'}
        <button
          type="button"
          class="inline-flex items-center gap-2 px-3 py-1 text-sm border border-line rounded cursor-pointer bg-chrome text-fg hover:brightness-110"
          onclick={handleCancel}
        >Cancel</button>
      {/if}
      <button
        type="button"
        class="inline-flex items-center gap-2 px-3 py-1 text-sm border border-brand rounded cursor-pointer bg-brand text-on-brand hover:brightness-110"
        onclick={handleConfirm}
      >OK</button>
    </div>
  </Modal>
{/if}
