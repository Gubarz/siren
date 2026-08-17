// Local line editor for piped (non-PTY) remote shells, e.g. a Windows
// powershell.exe/cmd.exe running on a plain stdin pipe. Such a shell has no
// console line discipline: it cannot process backspace (xterm sends DEL,
// which piped Windows shells ignore) and its echo of typed characters comes
// back late over the tunnel, making the display trail the operator's
// keystrokes.
//
// Instead we edit locally: printable characters are echoed instantly to the
// terminal, backspace erases the local copy, and only the completed line is
// sent to the remote shell on Enter. The remote's own echo of the submitted
// line is swallowed from the output stream so the line does not appear
// twice.

const ESC = '\x1b'
const ERASE = '\x08\x1b[K' // back one cell, clear to end of line

export function createShellLineEditor({ suppressTimeoutMs = 2000 } = {}) {
  let line = ''
  let escSeq = ''
  let suppress = []
  let suppressTimer = null
  let pendingNewline = false

  const disarm = () => {
    if (suppressTimer !== null) {
      clearTimeout(suppressTimer)
      suppressTimer = null
    }
    suppress = []
    pendingNewline = false
  }

  const arm = () => {
    if (suppress.length === 0) return
    if (suppressTimer !== null) clearTimeout(suppressTimer)
    suppressTimer = setTimeout(disarm, suppressTimeoutMs)
  }

  // output filters remote shell output: the echo of the last submitted
  // line(s) is consumed so it does not duplicate the local echo. Returns
  // the bytes that should be displayed.
  const output = (data) => {
    if (suppress.length === 0 && pendingNewline) {
      // The newline the remote emits right after its echo can arrive in a
      // chunk of its own; the local render already moved to the next line.
      pendingNewline = false
      let i = 0
      while (i < data.length && (data[i] === '\r' || data[i] === '\n')) i++
      if (i > 0) data = data.slice(i)
      if (data === '') return ''
    }
    while (suppress.length > 0 && data.length > 0) {
      const expected = suppress[0]
      if (data.startsWith(expected)) {
        suppress.shift()
        data = data.slice(expected.length)
        // Swallow the CR/LF the remote emits right after its echo; the
        // local render already advanced to the next line.
        while (data[0] === '\r' || data[0] === '\n') data = data.slice(1)
        if (data === '' && suppress.length === 0) pendingNewline = true
        continue
      }
      if (expected.startsWith(data)) {
        suppress[0] = expected.slice(data.length)
        data = ''
        continue
      }
      if (data[0] === '\r' || data[0] === '\n') {
        // The remote may emit the newline before the echo of the next
        // submitted line (multi-line paste), or between echoes. Treat a
        // newline run as echo noise while suppression is armed.
        let i = 0
        while (i < data.length && (data[i] === '\r' || data[i] === '\n')) i++
        if (i === data.length) return ''
        if (data.slice(i).startsWith(expected)) {
          data = data.slice(i)
          continue
        }
      }
      disarm()
      break
    }
    if (suppress.length === 0 && suppressTimer !== null) {
      clearTimeout(suppressTimer)
      suppressTimer = null
    }
    return data
  }

  // input consumes typed bytes and returns { render, send }: render goes to
  // the local terminal, send goes to the remote shell (usually empty until
  // Enter).
  const input = (data) => {
    let render = ''
    let send = ''

    const submit = () => {
      if (line !== '') {
        suppress.push(line)
        arm()
      }
      send += line + '\r'
      line = ''
      render += '\r\n'
    }

    for (const ch of data) {
      if (escSeq !== '') {
        escSeq += ch
        // '[' right after ESC starts a CSI sequence whose parameter bytes
        // follow until the final byte. Otherwise a byte in 0x40..0x7e
        // completes the sequence; anything below stays in it.
        if (escSeq === ESC + '[' || ch < '\x40' || ch > '\x7e') continue
        escSeq = ''
        continue
      }
      if (ch === ESC) {
        escSeq = ESC
        continue
      }
      if (ch === '\r' || ch === '\n') {
        submit()
        continue
      }
      if (ch === '\x7f' || ch === '\x08') {
        if (line !== '') {
          // Pop a whole code point so surrogate pairs are not split into
          // broken half-characters that would corrupt the submitted line.
          line = Array.from(line).slice(0, -1).join('')
          render += ERASE
        }
        continue
      }
      if (ch === '\x03') {
        line = ''
        render += '^C\r\n'
        continue
      }
      if (ch >= ' ') {
        line += ch
        render += ch
      }
    }
    return { render, send }
  }

  return { input, output, line: () => line }
}
