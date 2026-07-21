<script>
  import Modal from '../../../components/patterns/Modal.svelte'
  import Button from '../../../components/ui/Button.svelte'
  import TextInput from '../../../components/ui/TextInput.svelte'
  import TextArea from '../../../components/ui/TextArea.svelte'
  import TagBadge from '../../../components/ui/TagBadge.svelte'
  import { GetAgentTags, SetAgentTags, ListKnownTags } from '../../../api/tags.js'
  import { GetAgentNotes, SaveAgentNote } from '../../../api/agents.js'
  import { errorMessage } from '../../../utils/errors.js'

  // Edit both operator tags AND a free-text note for one agent — same
  // backing file, same round-trip, so keep them together instead of two
  // separate modals.

  let {
    open = $bindable(false),
    onclose = () => {},
    agent = null,
  } = $props()

  let tags = $state([])
  let note = $state('')
  let draft = $state('')
  let knownTags = $state([])
  let submitting = $state(false)
  let error = $state('')

  $effect(() => {
    if (!open || !agent) return
    error = ''
    draft = ''
    Promise.all([
      GetAgentTags(agent.ID),
      GetAgentNotes().catch(() => ({})),
      ListKnownTags(),
    ]).then(([t, allNotes, all]) => {
      tags = t || []
      note = (allNotes && allNotes[agent.ID]) || ''
      knownTags = all || []
    }).catch((err) => {
      error = errorMessage(err, 'Load failed: ')
    })
  })

  function normalize(tag) {
    return String(tag || '').trim().toLowerCase()
  }

  function addTag(raw) {
    const t = normalize(raw)
    if (!t) return
    if (tags.includes(t)) return
    tags = [...tags, t].sort()
    draft = ''
  }

  function removeTag(tag) {
    tags = tags.filter((t) => t !== tag)
  }

  function onKeydown(e) {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addTag(draft)
    } else if (e.key === 'Backspace' && !draft && tags.length > 0) {
      tags = tags.slice(0, -1)
    }
  }

  let suggestions = $derived(() => {
    const q = normalize(draft)
    if (!q) return []
    return knownTags.filter((t) => t.includes(q) && !tags.includes(t)).slice(0, 6)
  })

  async function submit() {
    if (!agent) return
    submitting = true
    error = ''
    try {
      await SetAgentTags(agent.ID, tags)
      await SaveAgentNote(agent.ID, note)
      open = false
    } catch (err) {
      error = errorMessage(err, 'Save failed: ')
    } finally {
      submitting = false
    }
  }
</script>

<Modal bind:open title="Edit Tags & Notes" size="md" {onclose}>
  {#if agent}
    <p class="text-fg-muted text-sm mb-4">
      Tags for <span class="font-mono">{agent.Name || agent.ID}</span> —
      stored locally, not synced with the teamserver.
    </p>

    <div class="mb-3">
      <label for="edit-tags-input" class="block text-xs uppercase tracking-wider text-fg-muted mb-1">Tags</label>
      <div class="flex flex-wrap gap-1 mb-2 min-h-6">
        {#each tags as tag}
          <TagBadge {tag} removable onremove={() => removeTag(tag)} />
        {/each}
      </div>
      <TextInput
        id="edit-tags-input"
        bind:value={draft}
        placeholder="Type a tag (e.g. env:prod or role:dc) and press Enter…"
        onkeydown={onKeydown}
      />
      <div class="flex flex-wrap gap-1 mt-1.5 text-2xs text-fg-muted items-center">
        <span class="mr-1">Typed prefixes:</span>
        {#each ['env:', 'role:', 'status:', 'prio:', 'owner:', 'group:'] as prefix}
          <button
            type="button"
            class="px-1.5 py-0.5 rounded bg-surface-200 hover:bg-surface-300 font-mono text-fg-muted hover:text-fg text-4xs transition-colors"
            onclick={() => { draft = prefix; }}
          >
            {prefix}
          </button>
        {/each}
      </div>
      {#if suggestions().length > 0}
        <div class="flex flex-wrap gap-1 mt-2">
          {#each suggestions() as s}
            <Button color="dark" size="xs" onclick={() => addTag(s)}>{s}</Button>
          {/each}
        </div>
      {/if}
    </div>

    <div class="mb-3">
      <label for="edit-notes-input" class="block text-xs uppercase tracking-wider text-fg-muted mb-1">Note</label>
      <TextArea id="edit-notes-input" bind:value={note} rows={4} placeholder="Free-text note (persisted locally)…" />
    </div>

    {#if error}
      <div class="text-sm text-danger-500 mb-2">{error}</div>
    {/if}
  {/if}

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <Button color="dark" onclick={() => open = false} disabled={submitting}>Cancel</Button>
      <Button color="primary" onclick={submit} disabled={submitting || !agent}>
        {submitting ? 'Saving…' : 'Save'}
      </Button>
    </div>
  {/snippet}
</Modal>
