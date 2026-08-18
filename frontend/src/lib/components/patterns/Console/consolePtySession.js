import {
  AcquireConsole,
  GetConsoleOutput,
  ResizeConsole,
  StopConsole,
  WriteConsole,
} from '../../../api/console.js'
import { onConsoleExit, onConsoleOpenShell, onConsoleOutput } from '../../../api/runtime.js'
import { errorMessage } from '$utils/errors.js'
import { createXterm } from '$utils/xterm.js'

const RELEASE_DELAY_MS = 3000
const STOP_CONSOLE_ON_DISPOSE = !import.meta.hot
const sessionState = globalThis.__sliverGuiConsolePtySessionState ||= { sessions: new Map() }
const { sessions } = sessionState

if (!sessionState.pageHideInstalled) {
  sessionState.pageHideInstalled = true
  globalThis.addEventListener?.('pagehide', () => {
    for (const record of sessions.values()) {
      releaseConsoleLease(record)
    }
  })
}

function releaseConsoleLease(record) {
  if (!record.jobID || record.leaseReleased || !STOP_CONSOLE_ON_DISPOSE) return
  record.leaseReleased = true
  StopConsole(record.jobID).catch(() => {})
}

function decodeBase64(b64) {
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return bytes
}

function isTargetNavigation(line) {
  const trimmed = line.trim()
  if (trimmed === 'background') return true
  if (trimmed === 'use' || trimmed.startsWith('use ')) return true
  return /^sessions(?:\s|$)/.test(trimmed) && /(?:^|\s)(?:-i|--interact)(?:=|\s|$)/.test(trimmed)
}

function promptAction(record, data) {
  let action = null
  for (const ch of data) {
    if (ch === '\r' || ch === '\n') {
      const line = record.promptLine
      record.promptLine = ''
      if (record.sessionID && isTargetNavigation(line)) {
        action = { type: 'pinned' }
      }
      continue
    }
    if (ch === '\x7f' || ch === '\b') {
      record.promptLine = record.promptLine.slice(0, -1)
      continue
    }
    if (ch === '\x03' || ch === '\x15' || ch === '\x1b') {
      record.promptLine = ''
      continue
    }
    if (ch >= ' ') {
      record.promptLine += ch
    }
  }
  return action
}

function installRuntimeListeners(record) {
  record.stopOutput = onConsoleOutput((ev) => {
    if (ev.jobID !== record.jobID) return
    const output = decodeBase64(ev.data)
    if (record.replaying) record.replayQueue.push(output)
    else record.term.write(output)
  })
  record.stopExit = onConsoleExit((ev) => {
    if (ev.jobID !== record.jobID) return
    record.term.write(`\r\n\x1b[90m[console exited${ev.exitCode !== undefined ? ` (${ev.exitCode})` : ''}]\x1b[0m\r\n`)
    record.jobID = null
  })
  record.stopOpenShell = onConsoleOpenShell((ev) => {
    if (ev.jobID !== record.jobID) return
    record.term.write('\r\n\x1b[90m[opened shell tab]\x1b[0m\r\n')
    record.onshell?.(ev.tail || '')
  })
}

function startConsole(record) {
  if (record.starting || record.jobID) return
  record.starting = true
  AcquireConsole(record.sessionID)
    .then(({ jobID: id, existing }) => {
      record.starting = false
      if (record.disposed) {
        if (STOP_CONSOLE_ON_DISPOSE) StopConsole(id).catch(() => {})
        return
      }
      record.replaying = true
      record.muteReplayInput = existing
      record.jobID = id
      record.leaseReleased = false
      record.term.focus()
      resizeConsole(record)
      GetConsoleOutput(id)
        .then((output) => replayConsoleOutput(record, output))
        .catch(() => finishConsoleReplay(record))
    })
    .catch((err) => {
      record.starting = false
      if (record.disposed) return
      record.term.write(`\r\n\x1b[31m[!] ${errorMessage(err)}\x1b[0m\r\n`)
    })
}

function replayConsoleOutput(record, output) {
  if (record.disposed) return
  const buffered = output ? decodeBase64(output) : null
  if (!buffered?.length) {
    finishConsoleReplay(record)
    return
  }
  if (record.muteReplayInput) record.term.options.disableStdin = true
  record.term.write(buffered, () => finishConsoleReplay(record))
}

function finishConsoleReplay(record) {
  if (record.disposed) return
  if (record.muteReplayInput) record.term.options.disableStdin = false
  record.muteReplayInput = false
  for (const output of record.replayQueue) record.term.write(output)
  record.replayQueue = []
  record.replaying = false
}

function createRecord(sessionID, hostEl, onshell) {
  const { term, fit } = createXterm(hostEl)
  const record = {
    sessionID,
    term,
    fit,
    jobID: null,
    leaseReleased: false,
    starting: false,
    disposed: false,
    replaying: false,
    muteReplayInput: false,
    replayQueue: [],
    refs: 0,
    releaseTimer: null,
    promptLine: '',
    onshell,
    stopOutput: null,
    stopExit: null,
    stopOpenShell: null,
    dataDisposable: null,
  }

  installRuntimeListeners(record)
  record.dataDisposable = term.onData((data) => {
    if (record.disposed || !record.jobID) return
    const action = promptAction(record, data)
    if (action?.type === 'pinned') {
      WriteConsole(record.jobID, '\x15').catch(() => {})
      term.write('\r\n\x1b[90m[this console is pinned to this agent; open another agent console to switch]\x1b[0m\r\n')
      return
    }
    WriteConsole(record.jobID, data).catch(() => {})
  })

  startConsole(record)
  return record
}

function attachRecord(record, hostEl, onshell) {
  record.onshell = onshell
  if (record.releaseTimer) {
    clearTimeout(record.releaseTimer)
    record.releaseTimer = null
  }
  if (record.term.element && record.term.element.parentElement !== hostEl) {
    hostEl.appendChild(record.term.element)
  }
  resizeConsole(record)
}

function resizeConsole(record) {
  try {
    record.fit.fit()
    if (record.jobID) {
      ResizeConsole(record.jobID, record.term.cols, record.term.rows).catch(() => {})
    }
  } catch {
    // xterm briefly has no measurable parent while Svelte is moving panes.
  }
}

function disposeRecord(key, record) {
  record.disposed = true
  record.stopOutput?.()
  record.stopExit?.()
  record.stopOpenShell?.()
  record.dataDisposable?.dispose()
  releaseConsoleLease(record)
  record.term.dispose()
  sessions.delete(key)
}

export function acquireConsolePty(sessionID, hostEl, onshell = null) {
  let record = sessions.get(sessionID)
  if (!record || record.disposed) {
    record = createRecord(sessionID, hostEl, onshell)
    sessions.set(sessionID, record)
  } else {
    attachRecord(record, hostEl, onshell)
  }

  record.refs += 1

  return {
    resize: () => resizeConsole(record),
    release: () => {
      record.refs = Math.max(0, record.refs - 1)
      if (record.refs > 0 || record.releaseTimer) return
      record.releaseTimer = setTimeout(() => {
        record.releaseTimer = null
        if (record.refs > 0) return
        disposeRecord(sessionID, record)
      }, RELEASE_DELAY_MS)
    },
  }
}
