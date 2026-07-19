<script>
  import Icon from '$components/ui/Icon.svelte'
  import Button from '$components/ui/Button.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import { config, ZOOM_MIN, ZOOM_MAX, ZOOM_STEP } from '$stores/config.svelte.js'
  import { getStoredThemePreference, SYSTEM_THEME, theme } from '$stores/ui/theme.svelte.js'
  import NotificationSettings from './NotificationSettings.svelte'
  import HealthPanel from './HealthPanel.svelte'

  const themes = [
    { id: SYSTEM_THEME, name: 'System', bg: '#2a2a2a', accent: '#e0e0e0', text: '#eee' },
    { id: 'dark', name: 'Dark', bg: '#1a1a1a', accent: '#bd93f9', text: '#eee' },
    { id: 'light', name: 'Light', bg: '#f5f5f5', accent: '#2e7d32', text: '#333' },
    { id: 'hacker', name: 'Hacker', bg: '#000000', accent: '#00ff00', text: '#eee' },
    { id: 'cyberpunk', name: 'Cyberpunk', bg: '#0d0221', accent: '#00f0ff', text: '#eee' },
    { id: 'dracula', name: 'Dracula', bg: '#282a36', accent: '#bd93f9', text: '#eee' },
  ];

  let current = $state(getStoredThemePreference());
  let zoom = $derived(config?.zoom ?? 1.0);
  let confirmShellAdult = $derived(config?.confirmShellAdult !== false);

  function setTheme(id) {
    current = id;
    theme.set(id);
  }

  function zoomPercent(z) {
    return `${Math.round(z * 100)}%`;
  }
</script>

<div class="h-full overflow-auto p-6">
  <div class="grid gap-5 grid-cols-1 lg:grid-cols-2 max-w-7xl mx-auto">
    <div class="bg-panel border border-panel-border rounded-lg px-5 py-5 lg:col-span-2">
      <h3 class="m-0 mb-1 text-fg text-base">Theme</h3>
      <p class="text-xs mt-1 mb-4">Click a theme to apply it instantly. Your choice is remembered.</p>
      <div class="grid grid-cards-auto-fill-130 gap-3">
        {#each themes as t}
          <Button
            color="alternative"
            class={`relative !h-16 !w-full !flex !items-center !justify-start gap-2 !px-3 !border-2 !rounded-md !text-left !font-inherit transition-transform hover:-translate-y-1 ${current === t.id ? '!border-white' : '!border-transparent'}`}
            onclick={() => setTheme(t.id)}
            style="background:{t.bg}"
          >
            <div class="w-4 h-4 rounded-full" style="background:{t.accent}"></div>
            <span style="color:{t.text}">{t.name}</span>
            {#if current === t.id}<Icon name="check" class="absolute top-2 right-2" style="color:{t.accent}" />{/if}
          </Button>
        {/each}
      </div>
    </div>
    <div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
      <h3 class="m-0 mb-1 text-fg text-base">About</h3>
      <p class="text-xs mt-1 mb-4 text-fg-muted">A Sliver C2 graphical client. Built with Wails + Svelte.</p>
    </div>

    <div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
      <h3 class="m-0 mb-1 text-fg text-base">Zoom</h3>
      <p class="text-xs mt-1 mb-4">
        Scales the whole UI. Also bound to <code class="font-mono text-sm px-1 py-px rounded bg-chrome border border-line">Ctrl+=</code>, <code class="font-mono text-sm px-1 py-px rounded bg-chrome border border-line">Ctrl+-</code>, and <code class="font-mono text-sm px-1 py-px rounded bg-chrome border border-line">Ctrl+0</code>.
      </p>
      <div class="flex items-center gap-2">
        <Button color="dark" size="sm" icon="chevron-down" aria-label="Zoom out" onclick={() => config.zoomOut()} disabled={zoom <= ZOOM_MIN + 0.001} />
        <input
          type="range"
          min={ZOOM_MIN}
          max={ZOOM_MAX}
          step={ZOOM_STEP}
          value={zoom}
          oninput={(e) => config.setZoom(Number(e.currentTarget.value))}
          class="flex-1"
        />
        <Button color="dark" size="sm" icon="chevron-up" aria-label="Zoom in" onclick={() => config.zoomIn()} disabled={zoom >= ZOOM_MAX - 0.001} />
        <span class="min-w-11 text-right tabular-nums text-fg-muted">{zoomPercent(zoom)}</span>
        <Button color="dark" size="sm" onclick={() => config.zoomReset()} disabled={Math.abs(zoom - 1.0) < 0.001}>
          Reset
        </Button>
      </div>
    </div>

    <HealthPanel />

    <div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
      <h3 class="m-0 mb-1 text-fg text-base">Shell Safety</h3>
      <p class="text-xs mt-1 mb-4 text-fg-muted">Confirmation before starting an interactive shell.</p>
      <label class="flex items-center gap-2 text-sm">
        <Checkbox
          checked={confirmShellAdult}
          onchange={() => config.set('confirmShellAdult', !confirmShellAdult)}
        />
        Ask before starting a shell
      </label>
    </div>

    <NotificationSettings />
  </div>
</div>
