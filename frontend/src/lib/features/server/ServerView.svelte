<script>
  import ListenersPanel from './ListenersPanel.svelte';
  import LootPanel from './LootPanel.svelte';
  import CredentialsPanel from './CredentialsPanel.svelte';
  import OperatorsPanel from './OperatorsPanel.svelte';
  import EventsLog from './EventsLog.svelte';
  import ImplantBuildsPanel from './ImplantBuildsPanel.svelte';
  import ProfilesPanel from './ProfilesPanel.svelte';
  import CanariesPanel from './CanariesPanel.svelte';
  import ServerInfoPanel from './ServerInfoPanel.svelte';
  import HTTPC2ProfilesPanel from './HTTPC2ProfilesPanel.svelte';
  import TrafficEncodersPanel from './TrafficEncodersPanel.svelte';
  import ShellcodePanel from './ShellcodePanel.svelte';
  import WebsitesPanel from './WebsitesPanel.svelte';
  import HostsPanel from './HostsPanel.svelte';
  import MonitoringProvidersPanel from './MonitoringProvidersPanel.svelte';
  import CrackingPanel from './CrackingPanel.svelte';
  import BuildFarmPanel from './BuildFarmPanel.svelte';
  import CasesPanel from './CasesPanel.svelte';
  import AliasesPanel from './AliasesPanel.svelte';
  import Console from '$components/patterns/Console/Console.svelte';
  import Button from '$components/ui/Button.svelte';
  import Icon from '$components/ui/Icon.svelte';
  import Menu from '$components/ui/Menu.svelte';
  import MenuItem from '$components/ui/MenuItem.svelte';

  let { active = $bindable('listeners') } = $props();

  const tabs = [
    { id: 'console', label: 'Console', icon: 'terminal' },
    { id: 'listeners', label: 'Listeners', icon: 'headphones' },
    { id: 'builds', label: 'Builds', icon: 'factory' },
    { id: 'profiles', label: 'Profiles', icon: 'sliders' },
    { id: 'events', label: 'Events', icon: 'history' },
    { id: 'loot', label: 'Loot', icon: 'database' },
    { id: 'creds', label: 'Credentials', icon: 'key' },
    { id: 'operators', label: 'Operators', icon: 'users' },
    { id: 'aliases', label: 'Aliases', icon: 'package' },
    { id: 'cases', label: 'Cases', icon: 'folder' },
    { id: 'websites', label: 'Websites', icon: 'globe' },
    { id: 'http-c2', label: 'HTTP C2 Profiles', icon: 'network' },
    { id: 'traffic-encoders', label: 'Traffic Encoders', icon: 'shuffle' },
    { id: 'shellcode', label: 'Shellcode', icon: 'package' },
    { id: 'hosts', label: 'Hosts', icon: 'monitor' },
    { id: 'canaries', label: 'Canaries', icon: 'alert-triangle' },
    { id: 'monitors', label: 'Threat Monitors', icon: 'shield' },
    { id: 'cracking', label: 'Cracking', icon: 'skull' },
    { id: 'build-farm', label: 'Build Farm', icon: 'server' },
    { id: 'server', label: 'Server', icon: 'cog' },
  ];

  const primaryTabs = tabs.slice(0, 5);
  const tabGroups = [
    {
      id: 'ops',
      label: 'Operations',
      icon: 'users',
      tabs: tabs.filter((tab) => ['operators', 'aliases', 'cases', 'hosts', 'canaries', 'monitors'].includes(tab.id)),
    },
    {
      id: 'assets',
      label: 'Assets',
      icon: 'database',
      tabs: tabs.filter((tab) => ['loot', 'creds', 'websites'].includes(tab.id)),
    },
    {
      id: 'advanced',
      label: 'Advanced',
      icon: 'sliders',
      tabs: tabs.filter((tab) => ['http-c2', 'traffic-encoders', 'shellcode', 'cracking', 'build-farm', 'server'].includes(tab.id)),
    },
  ];
  const navTabBase = '!h-10 !rounded-none !border-x-0 !border-t-0 !border-b-2 !bg-transparent !px-3 !py-0 !text-sm !font-medium !shadow-none focus:!ring-0';
  const activeTabClass = '!border-brand !text-brand !bg-brand/5';
  const inactiveTabClass = '!border-transparent !text-fg-muted hover:!border-fg-muted hover:!text-fg';
  let opsOpen = $state(false);
  let assetsOpen = $state(false);
  let advancedOpen = $state(false);

  function selectTab(tabId, groupId = '') {
    active = tabId;
    closeGroup(groupId);
  }

  function groupIsActive(group) {
    return group.tabs.some((tab) => tab.id === active);
  }

  function closeGroup(groupId) {
    if (groupId === 'ops') opsOpen = false;
    if (groupId === 'assets') assetsOpen = false;
    if (groupId === 'advanced') advancedOpen = false;
  }

  function groupOpen(groupId) {
    if (groupId === 'ops') return opsOpen;
    if (groupId === 'assets') return assetsOpen;
    if (groupId === 'advanced') return advancedOpen;
    return false;
  }
</script>

<div class="flex flex-col h-full bg-canvas">
  {#snippet menuItems(grp)}
    {#each grp.tabs as tab (tab.id)}
      <MenuItem
        onclick={() => selectTab(tab.id, grp.id)}
        class={active === tab.id ? '!bg-brand/10 !text-brand' : ''}
      >
        <Icon name={tab.icon} size={14} />
        <span>{tab.label}</span>
      </MenuItem>
    {/each}
  {/snippet}
  <div class="flex min-w-0 items-center gap-1 border-b border-line bg-chrome px-2" role="tablist" tabindex="-1">
    {#each primaryTabs as tab (tab.id)}
      <Button
        color="alternative"
        size="xs"
        class="{navTabBase} {active === tab.id ? activeTabClass : inactiveTabClass}"
        onclick={() => selectTab(tab.id)}
      >
        <Icon name={tab.icon} size={14} />
        {tab.label}
      </Button>
    {/each}

    <div class="mx-1 h-5 w-px bg-line"></div>

    {#each tabGroups as group (group.id)}
      {@const isActiveGroup = groupIsActive(group)}
      <div class="relative shrink-0">
        <Button
          color="alternative"
          size="xs"
          class="{navTabBase} {isActiveGroup ? activeTabClass : inactiveTabClass}"
          aria-haspopup="true"
          aria-expanded={groupOpen(group.id)}
        >
          <Icon name={group.icon} size={14} />
          {group.label}
          <Icon name="chevron-down" size={12} />
        </Button>
        {#if group.id === 'ops'}
          <Menu bind:isOpen={opsOpen} minWidth="13rem">{@render menuItems(group)}</Menu>
        {:else if group.id === 'assets'}
          <Menu bind:isOpen={assetsOpen} minWidth="13rem">{@render menuItems(group)}</Menu>
        {:else if group.id === 'advanced'}
          <Menu bind:isOpen={advancedOpen} minWidth="13rem">{@render menuItems(group)}</Menu>
        {/if}
      </div>
    {/each}
  </div>

  <div class="flex-1 overflow-auto relative">
    {#if active === 'console'}
      <div class="flex flex-col h-full p-0">
        <Console sessionID="" />
      </div>
    {:else if active === 'listeners'}
      <ListenersPanel embedded />
    {:else if active === 'loot'}
      <LootPanel embedded />
    {:else if active === 'creds'}
      <CredentialsPanel embedded />
    {:else if active === 'operators'}
      <OperatorsPanel embedded />
    {:else if active === 'aliases'}
      <AliasesPanel embedded />
    {:else if active === 'cases'}
      <CasesPanel embedded />
    {:else if active === 'websites'}
      <WebsitesPanel embedded />
    {:else if active === 'http-c2'}
      <HTTPC2ProfilesPanel embedded />
    {:else if active === 'traffic-encoders'}
      <TrafficEncodersPanel embedded />
    {:else if active === 'shellcode'}
      <ShellcodePanel embedded />
    {:else if active === 'hosts'}
      <HostsPanel embedded />
    {:else if active === 'canaries'}
      <CanariesPanel embedded />
    {:else if active === 'monitors'}
      <MonitoringProvidersPanel embedded />
    {:else if active === 'cracking'}
      <CrackingPanel embedded />
    {:else if active === 'build-farm'}
      <BuildFarmPanel embedded />
    {:else if active === 'server'}
      <ServerInfoPanel embedded />
    {:else if active === 'events'}
      <EventsLog />
    {:else if active === 'builds'}
      <ImplantBuildsPanel embedded />
    {:else if active === 'profiles'}
      <ProfilesPanel embedded />
    {/if}
  </div>
</div>
