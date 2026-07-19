<script>
  import Modal from '$components/patterns/Modal.svelte'

  let { open = $bindable(false), onclose = () => {} } = $props()
</script>

<Modal bind:open title="Docs" size="5xl" {onclose}>
  <iframe
    title="sliver-gui docs"
    src="/docs/index.html"
    class="h-vh-75 w-full rounded border border-line bg-canvas"
  ></iframe>
</Modal>
