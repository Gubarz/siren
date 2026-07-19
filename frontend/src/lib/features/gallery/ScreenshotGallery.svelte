<script>
  import { onMount } from 'svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import GalleryCard from '$components/ui/GalleryCard.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import LoadingState from '$components/ui/LoadingState.svelte'
  import ErrorState from '$components/ui/ErrorState.svelte'
  import EmptyState from '$components/ui/EmptyState.svelte'
  import { GetScreenshotData, listLoot } from '../../api/server.js'
  import { errorMessage } from '../../utils/errors.js'

  let {
    onclose,
  } = $props()

  let screenshots = $state([])
  let loading = $state(true)
  let errorMsg = $state('')
  let selectedImage = $state(null)

  onMount(async () => {
    try {
      const allLoot = await listLoot()
      const imageLoot = allLoot.filter(l => {
        if (!l || !l.Name) return false
        const name = l.Name.toLowerCase()
        return name.endsWith('.png') || name.endsWith('.jpg') || name.endsWith('.jpeg')
      })
      for (const item of imageLoot) {
        try {
          const dataURI = await GetScreenshotData(item.ID)
          screenshots = [...screenshots, { ...item, dataURI }]
        } catch (e) {
          console.error('Failed to load image', item.ID, e)
        }
      }
    } catch (e) {
      console.error(e)
      errorMsg = errorMessage(e)
    } finally {
      loading = false
    }
  })

  function openLightbox(img) { selectedImage = img }
  function closeLightbox() { selectedImage = null }
</script>

<Panel {onclose} title="Screenshot Gallery" icon="images">
  {#if loading}
    <LoadingState description="Loading screenshots from Loot..." />
  {:else if errorMsg}
    <ErrorState error={errorMsg} title="Failed to load screenshots" />
  {:else if screenshots.length === 0}
    <EmptyState icon="images" title="No screenshots found" description="No screenshot files (.png, .jpg, .jpeg) were found in Loot." />
  {:else}
    <div class="grid grid-cards-auto-fill-200 gap-5 p-5">
      {#each screenshots as img}
        <GalleryCard src={img.dataURI} alt={img.Name} title={img.Name} onclick={() => openLightbox(img)} />
      {/each}
    </div>
  {/if}
</Panel>

{#if selectedImage}
  <div
    class="fixed inset-0 z-10000 bg-black/90 backdrop-blur-sm flex flex-col items-center justify-center"
    role="dialog"
    aria-modal="true"
    aria-label={selectedImage.Name}
  >
    <!-- eslint-disable-next-line local/no-raw-button -->
    <button
      type="button"
      class="absolute inset-0 bg-transparent border-0 p-0 cursor-default"
      aria-label="Close image"
      onclick={closeLightbox}
    ></button>
    <figure class="relative z-10 flex flex-col items-center max-w-pct-90 max-h-pct-90 m-0 pointer-events-none">
      <img src={selectedImage.dataURI} alt={selectedImage.Name} class="max-w-full max-h-vh-85 shadow-2xl border-2 border-line" />
      <figcaption class="mt-4 text-white text-base">{selectedImage.Name} - {selectedImage.Size} bytes</figcaption>
    </figure>
    <IconButton
      icon="x"
      label="Close image"
      size="md"
      color="alternative"
      onclick={closeLightbox}
      class="absolute! top-5 right-8 z-20"
    />
  </div>
{/if}
