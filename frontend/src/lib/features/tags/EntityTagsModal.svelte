<script>
  import Modal from '../../components/patterns/Modal.svelte'
  import Button from '../../components/ui/Button.svelte'
  import TextInput from '../../components/ui/TextInput.svelte'
  import TagBadge from '../../components/ui/TagBadge.svelte'
  import {
    GetEntityTags,
    GetAgentTags,
    SetEntityTags,
    SetAgentTags,
    ListKnownTags,
    GetEntityColor,
    GetAllAgentColors,
    SetEntityColor,
    SetAgentColor,
  } from '../../api/tags.js'
  import { errorMessage } from '../../utils/errors.js'
  import { ROW_COLORS, colorTint } from '../../utils/agentColors.js'
  import { entityTags } from '../../stores/resources/entityTags.svelte.js'
  import { entityColors } from '../../stores/resources/entityColors.svelte.js'
  import { agentTags } from '../../stores/resources/agentTags.svelte.js'
  import { agentColors } from '../../stores/resources/agentColors.svelte.js'

  const TAG_PREFIXES = ['env:', 'role:', 'status:', 'prio:', 'owner:', 'group:']

  let {
    open = $bindable(false),
    onclose = () => {},
    entityType = '',
    entityID = '',
    entityLabel = '',
    entities = [],
  } = $props()

  let tags = $state([])
  let draft = $state('')
  let knownTags = $state([])
  let color = $state('')
  let initialTags = $state([])
  let targetTags = $state({})
  let colorTouched = $state(false)
  let submitting = $state(false)
  let error = $state('')

  let targets = $derived(
    Array.isArray(entities) && entities.length > 0
      ? entities.filter((e) => e?.type && e?.id).map((e) => ({
          type: e.type,
          id: e.id,
          label: e.label || e.id,
        }))
      : entityType && entityID
      ? [{ type: entityType, id: entityID, label: entityLabel || entityID }]
      : [],
  )
  let isBulk = $derived(targets.length > 1)

  $effect(() => {
    if (!open || targets.length === 0) return
    error = ''
    draft = ''
    colorTouched = false
    const loadTags = Promise.all(targets.map((target) => getTags(target)))
    const loadColors = Promise.all(targets.map((target) => getColor(target)))
    Promise.all([
      loadTags,
      loadColors,
      ListKnownTags(),
    ]).then(([tagLists, colors, all]) => {
      const keyedTags = {}
      for (const [index, target] of targets.entries()) {
        keyedTags[targetKey(target)] = tagLists[index] || []
      }
      targetTags = keyedTags
      tags = unionTags(tagLists)
      initialTags = tags
      color = sameValue(colors) || ''
      knownTags = all || []
    }).catch((err) => {
      error = errorMessage(err, 'Load failed: ')
    })
  })

  function targetKey(target) {
    return `${target.type}:${target.id}`
  }

  function unionTags(lists) {
    return [...new Set((lists || []).flat().map(normalize).filter(Boolean))].sort()
  }

  function sameValue(values) {
    const nonNull = (values || []).map((v) => v || '')
    if (nonNull.length === 0) return ''
    return nonNull.every((v) => v === nonNull[0]) ? nonNull[0] : ''
  }

  function getTags(target) {
    return target.type === 'agent' ? GetAgentTags(target.id) : GetEntityTags(target.type, target.id)
  }

  function getColor(target) {
    if (target.type === 'agent') {
      return GetAllAgentColors().then((colors) => colors?.[target.id] || '')
    }
    return GetEntityColor(target.type, target.id)
  }

  function setTags(target, nextTags) {
    return target.type === 'agent' ? SetAgentTags(target.id, nextTags) : SetEntityTags(target.type, target.id, nextTags)
  }

  function setColor(target, nextColor) {
    return target.type === 'agent' ? SetAgentColor(target.id, nextColor) : SetEntityColor(target.type, target.id, nextColor)
  }

  function normalize(tag) {
    return String(tag || '').trim().toLowerCase()
  }

  function addTag(raw) {
    const t = normalize(raw)
    if (!t || tags.includes(t)) return
    tags = [...tags, t].sort()
    draft = ''
  }

  function tagsWithDraft() {
    const t = normalize(draft)
    if (!t || tags.includes(t)) return tags
    return [...tags, t].sort()
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

  async function refreshTagResources() {
    await Promise.all([
      entityTags.refresh(),
      entityColors.refresh(),
      targets.some((target) => target.type === 'agent') ? agentTags.refresh() : Promise.resolve(),
      targets.some((target) => target.type === 'agent') ? agentColors.refresh() : Promise.resolve(),
    ])
  }

  function mergedBulkTags(target, nextTags) {
    const current = targetTags[targetKey(target)] || []
    const original = new Set(initialTags)
    const next = new Set(nextTags)
    const additions = nextTags.filter((tag) => !original.has(tag))
    const removals = initialTags.filter((tag) => !next.has(tag))
    return [...new Set([
      ...current.filter((tag) => !removals.includes(tag)),
      ...additions,
    ])].sort()
  }

  async function submit() {
    if (targets.length === 0) return
    submitting = true
    error = ''
    try {
      const nextTags = tagsWithDraft()
      tags = nextTags
      draft = ''
      await Promise.all(targets.map((target) => setTags(
        target,
        isBulk ? mergedBulkTags(target, nextTags) : nextTags,
      )))
      if (!isBulk || colorTouched) {
        await Promise.all(targets.map((target) => setColor(target, color)))
      }
      await refreshTagResources()
      open = false
    } catch (err) {
      error = errorMessage(err, 'Save failed: ')
    } finally {
      submitting = false
    }
  }
</script>

<Modal bind:open title={`Tags / Color - ${isBulk ? `${targets.length} items` : entityLabel || entityID}`} size="md" {onclose}>
  {#if targets.length > 0}
    <div class="space-y-4">
      <div>
        <label for="edit-entity-tags-input" class="block text-xs uppercase tracking-wider text-fg-muted mb-1">Tags</label>
        <div class="flex flex-wrap gap-1 mb-2 min-h-6">
          {#each tags as tag}
            <TagBadge {tag} removable onremove={() => removeTag(tag)} />
          {/each}
        </div>
        <TextInput
          id="edit-entity-tags-input"
          bind:value={draft}
          placeholder="Type a tag and press Enter..."
          onkeydown={onKeydown}
        />
        <div class="flex flex-wrap gap-1 mt-2 text-2xs text-fg-muted items-center">
          <span class="mr-1">Typed prefixes:</span>
          {#each TAG_PREFIXES as prefix}
            <Button
              color="dark"
              size="xs"
              class="font-mono text-4xs!"
              onclick={() => { draft = prefix }}
            >
              {prefix}
            </Button>
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

      <div>
        <div class="block text-xs uppercase tracking-wider text-fg-muted mb-1">Color</div>
        <div class="flex flex-wrap gap-2">
          {#each ROW_COLORS as name}
            <Button
              color="alternative"
              size="xs"
              title={name}
              aria-label={`Set ${name} color`}
              class="h-7! w-7! p-0! rounded border border-line transition-transform hover:scale-105 focus:outline-hidden focus:ring-2 focus:ring-brand/60 {color === name ? 'ring-2 ring-brand' : ''}"
              style={`background-color: ${colorTint(name)};`}
              onclick={() => { color = name; colorTouched = true }}
            />
          {/each}
          <Button
            color="dark"
            size="xs"
            class="h-7!"
            onclick={() => { color = ''; colorTouched = true }}
          >
            Clear
          </Button>
        </div>
      </div>

      {#if error}
        <div class="text-sm text-danger-500">{error}</div>
      {/if}
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <Button color="dark" onclick={() => open = false} disabled={submitting}>Cancel</Button>
      <Button color="primary" onclick={submit} disabled={submitting || targets.length === 0}>
        {submitting ? 'Saving...' : 'Save'}
      </Button>
    </div>
  {/snippet}
</Modal>
