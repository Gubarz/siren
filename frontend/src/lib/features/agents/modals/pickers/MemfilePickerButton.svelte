<script>
  import Button from '$components/ui/Button.svelte'
  import { listMemfiles } from '../../../../api/memfiles.js'
  import { contextMenu } from '$stores/ui/contextMenu.svelte.js'

  // Small "Use memfile…" trigger for the Execute/Sideload/SpawnDll
  // modals — clicking pulls the current session's memfile list and
  // opens a context menu so the operator can pick one. Onpicks with
  // the full remote path so the caller can drop it straight into the
  // path input.

  let { sessionID = '', onpick = () => {}, label = 'Memfile…' } = $props()

  let loading = $state(false)

  async function open(event) {
    if (!sessionID) return
    loading = true
    try {
      const { files, path } = await listMemfiles(sessionID)
      const list = Array.isArray(files) ? files : []
      if (list.length === 0) {
        contextMenu.open({
          x: event.clientX, y: event.clientY,
          sections: [{ items: [{ label: 'No memfiles registered', disabled: true }] }],
        })
        return
      }
      const items = list.map((f) => {
        const name = f.Name || f.name || f.FullPath || f.fullPath || String(f)
        const full = f.FullPath || f.fullPath || `${path.replace(/\/$/, '')}/${name}`
        return { icon: 'file', label: name, description: full, on: () => onpick(full) }
      })
      contextMenu.open({
        x: event.clientX, y: event.clientY,
        sections: [{ title: 'Memfiles', items }],
      })
    } catch {}
    finally { loading = false }
  }
</script>

<Button color="dark" size="xs" onclick={open} disabled={!sessionID || loading}>
  {loading ? 'Loading…' : label}
</Button>
