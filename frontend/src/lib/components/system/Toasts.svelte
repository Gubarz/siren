<script>
  import { toast } from '../../stores/ui/toast.svelte.js'
  import { config } from '../../stores/config.svelte.js'
  import Icon from '../ui/Icon.svelte'
  import { onSliverEvent } from '../../api/runtime.js'
  import { shouldShowNotification } from '../../utils/notifications.js'

  let items = $derived(toast.items)
  let notificationPrefs = $derived(config?.notifications)

  // The event-stream subscription is a live listener with a cleanup fn.
  // $effect returns it, so no onMount/onDestroy pair is needed.
  $effect(() => onSliverEvent((ev) => {
    if (!shouldShowNotification(notificationPrefs, ev.type)) return
    const t = describe(ev)
    if (t) toast.push(t)
  }))

  function describe(ev) {
    const host = ev.hostname ? ` ${ev.hostname}` : ''
    const who = ev.username ? ` (${ev.username})` : ''
    switch (ev.type) {
      case 'session-connected':
        return { variant: 'success', message: `Session opened:${host}${who}` }
      case 'session-disconnected':
        return { variant: 'warning', message: `Session closed:${host}` }
      case 'session-updated':
        return null
      case 'beacon-registered':
        return { variant: 'success', message: `Beacon registered:${host}${who}` }
      case 'job-started':
        return { variant: 'info', message: `Job started${ev.job ? ': ' + ev.job : ''}` }
      case 'job-stopped':
        return { variant: 'warning', message: `Job stopped${ev.job ? ': ' + ev.job : ''}` }
      default:
        return null
    }
  }
</script>

{#if items.length > 0}
  <div class="fixed bottom-10 right-4 flex flex-col gap-2 z-toast pointer-events-none">
    {#each items as t (t.id)}
      <div
        class="pointer-events-auto flex items-center gap-2 px-4 py-2 rounded-lg shadow-lg border-l-4 text-sm text-fg bg-surface-100 border-line animate-toast-in"
        class:border-success-500={t.variant === 'success'}
        class:border-warning-500={t.variant === 'warning'}
        class:border-danger-500={t.variant === 'danger'}
        class:border-primary-500={t.variant === 'info'}
      >
        {#if t.variant === 'success'}
          <Icon name="check" size={16} class="text-success-500 shrink-0" />
        {:else if t.variant === 'warning'}
          <Icon name="alert-triangle" size={16} class="text-warning-500 shrink-0" />
        {:else if t.variant === 'danger'}
          <Icon name="alert-circle" size={16} class="text-danger-500 shrink-0" />
        {:else}
          <Icon name="info" size={16} class="text-primary-500 shrink-0" />
        {/if}
        <span class="flex-1">{t.message}</span>
        <button
          type="button"
          class="text-fg-muted hover:text-fg shrink-0"
          onclick={() => toast.dismiss(t.id)}
          aria-label="Dismiss"
        >
          <Icon name="x" size={14} />
        </button>
      </div>
    {/each}
  </div>
{/if}
