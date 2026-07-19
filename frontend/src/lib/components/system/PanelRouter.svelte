<script>
  import { overlays } from '$stores/ui/overlays.svelte.js'
  import ListenersPanel from '$features/server/ListenersPanel.svelte'
  import LootPanel from '$features/server/LootPanel.svelte'
  import CredentialsPanel from '$features/server/CredentialsPanel.svelte'
  import OperatorsPanel from '$features/server/OperatorsPanel.svelte'
  import PivotsPanel from '$features/server/PivotsPanel.svelte'
  import ArmoryPanel from '$features/server/ArmoryPanel.svelte'
  import GenerateModal from '$features/server/GenerateModal.svelte'
  import ScreenshotGallery from '$features/gallery/ScreenshotGallery.svelte'

  let current = $derived(overlays)
</script>

{#key current?.instance}
  {#if current?.name === 'listeners'}
    <ListenersPanel onclose={() => overlays.close()} />
  {/if}
  {#if current?.name === 'loot'}
    <LootPanel onclose={() => overlays.close()} />
  {/if}
  {#if current?.name === 'credentials'}
    <CredentialsPanel onclose={() => overlays.close()} />
  {/if}
  {#if current?.name === 'operators'}
    <OperatorsPanel onclose={() => overlays.close()} />
  {/if}
  {#if current?.name === 'pivots'}
    <PivotsPanel onclose={() => overlays.close()} />
  {/if}
  {#if current?.name === 'armory'}
    <ArmoryPanel onclose={() => overlays.close()} />
  {/if}
  {#if current?.name === 'generate'}
    <GenerateModal open={true} onclose={() => overlays.close()} {...current.props} />
  {/if}
  {#if current?.name === 'gallery'}
    <ScreenshotGallery onclose={() => overlays.close()} />
  {/if}
{/key}
