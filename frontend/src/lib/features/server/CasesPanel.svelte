<script>
  import Button from '$components/ui/Button.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import PanelBody from '$components/patterns/PanelBody.svelte'
  import { cases } from '$stores/resources/cases.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(cases)
  import { CreateCase, DeleteCase } from '../../api/cases.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'
  import CaseDetail from './cases/CaseDetail.svelte'

  let { embedded = false, onclose } = $props()

  let list = $derived(cases?.data || [])
  let error = $derived(cases?.error || '')
  let selected = $state('')

  async function refresh() { await cases.refresh() }

  async function createCase() {
    const name = await dialog.prompt('Case name:', 'New Case', '')
    if (!name) return
    const description = await dialog.prompt('Short description (optional):', 'New Case', '')
    try {
      const c = await CreateCase(name, description || '')
      selected = c?.id || ''
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Create failed: '), 'Case')
    }
  }

  async function removeCase(c) {
    if (!(await dialog.confirm(`Delete case "${c.name}"?`, 'Confirm Delete'))) return
    try {
      await DeleteCase(c.id)
      if (selected === c.id) selected = ''
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Delete failed: '), 'Case')
    }
  }

  function counts(c) {
    return (c.agentIds?.length || 0) +
      (c.lootIds?.length || 0) +
      (c.credIds?.length || 0) +
      (c.hostIds?.length || 0) +
      (c.canaryIds?.length || 0)
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Cases'} icon={embedded ? '' : 'folder'}>
  <PanelBody error={error || null} empty={false}>
    <div class="flex h-full min-h-0">
      <aside class="w-64 shrink-0 border-r border-line flex flex-col">
        <div class="flex items-center gap-2 px-3 py-2 border-b border-line">
          <span class="text-xs font-semibold uppercase tracking-wider text-fg-muted">Cases</span>
          <div class="ml-auto flex gap-1">
            <Button color="dark" size="xs" onclick={refresh}>↻</Button>
            <Button color="primary" size="xs" icon="plus" onclick={createCase}>New</Button>
          </div>
        </div>
        <div class="flex-1 overflow-auto">
          {#each list as c}
            <div
              class="flex cursor-pointer items-center gap-2 px-3 py-2 text-xs hover:bg-row-hover {selected === c.id ? 'bg-row-selected' : ''}"
              onclick={() => (selected = c.id)}
              onkeydown={(e) => { if (e.key === 'Enter') selected = c.id }}
              role="button"
              tabindex="0"
            >
              <div class="flex-1 min-w-0">
                <div class="truncate font-semibold">{c.name}</div>
                {#if c.description}
                  <div class="truncate text-fg-muted">{c.description}</div>
                {/if}
              </div>
              <span class="text-fg-muted">{counts(c)}</span>
              <IconButton
                icon="x" label="Delete case" size="xs" color="red"
                onclick={(e) => { e.stopPropagation(); removeCase(c) }}
              />
            </div>
          {:else}
            <div class="p-3 text-xs text-fg-muted">No cases yet.</div>
          {/each}
        </div>
      </aside>

      <div class="flex-1 min-w-0">
        <CaseDetail caseID={selected} onchange={refresh} />
      </div>
    </div>
  </PanelBody>
</Panel>
