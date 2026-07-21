// Pure helpers for ProcessExplorer: tree builder + context-menu factory.
import { commentsModal } from '../../stores/ui/commentsModal.svelte.js'

// buildProcessTree flattens the process list into a depth-annotated array
// so the DataTable can render tree indentation without a real tree widget.
export function buildProcessTree(procs) {
  const tree = {}
  const pidMap = new Set()

  const normalized = procs.map((p) => ({
    ...p,
    p_id: p.Pid || p.pid,
    pp_id: p.Ppid || p.ppid,
  }))
  normalized.forEach((p) => pidMap.add(p.p_id))

  const rootNodes = []
  normalized.forEach((p) => {
    if (p.pp_id === 0 || !pidMap.has(p.pp_id)) {
      rootNodes.push(p)
    } else {
      if (!tree[p.pp_id]) tree[p.pp_id] = []
      tree[p.pp_id].push(p)
    }
  })

  const flattened = []
  const traverse = (node, depth) => {
    flattened.push({ ...node, _depth: depth })
    if (tree[node.p_id]) {
      tree[node.p_id]
        .sort((a, b) => a.p_id - b.p_id)
        .forEach((child) => traverse(child, depth + 1))
    }
  }

  rootNodes.sort((a, b) => a.p_id - b.p_id).forEach((root) => traverse(root, 0))
  return flattened
}

// commandInvoker returns an on-handler that opens the command modal with
// a canned config + initial values. Every entry in the ProcessExplorer
// menu uses the same pattern, so factoring it out kills a lot of
// boilerplate.
function commandInvoker(commandModal, name, initialValues) {
  return () => commandModal.open({
    command: { name, path: name, arguments: [], flags: [], supported: true },
    initialValues,
    sourceContext: 'process-explorer',
  })
}

export function buildProcessContextSections({ pid, procName, commandModal, killProcess }) {
  return [
    {
      title: 'Injection',
      items: [
        { icon: 'syringe', label: 'Execute Assembly…',
          on: commandInvoker(commandModal, 'execute-assembly', { ppid: pid, process: procName }) },
        { icon: 'syringe', label: 'Sideload DLL…',
          on: commandInvoker(commandModal, 'sideload', { 'process-name': procName }) },
      ],
    },
    {
      title: 'Process',
      items: [
        { icon: 'arrow-left-right', label: 'Migrate Into…',
          on: commandInvoker(commandModal, 'migrate', { pid, arch: '' }) },
        { icon: 'download', label: 'Dump Memory…',
          on: commandInvoker(commandModal, 'procdump', { pid }) },
        { icon: 'shield-user', label: 'Get System…',
          on: () => {
            if (procName) commandInvoker(commandModal, 'getsystem', { process: procName })()
          } },
        { icon: 'message-square', label: 'Comments / Notes…',
          on: () => commentsModal.openComments('process', String(pid), `${procName || 'PID'} (${pid})`) },
      ],
    },
    {
      items: [
        { icon: 'skull', label: 'Kill Process', danger: true, on: () => killProcess(pid) },
      ],
    },
  ]
}
