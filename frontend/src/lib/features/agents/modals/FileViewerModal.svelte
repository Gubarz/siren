<script>
  import IconButton from '$components/ui/IconButton.svelte'
  import HexViewer from '$components/ui/HexViewer.svelte'
  import TextViewer from '$components/ui/TextViewer.svelte'

  let { viewerData, onclose } = $props()
</script>

{#if viewerData}
  <div class="fixed inset-0 bg-black/60 z-100 flex items-center justify-center" onclick={onclose} onkeydown={(e) => { if (e.key === 'Escape') onclose() }} role="dialog" tabindex="-1">
    <div class="w-vw-90 h-vh-85 bg-panel border border-line rounded-lg flex flex-col overflow-hidden shadow-2xl" role="none" onclick={(e) => e.stopPropagation()}>
      <div class="flex items-center px-4 py-2 bg-chrome border-b border-line">
        <span class="flex-1 font-mono text-sm font-semibold">{viewerData.filename}</span>
        <IconButton icon="x" label="Close" tooltip="Close" onclick={onclose} />
      </div>
      <div class="flex flex-1 min-h-0 flex-col">
        {#if viewerData.isBinary}
          <HexViewer data={viewerData.data} />
        {:else}
          <TextViewer data={viewerData.data} filename={viewerData.filename} />
        {/if}
      </div>
    </div>
  </div>
{/if}
