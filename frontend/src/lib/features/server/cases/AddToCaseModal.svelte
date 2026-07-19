<script>
  import Modal from '$components/patterns/Modal.svelte'
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import { AddToCase, CreateCase } from '../../../api/cases.js'
  import { cases } from '$stores/resources/cases.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(cases)
  import { errorMessage } from '../../../utils/errors.js'

  // Generic add-to-case picker. Every panel that wants a "Send to case…"
  // action opens this and passes {collection, itemID, label} — no per-
  // panel duplication.

  let {
    open = $bindable(false),
    collection = 'agent', // 'agent' | 'loot' | 'cred' | 'host' | 'canary'
    itemID = '',
    label = '',
    onclose = () => {},
  } = $props()

  let list = $derived(cases?.data || [])

  let selectedID = $state('')
  let newName = $state('')
  let submitting = $state(false)
  let error = $state('')

  $effect(() => { if (open) { error = ''; selectedID = list[0]?.id || ''; newName = '' } })

  async function addToExisting() {
    if (!selectedID) return
    submitting = true
    error = ''
    try {
      await AddToCase(selectedID, collection, itemID)
      open = false
    } catch (err) {
      error = errorMessage(err, 'Add failed: ')
    } finally { submitting = false }
  }

  async function createAndAdd() {
    if (!newName.trim()) return
    submitting = true
    error = ''
    try {
      const c = await CreateCase(newName.trim(), '')
      await AddToCase(c.id, collection, itemID)
      open = false
    } catch (err) {
      error = errorMessage(err, 'Create failed: ')
    } finally { submitting = false }
  }
</script>

<Modal bind:open title="Add to Case" size="md" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    Attach <span class="font-mono">{label || itemID}</span> ({collection}) to a case.
  </p>

  {#if list.length > 0}
    <div class="mb-4">
      <div class="text-xs uppercase tracking-wider text-fg-muted mb-2">Existing case</div>
      <div class="flex flex-col gap-1 max-h-40 overflow-auto mb-2">
        {#each list as c}
          <label class="flex items-center gap-2 px-2 py-1 rounded hover:bg-surface-200 text-sm">
            <input type="radio" name="case-choice" value={c.id} bind:group={selectedID} />
            <span class="flex-1 truncate font-mono">{c.name}</span>
            <span class="text-fg-muted text-xs">
              {(c.agentIds?.length || 0) + (c.lootIds?.length || 0) + (c.credIds?.length || 0) + (c.hostIds?.length || 0)} items
            </span>
          </label>
        {/each}
      </div>
      <Button color="primary" onclick={addToExisting} disabled={!selectedID || submitting}>
        Add to selected
      </Button>
    </div>
  {/if}

  <div>
    <div class="text-xs uppercase tracking-wider text-fg-muted mb-2">Or start a new case</div>
    <div class="flex gap-2">
      <div class="flex-1">
        <TextInput bind:value={newName} placeholder="Case name" />
      </div>
      <Button color="dark" onclick={createAndAdd} disabled={!newName.trim() || submitting}>Create + add</Button>
    </div>
  </div>

  {#if error}<div class="mt-3 text-sm text-danger-500">{error}</div>{/if}

  {#snippet footer()}
    <div class="flex justify-end">
      <Button color="dark" onclick={() => open = false} disabled={submitting}>Cancel</Button>
    </div>
  {/snippet}
</Modal>
