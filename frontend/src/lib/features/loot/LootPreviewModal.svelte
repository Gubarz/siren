<script>
  import Icon from '$components/ui/Icon.svelte'
  import Modal from '$components/patterns/Modal.svelte'
  import Tabs from '$components/patterns/Tabs.svelte'
  import { GetLootContent } from '../../api/server.js'
  import { detectMime, fileTypeLabel } from './magic.js'

  let { lootID, lootName, lootFileType, open = $bindable(false) } = $props()

  let activeTab = $state('text')
  let loading = $state(true)
  let error = $state('')
  let rawBytes = $state(null)
  let mimeInfo = $state(null)

  let tabs = [
    { id: 'text', label: 'Text', icon: 'align-left' },
    { id: 'hex', label: 'Hex', icon: 'code' },
    { id: 'strings', label: 'Strings', icon: 'file-text' },
  ]

  let textContent = $derived.by(() => {
    if (!rawBytes) return ''
    return new TextDecoder('utf-8', { fatal: false }).decode(rawBytes)
  })

  let hexRows = $derived.by(() => {
    if (!rawBytes) return []
    const rows = []
    for (let i = 0; i < rawBytes.length; i += 16) {
      const chunk = rawBytes.slice(i, i + 16)
      const hex = Array.from(chunk, (b) => b.toString(16).padStart(2, '0').toUpperCase()).join(' ')
      const ascii = Array.from(chunk, (b) => (b >= 0x20 && b <= 0x7E ? String.fromCharCode(b) : '.')).join('')
      rows.push({ offset: i.toString(16).padStart(8, '0'), hex: hex.padEnd(47, ' '), ascii })
    }
    return rows
  })

  let extractedStrings = $derived.by(() => {
    if (!rawBytes) return []
    const results = []
    let run = ''
    for (let i = 0; i < rawBytes.length; i++) {
      const b = rawBytes[i]
      if (b >= 0x20 && b <= 0x7E) {
        run += String.fromCharCode(b)
      } else {
        if (run.length >= 4) results.push(run)
        run = ''
      }
    }
    if (run.length >= 4) results.push(run)
    return results
  })

  let displayName = $derived(lootName ?? lootID?.substring(0, 8) ?? 'Loot')
  let titleText = $derived('Preview: ' + displayName)

  async function loadContent() {
    if (!lootID) return
    loading = true
    error = ''
    try {
      const base64 = await GetLootContent(lootID)
      if (!base64) { error = 'Empty response'; return }
      const binary = atob(base64)
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
      rawBytes = bytes
      mimeInfo = detectMime(bytes)
    } catch (e) {
      error = e?.message ?? String(e)
    } finally {
      loading = false
    }
  }

  $effect(() => {
    if (open && lootID) loadContent()
  })
</script>

<Modal bind:open title={titleText} icon="eye" size="lg">
  {#if loading}
    <div class="flex items-center justify-center py-12 gap-2">
      <Icon name="loader" size={24} class="animate-spin text-brand" />
      <span class="text-sm text-fg-muted">Loading preview...</span>
    </div>
  {:else if error}
    <div class="p-4 text-sm text-red-600 bg-red-50 border border-red-200 rounded-md">{error}</div>
  {:else}
    <div class="flex items-center gap-3 pb-2 text-xs text-fg-muted">
      <span>Server: {fileTypeLabel(lootFileType)}</span>
      {#if mimeInfo}
        <span class="px-2 py-1 rounded bg-brand/10 text-brand font-medium">{mimeInfo.label}</span>
      {:else}
        <span>No magic match</span>
      {/if}
      <span>{rawBytes.length.toLocaleString()} bytes</span>
    </div>

    <Tabs {tabs} bind:active={activeTab} />

    <div class="mt-2 max-h-96 overflow-auto">
      {#if activeTab === 'text'}
        <pre class="text-xs font-mono whitespace-pre-wrap break-all p-2 bg-surface rounded">{textContent}</pre>
      {:else if activeTab === 'hex'}
        <pre class="text-xs font-mono leading-relaxed p-2 bg-surface rounded whitespace-pre">
          {#each hexRows as row}
            <span class="text-fg-muted">{row.offset}</span>  {row.hex}  |{row.ascii}|
          {/each}
        </pre>
      {:else}
        <div class="p-2 bg-surface rounded">
          {#if extractedStrings.length === 0}
            <span class="text-xs text-fg-muted">No printable strings found</span>
          {:else}
            {#each extractedStrings as str}
              <div class="text-xs font-mono py-1 border-b border-line/50 last:border-0">{str}</div>
            {/each}
          {/if}
        </div>
      {/if}
    </div>
  {/if}
</Modal>
