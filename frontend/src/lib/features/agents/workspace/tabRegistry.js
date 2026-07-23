// Registry for every tab type the AgentBottomPane can render. Instead of
// a fifteen-branch {#if/:else if} tree in the template, we look up the
// component + prop shape here.
//
// Each entry has:
//   component — the Svelte component to render
//   props     — a function that unpacks tab metadata + context into the
//               props the component expects. Kept as a function so the
//               registry can express per-tab quirks (BeaconTasks wants
//               `beaconID` not `sessionID`, ProcessExplorer wants an
//               `oncommand` callback, etc.) without smearing them across
//               the caller.

import BeaconTasks from '../BeaconTasks.svelte'
import FileBrowser from '../FileBrowser.svelte'
import ProcessExplorer from '../ProcessExplorer.svelte'
import RegistryBrowser from '../RegistryBrowser.svelte'
import ScreenshotViewer from '../ScreenshotViewer.svelte'
import ShellTerminal from '../ShellTerminal.svelte'
import GrepTab from '../GrepTab.svelte'
import ServicesTab from '../ServicesTab.svelte'
import TunnelsPanel from '../TunnelsPanel.svelte'
import ExtensionsTab from '../ExtensionsTab.svelte'
import MemfilesTab from '../MemfilesTab.svelte'
import NetworkCommandTab from '../NetworkCommandTab.svelte'
import WireGuardTunnelsTab from '../WireGuardTunnelsTab.svelte'
import PrivilegesTab from '../PrivilegesTab.svelte'
import Console from '$components/patterns/Console/Console.svelte'

const bySessionID = (tab) => ({ sessionID: tab.sessionId })

const TabRegistry = {
  console: {
    component: Console,
    props: (tab, ctx) => ({
      sessionID: tab.sessionId,
      onshell: (program) => ctx.openShell(tab.sessionId, program),
    }),
  },
  tasks: {
    component: BeaconTasks,
    props: (tab, ctx) => ({
      beaconID: tab.sessionId,
      active: ctx.isActive(tab),
    }),
  },
  fileBrowser:     { component: FileBrowser,       props: bySessionID },
  processExplorer: {
    component: ProcessExplorer,
    props: (tab, ctx) => ({
      sessionID: tab.sessionId,
      oncommand: ({ cmd }) => ctx.runConsoleCommand(tab.sessionId, cmd),
    }),
  },
  registryBrowser: { component: RegistryBrowser,   props: bySessionID },
  ifconfig:        { component: NetworkCommandTab, props: (tab) => ({ sessionID: tab.sessionId, command: 'ifconfig' }) },
  netstat:         { component: NetworkCommandTab, props: (tab) => ({ sessionID: tab.sessionId, command: 'netstat' }) },
  screenshot:      { component: ScreenshotViewer,  props: bySessionID },
  grep:            { component: GrepTab,           props: bySessionID },
  services:        { component: ServicesTab,       props: bySessionID },
  tunneling:       { component: TunnelsPanel,      props: bySessionID },
  extensions:      { component: ExtensionsTab,     props: bySessionID },
  memfiles:        { component: MemfilesTab,       props: bySessionID },
  privileges:       { component: PrivilegesTab,      props: bySessionID },
  'wg-tunnels':    { component: WireGuardTunnelsTab, props: bySessionID },
}

// resolveTab is what AgentBottomPane calls per-tab. Handles the `shell-*`
// prefix separately since each shell tab is keyed by a runtime-generated
// id, not a static registry entry.
export function resolveTab(tab, ctx) {
  if (!tab?.type) return null
  if (tab.type.startsWith('shell-')) {
    const shell = ctx.shellsByID?.[tab.type]
    return shell ? { component: ShellTerminal, props: { shell, active: ctx.isActive(tab) } } : null
  }
  const entry = TabRegistry[tab.type]
  if (!entry) return null
  return { component: entry.component, props: entry.props(tab, ctx) }
}
