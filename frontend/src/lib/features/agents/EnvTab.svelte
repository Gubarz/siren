<script>
  import { onMount } from 'svelte'
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import PanelBody from '$components/patterns/PanelBody.svelte'
  import { GetEnv, SetEnv, UnsetEnv } from '../../api/env.js'
  import { errorMessage } from '../../utils/errors.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'

  let { sessionID = '' } = $props()

  let variables = $state([])
  let loading = $state(false)
  let error = $state('')
  let saving = $state(null)
  let drafts = $state({})

  onMount(() => refresh())

  function loadDrafts(vars) {
    const next = {}
    for (const v of vars) next[v.Key] = v.Value || ''
    return next
  }

  async function refresh() {
    loading = true
    error = ''
    try {
      const resp = await GetEnv(sessionID)
      variables = resp?.Variables || []
      drafts = loadDrafts(variables)
    } catch (err) {
      error = errorMessage(err, 'Failed to load env vars: ')
    } finally {
      loading = false
    }
  }

  function isDirty(key) {
    const draft = drafts[key]
    const orig = (variables.find(v => v.Key === key)?.Value || '')
    return draft !== undefined && draft !== orig
  }

  async function addVar() {
    const key = await dialog.prompt('Variable name:', 'Add Env Var', '')
    if (!key) return
    const value = await dialog.prompt('Value for ' + key + ':', 'Add Env Var', '')
    if (value === null) return
    saving = key
    error = ''
    try {
      await SetEnv(sessionID, key, value)
      await refresh()
    } catch (err) {
      error = errorMessage(err, 'SetEnv failed: ')
    } finally {
      saving = null
    }
  }

  async function saveVar(v) {
    const value = drafts[v.Key]
    saving = v.Key
    error = ''
    try {
      await SetEnv(sessionID, v.Key, value)
      await refresh()
    } catch (err) {
      error = errorMessage(err, 'SetEnv failed: ')
    } finally {
      saving = null
    }
  }

  async function deleteVar(v) {
    if (!await dialog.confirm(`Unset ${v.Key}?`, 'Confirm')) return
    saving = v.Key
    error = ''
    try {
      await UnsetEnv(sessionID, v.Key)
      await refresh()
    } catch (err) {
      error = errorMessage(err, 'UnsetEnv failed: ')
    } finally {
      saving = null
    }
  }
</script>

<div class="flex flex-col h-full">
  <Toolbar class="justify-end gap-1">
    <Button color="primary" size="xs" icon="plus" onclick={addVar}>Add Env Var</Button>
    <Button color="dark" size="xs" onclick={refresh} disabled={loading}>Refresh</Button>
  </Toolbar>

  <PanelBody {error} empty={!loading && !error && variables.length === 0} emptyIcon="braces" emptyTitle="No env vars">
    <div class="p-2">
      <table class="w-full border-collapse text-xs">
        <thead>
          <tr class="border-b border-line bg-table-header text-left text-fg-muted">
            <th class="px-3 py-2 font-medium w-1/3">Key</th>
            <th class="px-3 py-2 font-medium">Value</th>
            <th class="px-3 py-2 text-right font-medium w-28">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each variables as v (v.Key)}
            {@const disabled = saving === v.Key}
            <tr class="border-b border-line hover:bg-row-hover">
              <td class="px-3 py-2 font-mono">{v.Key}</td>
              <td class="px-3 py-2">
                <TextInput
                  class="w-full font-mono"
                  placeholder="value"
                  value={drafts[v.Key] ?? ''}
                  oninput={(e) => drafts[v.Key] = e.target.value}
                  disabled={disabled}
                />
              </td>
              <td class="px-3 py-2 text-right">
                <div class="flex gap-1 justify-end">
                  <Button color="green" size="xs" onclick={() => saveVar(v)} disabled={disabled || !isDirty(v.Key)}>Save</Button>
                  <Button color="red" size="xs" onclick={() => deleteVar(v)} disabled={disabled}>Del</Button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </PanelBody>
</div>
