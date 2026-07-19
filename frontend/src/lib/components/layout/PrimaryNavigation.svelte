<script>
  import Icon from '../ui/Icon.svelte';

  let { active = 'agents', onSelect = () => {} } = $props();

  const tabs = [
    { id: 'agents', label: 'Agents', icon: 'bullseye' },
    { id: 'automation', label: 'Automation', icon: 'bolt' },
    { id: 'server', label: 'Server', icon: 'server' },
    { id: 'settings', label: 'Settings', icon: 'cog' },
  ];
</script>

<nav aria-label="Primary" class="flex h-full items-stretch">
  {#each tabs as tab}
    <button
      type="button"
      style="--wails-draggable:no-drag"
      aria-current={active === tab.id ? 'page' : undefined}
      class="px-3 h-full text-sm font-medium whitespace-nowrap border-b-2 transition-colors cursor-pointer {active === tab.id ? 'text-brand border-brand' : 'text-fg-muted border-transparent hover:text-fg hover:border-fg-muted'}"
      onclick={() => onSelect(tab.id)}
    >
      <span class="flex items-center gap-2 justify-center">
        <Icon name={tab.icon} size={14} />
        {tab.label}
      </span>
    </button>
  {/each}
</nav>