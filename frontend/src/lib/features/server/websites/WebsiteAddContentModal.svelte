<script>
  import Modal from '$components/patterns/Modal.svelte'
  import Button from '$components/ui/Button.svelte'
  import PresetPicker from '$components/forms/PresetPicker.svelte'
  import TextField from '$components/forms/TextField.svelte'
  import { OpenFileDialog } from '../../../api/runtime.js'
  import { AddWebsiteContent, UpdateWebsiteContent } from '../../../api/websites.js'
  import { errorMessage } from '../../../utils/errors.js'

  // Handles both "add a path" and "replace an existing path" — the server
  // exposes two distinct RPCs but the form is identical, so we thread a
  // `mode` prop through and pick the right call on submit.

  let {
    open = $bindable(false),
    mode = 'add', // 'add' | 'replace'
    siteName = '',
    initialPath = '',
    initialContentType = '',
    onclose = () => {},
    onsuccess = () => {},
  } = $props()

  let urlPath = $state('')
  let contentType = $state('')
  let localPath = $state('')
  let submitting = $state(false)
  let error = $state('')

  $effect(() => {
    if (open) {
      urlPath = initialPath || ''
      contentType = initialContentType || ''
      localPath = ''
      error = ''
    }
  })

  async function pickFile() {
    try {
      const path = await OpenFileDialog(mode === 'add' ? 'Add content' : 'Replace content')
      if (path) localPath = path
    } catch (err) {
      error = errorMessage(err, 'File dialog failed: ')
    }
  }

  async function submit() {
    if (!siteName || !urlPath || !localPath) {
      error = 'Site, path, and a local file are all required.'
      return
    }
    submitting = true
    error = ''
    try {
      const req = { name: siteName, path: urlPath, contentType, localPath }
      if (mode === 'replace') await UpdateWebsiteContent(req)
      else await AddWebsiteContent(req)
      open = false
      onsuccess()
    } catch (err) {
      error = errorMessage(err, `${mode === 'replace' ? 'Update' : 'Add'} failed: `)
    } finally {
      submitting = false
    }
  }

  function applyPreset(values) {
    if (values.path != null && mode === 'add') urlPath = values.path
    if (values.contentType != null) contentType = values.contentType
  }
</script>

<Modal bind:open title={mode === 'replace' ? 'Replace Content' : 'Add Content'} size="xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    {#if mode === 'replace'}
      Overwrite <code>{urlPath}</code> on <code>{siteName}</code> with a new file.
    {:else}
      Register a file to be served under a URL path on <code>{siteName || '(new site)'}</code>.
      Creates the site if it doesn't exist.
    {/if}
  </p>

  <div class="grid gap-3">
    <TextField
      bind:value={urlPath}
      label="URL path"
      placeholder="/index.html"
      disabled={mode === 'replace'}
    />
    <TextField
      bind:value={contentType}
      label="Content-Type (blank auto-detects from extension)"
      placeholder="text/html; charset=utf-8"
    />
    <div class="flex items-end gap-2">
      <div class="flex-1">
        <TextField
          bind:value={localPath}
          label="Local file"
          placeholder="/tmp/index.html"
        />
      </div>
      <Button color="dark" size="sm" onclick={pickFile}>Browse…</Button>
    </div>
  </div>

  {#if error}
    <div class="mt-3 text-sm text-danger-500">{error}</div>
  {/if}

  {#snippet footer()}
    <div class="flex items-center justify-between gap-2">
      <PresetPicker
        commandPath="websites/content"
        currentValues={{ path: urlPath, contentType }}
        onapply={applyPreset}
      />
      <div class="flex justify-end gap-2">
        <Button color="dark" onclick={() => open = false} disabled={submitting}>Cancel</Button>
        <Button color="primary" onclick={submit} disabled={submitting}>
          {submitting ? 'Uploading…' : mode === 'replace' ? 'Replace' : 'Add'}
        </Button>
      </div>
    </div>
  {/snippet}
</Modal>
