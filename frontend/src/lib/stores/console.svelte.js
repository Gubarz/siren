import { ListCommands, RunSessionCommand, SendToSessionConsole } from '../api/console.js'
import { errorMessage } from '../utils/errors.js'

// Per-session console state, keyed by sessionID.
// Declared as a reactive $state object. Svelte 5 will deeply track
// mutations inside it (like array pushes, property sets, etc.).
const sessions = $state({})
const EMPTY_SESSION = { lines: [], history: [], histIdx: 0, busy: false }

function sessionKey(id) {
  return id || '_'
}

function createSession() {
  return { lines: [], history: [], histIdx: 0, busy: false }
}

// The real implant command list, fetched once from the backend.
let commandsPromise = null
export function getCommands() {
  if (!commandsPromise) {
    commandsPromise = ListCommands().catch(() => [])
  }
  return commandsPromise
}

export function peekSession(id) {
  return sessions[sessionKey(id)]
}

export function ensureSession(id) {
  const key = sessionKey(id)
  if (!sessions[key]) {
    sessions[key] = createSession()
  }
  return sessions[key]
}

export function getSession(id) {
  return ensureSession(id)
}

export function emptySession() {
  return EMPTY_SESSION
}

export function pushLine(id, line) {
  ensureSession(id).lines.push(line)
}

function clearSession(id) {
  ensureSession(id).lines = []
}

// dispatchCommand runs a command against a specific session and records the
// input + output into that session's buffer.
export async function dispatchCommand(id, raw) {
  const cmd = (raw || '').trim()
  if (!cmd) return

  const s = ensureSession(id)
  if (s.busy) {
    pushLine(id, { type: 'error', text: 'Another command is still running for this console.' })
    return
  }

  // History bookkeeping.
  s.history.push(cmd)
  s.histIdx = s.history.length

  // Client-only conveniences.
  if (cmd === 'clear' || cmd === 'cls') {
    clearSession(id)
    return
  }

  // Session commands route to the session's live subprocess console so
  // any interactive prompt (forms.Select / tea) renders in xterm.js.
  // Server-scoped commands (no session) still use the in-process path
  // because they don't have a per-target subprocess to land in.
  if (id) {
    try {
      await SendToSessionConsole(id, cmd)
    } catch (e) {
      pushLine(id, { type: 'error', text: errorMessage(e) })
    }
    return
  }

  pushLine(id, { type: 'input', text: cmd })
  s.busy = true

  try {
    const out = await RunSessionCommand(id, cmd)
    if (out !== '') {
      pushLine(id, { type: 'output', text: out })
    }
  } catch (e) {
    pushLine(id, { type: 'error', text: errorMessage(e) })
  } finally {
    s.busy = false
  }
}
