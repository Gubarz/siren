<script>
  import { onMount } from 'svelte'
  import {
    GetShellOutput,
    ResizeShell,
    WriteShell,
  } from '../../api/console.js'
  import { onShellOutput } from '../../api/runtime.js'
  import { errorMessage } from '../../utils/errors.js'
  import { createShellLineEditor } from '../../utils/shellLineEditor.js'
  import { createXterm } from '../../utils/xterm.js'

  let { shell = null } = $props()

  let hostEl
  let term
  let fit
  let ro
  let closed = $state(false)
  let lastCols = 0
  let lastRows = 0
  let shellID = $derived(shell?.id || shell?.ID || '')
  let shellPTY = $derived(shell?.pty ?? shell?.PTY ?? false)

  onMount(() => {
    ;({ term, fit } = createXterm(hostEl))
    term.focus()

    // For PTY shells, forward keystrokes verbatim — xterm produces correct
    // VT sequences for arrows, control chars, meta. For non-PTY (Windows
    // piped) shells the remote has no console line discipline: backspace is
    // ignored and its echo trails the operator's keystrokes, so a local
    // line editor echoes instantly, edits backspace locally, and submits
    // completed lines on Enter. The remote echo of a submitted line is
    // swallowed so it is not displayed twice.
    const lineEditor = shellPTY ? null : createShellLineEditor()

    const stopOutput = onShellOutput((event) => {
      if (event.id !== shellID) return
      if (event.data) term.write(lineEditor ? lineEditor.output(event.data) : event.data)
      if (event.error) term.write(`\r\n\x1b[31m[!] ${event.error}\x1b[0m\r\n`)
      if (event.closed) {
        closed = true
        term.write('\r\n\x1b[90m[shell closed]\x1b[0m\r\n')
      }
    })

    term.onData((data) => {
      if (closed) return
      if (lineEditor) {
        const { render, send } = lineEditor.input(data)
        if (render) term.write(render)
        if (send) {
          WriteShell(shellID, send).catch((err) => {
            term.write(`\r\n\x1b[31m[!] ${errorMessage(err)}\x1b[0m\r\n`)
            closed = true
          })
        }
        return
      }
      WriteShell(shellID, data).catch((err) => {
        term.write(`\r\n\x1b[31m[!] ${errorMessage(err)}\x1b[0m\r\n`)
        closed = true
      })
    })

    ro = new ResizeObserver(() => {
      if (!shellPTY) return
      try {
        fit.fit()
        const { cols, rows } = term
        if (cols === lastCols && rows === lastRows) return
        lastCols = cols
        lastRows = rows
        ResizeShell(shellID, rows, cols).catch(() => {})
      } catch { /* transient */ }
    })
    ro.observe(hostEl)

    // Replay any bytes buffered before we mounted (e.g. shell started in
    // another tab). Do this after subscribing so we don't lose live bytes
    // arriving during the fetch.
    ;(async () => {
      try {
        const buffered = await GetShellOutput(shellID)
        if (buffered) term.write(buffered)
      } catch (err) {
        term.write(`\r\n\x1b[31m[!] ${errorMessage(err)}\x1b[0m\r\n`)
      }
    })()

    return () => {
      stopOutput()
      ro?.disconnect()
      term?.dispose()
    }
  })

</script>

<div class="flex h-full w-full min-w-0 flex-col overflow-hidden bg-canvas text-fg">
  <div bind:this={hostEl} class="min-h-0 min-w-0 flex-1 overflow-hidden"></div>
</div>
