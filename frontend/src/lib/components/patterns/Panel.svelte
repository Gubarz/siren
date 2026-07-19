<script>
  import Modal from './Modal.svelte'
  import Icon from '../ui/Icon.svelte'

  let {
    title = '',
    icon = '',
    embedded = false,
    size = 'lg',
    class: className = '',
    onclose,
    children,
  } = $props()

  let modalOpen = $state(true)

  $effect.pre(() => {
    modalOpen = !embedded
  })

  function close() {
    if (!embedded) modalOpen = false
    onclose?.()
  }
</script>

{#if embedded}
  <div class="flex flex-col h-full overflow-hidden {className}">
    {#if title}
      <div class="flex items-center justify-between px-5 py-3 border-b border-line bg-surface-50 shrink-0">
        <div class="flex items-center gap-2">
          {#if icon}
            <Icon name={icon} size={18} class="text-brand" />
          {/if}
          <h2 class="text-base font-medium text-fg">{title}</h2>
        </div>
      </div>
    {/if}
    {#if children}
      {@render children()}
    {/if}
  </div>
{:else}
  <Modal bind:open={modalOpen} {title} {icon} {size} onclose={close}>
    {#if children}
      {@render children()}
    {/if}
  </Modal>
{/if}
