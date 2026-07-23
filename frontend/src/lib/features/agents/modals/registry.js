import ExecuteAssemblyModal from './commands/ExecuteAssemblyModal.svelte'
import SideloadModal from './commands/SideloadModal.svelte'
import SpawnDllModal from './commands/SpawnDllModal.svelte'
import ExecuteModal from './commands/ExecuteModal.svelte'
import ExecuteShellcodeModal from './commands/ExecuteShellcodeModal.svelte'
import MigrateModal from './commands/MigrateModal.svelte'
import ProcdumpModal from './commands/ProcdumpModal.svelte'
import GetSystemModal from './commands/GetSystemModal.svelte'
import ImpersonateModal from './commands/ImpersonateModal.svelte'
import MakeTokenModal from './commands/MakeTokenModal.svelte'
import RunAsModal from './commands/RunAsModal.svelte'
import RevToSelfModal from './commands/RevToSelfModal.svelte'
import GetPrivsModal from './commands/GetPrivsModal.svelte'
import Socks5StartModal from './commands/Socks5StartModal.svelte'
import PortfwdAddModal from './commands/PortfwdAddModal.svelte'
import RportfwdAddModal from './commands/RportfwdAddModal.svelte'
import BackdoorModal from './commands/BackdoorModal.svelte'
import DllHijackModal from './commands/DllHijackModal.svelte'
import PivotStartModal from './commands/PivotStartModal.svelte'

const OVERRIDES = {
  'execute-assembly': ExecuteAssemblyModal,
  'sideload': SideloadModal,
  'spawndll': SpawnDllModal,
  'execute': ExecuteModal,
  'execute-shellcode': ExecuteShellcodeModal,
  'migrate': MigrateModal,
  'procdump': ProcdumpModal,
  'getsystem': GetSystemModal,
  'impersonate': ImpersonateModal,
  'make-token': MakeTokenModal,
  'runas': RunAsModal,
  'rev2self': RevToSelfModal,
  'getprivs': GetPrivsModal,
  'socks5 start': Socks5StartModal,
  'portfwd add': PortfwdAddModal,
  'rportfwd add': RportfwdAddModal,
  'backdoor': BackdoorModal,
  'dllhijack': DllHijackModal,
  'pivots': PivotStartModal,
  'pivots tcp': PivotStartModal,
  'pivots named-pipe': PivotStartModal,
}

export function modalFor(commandPath) {
  return OVERRIDES[commandPath] || null
}
