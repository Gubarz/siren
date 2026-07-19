<script>
  import Button from '$components/ui/Button.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import PanelBody from '$components/patterns/PanelBody.svelte'
  import TextArea from '$components/ui/TextArea.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { httpC2Profiles } from '$stores/resources/httpC2Profiles.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(httpC2Profiles)
  import { GetHTTPC2ProfileByName, SaveHTTPC2ProfileJSON, profileName } from '../../api/httpc2.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'

  let { embedded = false, onclose } = $props()

  let selected = $state('')
  let profileText = $state('')
  let detailError = $state('')
  let detailLoading = $state(false)
  let saving = $state(false)
  let savedText = $state('')

  let profiles = $derived(httpC2Profiles.data || [])
  let selectedProfile = $derived(profiles.find((profile) => profileName(profile) === selected))
  let isDirty = $derived(profileText !== savedText)

  $effect(() => {
    if (!selected && profiles.length > 0) {
      selected = profileName(profiles[0])
    }
  })

  $effect(() => {
    if (selected) loadProfile(selected)
  })

  async function loadProfile(name) {
    const requestName = name
    detailLoading = true
    detailError = ''
    try {
      const profile = await GetHTTPC2ProfileByName(requestName)
      if (selected !== requestName) return
      const text = JSON.stringify(profile || {}, null, 2)
      profileText = text
      savedText = text
    } catch (err) {
      if (selected === requestName) detailError = errorMessage(err, 'Failed to load HTTP C2 profile: ')
    } finally {
      if (selected === requestName) detailLoading = false
    }
  }

  async function saveProfile() {
    if (!profileText.trim()) return
    saving = true
    detailError = ''
    try {
      await SaveHTTPC2ProfileJSON(profileText, true)
      const saved = JSON.parse(profileText)
      selected = profileName(saved) || selected
      savedText = profileText
      await httpC2Profiles.refresh()
    } catch (err) {
      detailError = errorMessage(err, 'Save failed: ')
    } finally {
      saving = false
    }
  }

  async function duplicateProfile() {
    if (!profileText.trim()) return
    const name = await dialog.prompt('New HTTP C2 profile name:', 'Duplicate Profile', `${selected}-copy`)
    if (!name) return
    try {
      const copy = JSON.parse(profileText)
      copy.ID = ''
      copy.Created = 0
      copy.Name = name
      profileText = JSON.stringify(copy, null, 2)
    } catch (err) {
      detailError = errorMessage(err, 'Duplicate failed: ')
    }
  }

  function pathCount(profile) {
    return (profile?.ImplantConfig?.PathSegments || profile?.implantConfig?.pathSegments || []).length
  }

  function headerCount(profile) {
    const implant = profile?.ImplantConfig?.Headers || profile?.implantConfig?.headers || []
    const server = profile?.ServerConfig?.Headers || profile?.serverConfig?.headers || []
    return implant.length + server.length
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'HTTP C2 Profiles'} icon={embedded ? '' : 'radio'}>
  <Toolbar class="justify-end">
    <Button color="dark" size="sm" onclick={() => httpC2Profiles.refresh()} disabled={httpC2Profiles.loading}>Refresh</Button>
    <Button color="dark" size="sm" onclick={() => selected && loadProfile(selected)} disabled={!selected || detailLoading}>Reload</Button>
    <Button color="dark" size="sm" onclick={duplicateProfile} disabled={!profileText}>Duplicate</Button>
    <Button color="primary" size="sm" onclick={saveProfile} disabled={!profileText || saving}>
      {saving ? 'Saving...' : 'Save'}
    </Button>
  </Toolbar>

  <PanelBody
    error={httpC2Profiles.error && !httpC2Profiles.loading ? httpC2Profiles.error : null}
    empty={!httpC2Profiles.loading && !httpC2Profiles.error && profiles.length === 0}
    emptyIcon="radio"
    emptyTitle="No HTTP C2 profiles"
  >
    <div class="flex h-full min-h-0">
      <aside class="w-72 shrink-0 border-r border-line">
        <div class="border-b border-line px-3 py-2 text-xs font-semibold uppercase tracking-wider text-fg-muted">
          Profiles
        </div>
        <div class="overflow-auto">
          {#each profiles as profile}
            {@const name = profileName(profile)}
            <Button
              color="alternative"
              size="xs"
              fullWidth
              class={`!justify-start !rounded-none !border-0 !border-b !border-line !bg-transparent !px-3 !py-2 !text-left !text-xs !text-fg hover:!bg-row-hover ${selected === name ? '!bg-row-selected' : ''}`}
              onclick={() => (selected = name)}
            >
              <span class="min-w-0 flex-1 truncate font-mono">{name}</span>
              <span class="text-fg-muted">{pathCount(profile)} paths</span>
            </Button>
          {/each}
        </div>
      </aside>

      <section class="flex min-w-0 flex-1 flex-col">
        <div class="flex items-center gap-3 border-b border-line px-3 py-2 text-xs">
          <span class="font-mono font-semibold">{selected || 'No profile selected'}</span>
          {#if selectedProfile}
            <span class="text-fg-muted">{headerCount(selectedProfile)} headers</span>
          {/if}
          {#if detailLoading}
            <span class="text-fg-muted">Loading...</span>
          {:else if isDirty}
            <span class="text-warning-500">Unsaved changes</span>
          {/if}
        </div>

        {#if detailError}
          <div class="border-b border-line px-3 py-2 text-xs text-danger-500">{detailError}</div>
        {/if}

        <div class="flex min-h-0 flex-1 p-3">
          <TextArea
            bind:value={profileText}
            disabled={!selected || detailLoading}
            rows={28}
            spellcheck="false"
            class="min-h-full w-full font-mono text-xs leading-5"
            divClass="flex min-h-0 flex-1"
          />
        </div>
      </section>
    </div>
  </PanelBody>
</Panel>
