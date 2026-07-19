<script>
  import Checkbox from '$components/ui/Checkbox.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import { config, DEFAULT_NOTIFICATION_TYPES } from '$stores/config.svelte.js'

  // Notification prefs: master switch, per-event-type toggles, DND window.
  // Everything writes back through `config.set('notifications', …)` so the
  // localStorage persistence layer picks it up unchanged.

  // prefs is $state (not $derived) because the UI mutates it locally
  // before persisting. The $effect syncs the store → local copy so if
  // config changes elsewhere the form reflects it.
  let prefs = $state({
    enabled: true,
    types: { ...DEFAULT_NOTIFICATION_TYPES },
    dnd: { enabled: false, start: '22:00', end: '08:00' },
  })
  $effect(() => {
    if (config?.notifications) prefs = structuredClone(config.notifications)
  })

  const TYPE_LABELS = [
    { id: 'session-connected', label: 'Session opened' },
    { id: 'session-disconnected', label: 'Session closed' },
    { id: 'beacon-registered', label: 'Beacon registered' },
    { id: 'job-started', label: 'Job started' },
    { id: 'job-stopped', label: 'Job stopped' },
  ]

  function persist() {
    config.set('notifications', prefs)
  }

  function toggleType(id) {
    prefs.types = { ...prefs.types, [id]: !(prefs.types[id] !== false) }
    persist()
  }

  function toggleEnabled() {
    prefs.enabled = !prefs.enabled
    persist()
  }

  function toggleDnd() {
    prefs.dnd = { ...prefs.dnd, enabled: !prefs.dnd.enabled }
    persist()
  }

  function setDndTime(field, value) {
    prefs.dnd = { ...prefs.dnd, [field]: value }
    persist()
  }
</script>

<div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
  <h3 class="m-0 mb-1 text-fg text-base">Notifications</h3>
  <p class="text-xs mt-1 mb-4 text-fg-muted">
    Which sliver events surface as toasts, and when to stay quiet.
  </p>

  <label class="flex items-center gap-2 text-sm mb-4">
    <Checkbox checked={prefs.enabled} onchange={toggleEnabled} />
    Enable notifications
  </label>

  <div class="mb-4 pl-4 border-l-2 border-line" class:opacity-50={!prefs.enabled}>
    <div class="text-xs uppercase tracking-wider text-fg-muted mb-2">Event types</div>
    <div class="flex flex-col gap-2">
      {#each TYPE_LABELS as t}
        <label class="flex items-center gap-2 text-sm">
          <Checkbox
            checked={prefs.types[t.id] !== false}
            onchange={() => toggleType(t.id)}
            disabled={!prefs.enabled}
          />
          {t.label}
        </label>
      {/each}
    </div>
  </div>

  <div class="pl-4 border-l-2 border-line" class:opacity-50={!prefs.enabled}>
    <label class="flex items-center gap-2 text-sm mb-2">
      <Checkbox
        checked={prefs.dnd.enabled}
        onchange={toggleDnd}
        disabled={!prefs.enabled}
      />
      Do Not Disturb window
    </label>
    <div class="flex items-center gap-2 pl-6 text-sm" class:opacity-50={!prefs.dnd.enabled}>
      <span class="text-fg-muted">From</span>
      <TextInput
        value={prefs.dnd.start}
        oninput={(e) => setDndTime('start', e.target.value)}
        placeholder="22:00"
        disabled={!prefs.enabled || !prefs.dnd.enabled}
        class="!w-20"
      />
      <span class="text-fg-muted">to</span>
      <TextInput
        value={prefs.dnd.end}
        oninput={(e) => setDndTime('end', e.target.value)}
        placeholder="08:00"
        disabled={!prefs.enabled || !prefs.dnd.enabled}
        class="!w-20"
      />
      <span class="text-fg-muted text-xs">24h clock — wraps midnight if start &gt; end</span>
    </div>
  </div>
</div>
