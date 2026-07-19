import {
  ResizeConsole,
  StartConsole,
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
    record.term.write(decodeBase64(ev.data))
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
  StartConsole(record.sessionID)
    .then((id) => {
      record.starting = false
      if (record.disposed) {
        if (STOP_CONSOLE_ON_DISPOSE) StopConsole(id).catch(() => {})
        return
      }
      record.jobID = id
      record.term.focus()
      resizeConsole(record)
    })
    .catch((err) => {
      record.starting = false
      if (record.disposed) return
      record.term.write(`\r\n\x1b[31m[!] ${errorMessage(err)}\x1b[0m\r\n`)
    })
}

function createRecord(sessionID, hostEl, onshell) {
  const { term, fit } = createXterm(hostEl)
  const record = {
    sessionID,
    term,
    fit,
    jobID: null,
    starting: false,
    disposed: false,
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
  if (record.jobID && STOP_CONSOLE_ON_DISPOSE) StopConsole(record.jobID).catch(() => {})
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
