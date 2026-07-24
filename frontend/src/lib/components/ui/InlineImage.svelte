<script>
  let { src = '', alt = '', maxHeight = '16rem' } = $props()

  let lightbox = $state(false)

  function openLightbox() {
    if (!src) return
    lightbox = true
  }

  function closeLightbox() {
    lightbox = false
  }
</script>

{#if src}
  <button
    type="button"
    class="border-0 bg-transparent p-0 cursor-pointer w-full"
    onclick={openLightbox}
  >
    <img
      {src}
      {alt}
      class="w-full object-contain border border-line shadow-sm"
      style="max-height: {maxHeight}"
    />
  </button>
{:else}
  <div class="flex items-center justify-center border border-line bg-chrome text-fg-muted p-8 text-sm" style="height: {maxHeight}">
    No image
  </div>
{/if}

{#if lightbox}
  <div
    class="fixed inset-0 z-10000 bg-black/90 backdrop-blur-sm flex flex-col items-center justify-center"
    role="dialog"
    aria-modal="true"
    aria-label={alt || 'Image'}
  >
    <button
      type="button"
      class="absolute inset-0 bg-transparent border-0 p-0 cursor-default"
      aria-label="Close image"
      onclick={closeLightbox}
    ></button>
    <figure class="relative z-10 flex flex-col items-center max-w-pct-90 max-h-pct-90 m-0">
      <img src={src} alt={alt} class="max-w-full max-h-vh-85 shadow-2xl border-2 border-line object-contain" />
      {#if alt}
        <figcaption class="mt-4 text-white text-base">{alt}</figcaption>
      {/if}
    </figure>
    <button
      type="button"
      class="absolute top-5 right-8 z-20 text-white bg-black/50 hover:bg-black/70 rounded p-2 border-0 cursor-pointer"
      aria-label="Close"
      onclick={closeLightbox}
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
    </button>
  </div>
{/if}
