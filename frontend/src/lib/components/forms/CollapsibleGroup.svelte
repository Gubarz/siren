<script>
  import Button from '../ui/Button.svelte'
  import Icon from '../ui/Icon.svelte'

  let { title = '', open = false, children } = $props()
</script>

<div class="mb-3">
  <Button
    color="alternative"
    class="!flex !items-center !justify-start !w-full gap-1 !px-2 !py-2 !bg-chrome !border !border-line !rounded !text-fg text-base font-semibold text-left select-none hover:!bg-brand-muted"
    onclick={() => open = !open}
  >
    <Icon name="chevron-right" size={10} class={`transition-transform duration-150 ${open ? 'rotate-90' : ''}`} />
    {title}
  </Button>
  {#if open}
    <div class="flex flex-col gap-2 px-2 pt-2">
      {#if children}
        {@render children()}
      {/if}
    </div>
  {/if}
</div>
