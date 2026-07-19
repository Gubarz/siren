<script>
  import { onMount } from 'svelte'
  import Button from '$components/ui/Button.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import PanelBody from '$components/patterns/PanelBody.svelte'
  import { GetWebsites, RemoveWebsite } from '../../api/websites.js'
  import { AddWebsiteContent } from '../../api/websites.js'
  import { OpenFileDialog } from '../../api/runtime.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'
  import WebsiteDetail from './websites/WebsiteDetail.svelte'

  // Websites panel — sites list on the left, content editor on the right.
  // The two share a refresh channel: any add/replace/remove on the detail
  // pane triggers a list reload so sizes / counts stay accurate.

  let { embedded = false, onclose } = $props()

  let sites = $state([])
  let selected = $state('')
  let loading = $state(false)
  let error = $state('')

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      const resp = await GetWebsites()
      sites = resp?.Websites || resp?.websites || []
      if (selected && !sites.find((s) => (s.Name || s.name) === selected)) selected = ''
    } catch (err) {
      error = errorMessage(err, 'Failed to load websites: ')
    } finally {
      loading = false
    }
  }

  async function createSite() {
    const name = await dialog.prompt('Site name:', 'New Website', '')
    if (!name) return
    const path = await dialog.prompt('URL path for the first file:', 'New Website', '/index.html')
    if (!path) return
    let localPath
    try {
      localPath = await OpenFileDialog('First file to serve')
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Pick failed: '), 'Website')
      return
    }
    if (!localPath) return
    try {
      await AddWebsiteContent({ name, path, contentType: '', localPath })
      selected = name
      await refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Create failed: '), 'Website')
    }
  }

  async function removeSite(site) {
    const name = site.Name || site.name
    if (!(await dialog.confirm(`Delete website "${name}" and every path under it?`, 'Confirm Delete'))) return
    try {
      await RemoveWebsite(name)
      if (selected === name) selected = ''
      await refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Delete failed: '), 'Website')
    }
  }

  function pathCount(site) {
    const c = site.Contents || site.contents || {}
    return Object.keys(c).length
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Websites'} icon={embedded ? '' : 'globe'}>
  <PanelBody
    error={error || null}
    empty={false}
  >
    <div class="flex h-full min-h-0">
      <aside class="w-64 shrink-0 border-r border-line flex flex-col">
        <div class="flex items-center gap-2 px-3 py-2 border-b border-line">
          <span class="text-xs font-semibold uppercase tracking-wider text-fg-muted">Sites</span>
          <div class="ml-auto flex gap-1">
            <Button color="dark" size="xs" onclick={refresh} disabled={loading}>↻</Button>
            <Button color="primary" size="xs" icon="plus" onclick={createSite}>New</Button>
          </div>
        </div>
        <div class="flex-1 overflow-auto">
          {#each sites as site}
            {@const name = site.Name || site.name}
            <div
              class="flex cursor-pointer items-center gap-2 px-3 py-2 text-xs hover:bg-row-hover {selected === name ? 'bg-row-selected' : ''}"
              onclick={() => (selected = name)}
              onkeydown={(e) => { if (e.key === 'Enter') selected = name }}
              role="button"
              tabindex="0"
            >
              <div class="flex-1 font-mono truncate" title={name}>{name}</div>
              <span class="text-fg-muted">{pathCount(site)}</span>
              <IconButton
                icon="x"
                label="Delete site"
                tooltip="Delete site"
                color="red"
                size="xs"
                onclick={(e) => { e.stopPropagation(); removeSite(site) }}
              />
            </div>
          {:else}
            <div class="p-3 text-xs text-fg-muted">No sites yet.</div>
          {/each}
        </div>
      </aside>

      <div class="flex-1 min-w-0">
        <WebsiteDetail name={selected} onchange={refresh} />
      </div>
    </div>
  </PanelBody>
</Panel>
