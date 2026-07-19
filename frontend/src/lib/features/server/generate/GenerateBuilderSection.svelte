<script>
  import { onMount } from 'svelte'
  import Button from '$components/ui/Button.svelte'
  import Select from '$components/ui/Select.svelte'
  import CollapsibleGroup from '$components/forms/CollapsibleGroup.svelte'
  import { builders } from '$stores/resources/builders.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(builders)

  let {
    buildTarget = $bindable('server'),
    externalBuild = null,
    externalStatus = '',
    externalBusy = false,
    onrefreshconfig,
    onsavebinary,
  } = $props()

  let binaryInput
  let builderList = $derived(builders.data || [])
  let buildOptions = $derived([
    { value: 'server', label: 'Teamserver' },
    ...builderList.map((builder) => {
      const name = builder.Name || builder.name || builder.ID || builder.id
      const osArch = `${builder.GOOS || builder.goos || '-'}/${builder.GOARCH || builder.goarch || '-'}`
      return { value: name, label: `${name} (${osArch})` }
    }),
  ])
  let externalBuildID = $derived(externalBuild?.Build?.ID || externalBuild?.build?.id || '')
  let selectedExternal = $derived(buildTarget !== 'server')

  onMount(() => {
    builders.refresh()
  })

  function pickBinary() {
    binaryInput?.click()
  }

  async function handleBinary(event) {
    const file = event.currentTarget.files?.[0]
    event.currentTarget.value = ''
    if (!file) return
    const data = Array.from(new Uint8Array(await file.arrayBuffer()))
    onsavebinary?.({ name: file.name, data })
  }
</script>

<CollapsibleGroup title="Build target" open={false}>
  <div class="grid gap-2">
    <div class="flex items-end gap-2">
      <div class="min-w-64 flex-1">
        <Select bind:value={buildTarget} options={buildOptions} size="sm" aria-label="Build target" />
      </div>
      <Button color="dark" size="sm" onclick={() => builders.refresh()} disabled={builders.loading}>
        {builders.loading ? 'Refreshing...' : 'Refresh'}
      </Button>
    </div>

    {#if selectedExternal}
      <div class="rounded border border-line bg-canvas p-2 text-xs">
        <div class="flex flex-wrap items-center gap-2">
          <span class="font-mono text-fg-muted">Builder: {buildTarget}</span>
          {#if externalBuildID}
            <span class="font-mono text-fg-muted">Build: {externalBuildID}</span>
          {/if}
          <Button color="dark" size="xs" onclick={onrefreshconfig} disabled={externalBusy || !externalBuildID}>
            Refresh config
          </Button>
          <Button color="primary" size="xs" icon="upload" onclick={pickBinary} disabled={externalBusy || !externalBuildID}>
            Attach binary
          </Button>
        </div>
        {#if externalStatus}
          <div class="mt-2 whitespace-pre-wrap text-fg-muted">{externalStatus}</div>
        {/if}
      </div>
    {/if}
  </div>
  <input bind:this={binaryInput} class="hidden" type="file" onchange={handleBinary} />
</CollapsibleGroup>
