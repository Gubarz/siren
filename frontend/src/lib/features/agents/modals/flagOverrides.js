export const GROUP_ORDER = ['payload', 'process', 'evasion', 'advanced', 'output', 'other']
export const GROUP_LABELS = {
  payload: 'Payload',
  process: 'Host Process',
  evasion: 'Evasion',
  advanced: 'Advanced',
  output: 'Output',
  other: 'Other',
}
export const COLLAPSED_GROUPS = new Set(['advanced', 'output'])

export const flagOverrides = {
  'execute-assembly': {
    args: [
      { name: 'local path to assembly', widgetHint: 'file', label: '.NET Assembly', group: 'payload' },
      { name: 'arguments', widgetHint: 'text', label: 'Assembly Arguments', group: 'payload' },
    ],
    flags: {
      'process': { group: 'process', label: 'Host process' },
      'ppid': { group: 'process', label: 'Parent PID', widgetHint: 'pidPicker', processFlag: 'process' },
      'process-arguments': { group: 'process', label: 'Process arguments' },
      'in-process': { group: 'evasion', label: 'Run in current process (no fork+run)' },
      'amsi-bypass': { group: 'evasion', label: 'Patch AMSI (evade Defender)' },
      'etw-bypass': { group: 'evasion', label: 'Patch ETW (evade Event Tracing)' },
      'arch': { group: 'advanced', label: 'Target architecture' },
      'runtime': { group: 'advanced', label: '.NET runtime version' },
      'app-domain': { group: 'advanced', label: 'App domain' },
      'class': { group: 'advanced', label: 'Class name' },
      'method': { group: 'advanced', label: 'Method name' },
      'save': { group: 'advanced', label: 'Save to disk on target' },
      'loot': { group: 'output', label: 'Save output to loot store' },
      'name': { group: 'output', label: 'Loot entry name' },
    },
  },
  'sideload': {
    args: [
      { name: 'local path to dll', widgetHint: 'file', label: 'DLL', group: 'payload' },
    ],
    flags: {
      'entry-point': { group: 'payload', label: 'Entry point' },
      'process-name': { group: 'process', label: 'Host process' },
      'args': { group: 'process', label: 'Arguments' },
      'is-unmanaged': { group: 'process', label: 'Use unmanaged process (no fork+run)' },
      'runtime': { group: 'advanced', label: '.NET runtime version' },
      'save': { group: 'advanced', label: 'Save to disk on target' },
    },
  },
  'spawndll': {
    args: [
      { name: 'local path to dll', widgetHint: 'file', label: 'DLL', group: 'payload' },
    ],
    flags: {
      'process-name': { group: 'process', label: 'Host process' },
      'args': { group: 'process', label: 'Arguments' },
      'kill': { group: 'process', label: 'Kill current process after execution' },
      'ppid': { group: 'process', label: 'Parent PID', widgetHint: 'pidPicker', processFlag: 'process-name' },
      'offset': { group: 'advanced', label: 'DLL offset' },
    },
  },
  'execute-shellcode': {
    args: [
      { name: 'local path to shellcode', widgetHint: 'file', label: 'Shellcode', group: 'payload' },
    ],
    flags: {
      'architecture': { group: 'process', label: 'Target architecture' },
      'process': { group: 'process', label: 'Host process' },
      'ppid': { group: 'process', label: 'Parent PID', widgetHint: 'pidPicker', processFlag: 'process' },
    },
  },
  'migrate': {
    args: [],
    flags: {
      'pid': { group: 'process', label: 'Target PID', widgetHint: 'pidPicker', default: '' },
      'process-name': { group: 'process', label: 'Process name' },
      'shellcode-encoder': { group: 'evasion', label: 'Shellcode encoder' },
      'timeout': { group: 'advanced', label: 'Timeout (seconds)' },
    },
  },
  'procdump': {
    args: [],
    flags: {
      'pid': { group: 'process', label: 'Process PID', widgetHint: 'pidPicker', processFlag: 'name' },
      'name': { group: 'process', label: 'Process name' },
      'loot': { group: 'output', label: 'Save dump to loot store' },
      'save': { group: 'output', label: 'Save dump to disk' },
    },
  },
  'getsystem': {
    args: [],
    flags: {
      'process': { group: 'process', label: 'Spawn process' },
      'config': { group: 'advanced', label: 'Implant profile' },
    },
  },
  'execute': {
    args: [
      { name: 'command', widgetHint: 'text', label: 'Program', group: 'payload' },
      { name: 'arguments', widgetHint: 'text', label: 'Arguments', group: 'payload' },
    ],
    flags: {
      'output': { group: 'output', label: 'Capture output' },
      'ignore-stderr': { group: 'output', label: 'Ignore stderr' },
      'stdout': { group: 'output', label: 'Redirect stdout on target' },
      'stderr': { group: 'output', label: 'Redirect stderr on target' },
      'loot': { group: 'output', label: 'Save output to loot store' },
      'name': { group: 'output', label: 'Loot entry name' },
      'ppid': { group: 'advanced', label: 'Parent PID (spoof)', widgetHint: 'pidPicker' },
      'token': { group: 'advanced', label: 'Use current impersonation token' },
    },
  },
  'impersonate': {
    args: [
      { name: 'username', widgetHint: 'text', label: 'Username', group: 'payload' },
    ],
    flags: {},
  },
  'make-token': {
    args: [],
    flags: {
      'username': { group: 'payload', label: 'Username' },
      'password': { group: 'payload', label: 'Password', widgetHint: 'password' },
      'domain': { group: 'payload', label: 'Domain' },
      'logon-type': { group: 'advanced', label: 'Logon type' },
    },
  },
  'runas': {
    args: [],
    flags: {
      'username': { group: 'payload', label: 'Username' },
      'password': { group: 'payload', label: 'Password', widgetHint: 'password' },
      'domain': { group: 'payload', label: 'Domain' },
      'program': { group: 'process', label: 'Program' },
      'args': { group: 'process', label: 'Arguments' },
      'net-only': { group: 'advanced', label: 'Net-only logon' },
      'show-window': { group: 'advanced', label: 'Show window' },
    },
  },
  'rev2self': {
    args: [],
    flags: {},
  },
  'getprivs': {
    args: [],
    flags: {},
  },
  'socks5 start': {
    args: [],
    flags: {
      'host': { group: 'payload', label: 'Bind host' },
      'port': { group: 'payload', label: 'Local port' },
      'user': { group: 'advanced', label: 'Auth username' },
    },
  },
  'portfwd add': {
    args: [],
    flags: {
      'bind': { group: 'payload', label: 'Local bind (host:port)' },
      'remote': { group: 'payload', label: 'Remote target (host:port)' },
    },
  },
  'rportfwd add': {
    args: [],
    flags: {
      'bind': { group: 'payload', label: 'Implant bind (host:port)' },
      'remote': { group: 'payload', label: 'Forward to (host:port)' },
    },
  },
  'backdoor': {
    args: [
      { name: 'remote file', widgetHint: 'remoteFile', label: 'Remote file to backdoor', group: 'payload' },
    ],
    flags: {
      'profile': { group: 'payload', label: 'Implant profile' },
    },
  },
  'dllhijack': {
    args: [
      { name: 'target', widgetHint: 'remoteFile', label: 'Target path (plant the DLL here)', group: 'payload' },
    ],
    flags: {
      'reference-path': { group: 'payload', label: 'Reference DLL on target', widgetHint: 'remoteFile' },
      'reference-file': { group: 'advanced', label: 'Reference DLL (local override)', widgetHint: 'file' },
      'file': { group: 'payload', label: 'Local hijack DLL', widgetHint: 'file' },
      'profile': { group: 'payload', label: 'Implant profile' },
    },
  },
}
