<script module>
  let armoryCache = null
  let armoryWarningCache = ''
</script>

<script>
  import { onMount } from 'svelte'
  import Icon from '$components/ui/Icon.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Button from '$components/ui/Button.svelte'
  import { ListArmoryPackages, ListCommands, RunSessionCommand } from '../../api/console.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'
  import { quote } from '../../utils/shell.js'
  import { stripTerminalFormatting } from '../../utils/text.js'

  let { onclose } = $props()

  let packages = $state(armoryCache || [])
  let loading = $state(!armoryCache)
  let error = $state('')
  let warning = $state(armoryWarningCache)
  let installing = $state('')

  const tableColumns = [
    { key: 'armory', label: 'Armory', width: 130 },
    { key: 'command', label: 'Command', width: 150 },
    { key: 'version', label: 'Version', width: 100 },
    { key: 'type', label: 'Type', width: 100 },
    { key: 'installedStr', label: 'Status', width: 120 },
    { key: 'action', label: 'Action', width: 150 },
  ]

  onMount(() => { loadArmory() })

  async function loadArmory(force = false) {
    if (armoryCache && !force) {
      packages = armoryCache
      warning = armoryWarningCache
      return
    }
    loading = true
    error = ''
    warning = ''
    try {
      const pkgInfos = await ListArmoryPackages(true)
      const installedCommands = await ListCommands()
      const installedSet = new Set(installedCommands || [])
      packages = (pkgInfos || []).map((p) => ({
        armory: p.armoryName,
        command: p.commandName,
        version: p.version,
        type: p.type,
        help: p.help,
        installed: installedSet.has(p.commandName),
        installedStr: installedSet.has(p.commandName) ? 'Installed' : 'Not Installed',
      }))
      armoryCache = packages
      armoryWarningCache = ''
      if (packages.length === 0) error = 'Armory returned no packages.'
    } catch {
      try {
        const output = await RunSessionCommand('', 'armory')
        const installedCommands = await ListCommands()
        packages = parseArmoryOutput(output, new Set(installedCommands || []))
        warning = parseArmoryWarnings(output)
        armoryCache = packages
        armoryWarningCache = warning
        if (packages.length === 0) {
          const cleaned = stripTerminalFormatting(output).trim()
          error = cleaned || 'Armory returned no packages.'
        }
      } catch (err) {
        error = errorMessage(err)
      }
    } finally {
      loading = false
    }
  }

  function parseArmoryOutput(output, installedCommands) {
    const cleanedOutput = stripTerminalFormatting(output)
    const lines = cleanedOutput.split('\n')
    const pkgs = []
    let columnsByName = null

    for (const line of lines) {
      if (!line.includes('\u2502') && !line.includes('|')) {
        if (/bundles/i.test(line)) columnsByName = null
        continue
      }

      const delimiter = line.includes('\u2502') ? '\u2502' : '|'
      const cells = line
        .split(delimiter)
        .slice(1, -1)
        .map((cell) => cell.trim())
      if (cells.length < 2) continue

      const normalized = cells.map((cell) => cell.toLowerCase())
      if (normalized.includes('command name')) {
        columnsByName = Object.fromEntries(normalized.map((name, index) => [name, index]))
        continue
      }
      if (!columnsByName) continue

      const command = cells[columnsByName['command name']] || ''
      if (!command || /^[-\s]+$/.test(command)) continue
      const installed = installedCommands.has(command)
      pkgs.push({
        armory: cells[columnsByName.armory] || '',
        command,
        version: cells[columnsByName.version] || '',
        type: cells[columnsByName.type] || '',
        installed,
        installedStr: installed ? 'Installed' : 'Not Installed',
      })
    }

    const parsed = pkgs.length > 0
      ? pkgs
      : parseBorderlessArmoryTable(cleanedOutput, installedCommands)
    return parsed.filter((pkg, index) => {
      return parsed.findIndex((candidate) =>
        candidate.armory === pkg.armory && candidate.command === pkg.command
      ) === index
    })
  }

  function parseBorderlessArmoryTable(output, installedCommands) {
    const packagesStart = output.search(/\bPackages\b[\s\S]*?\bArmory\s+Command Name\s+Version\s+Type\s+Help\b/i)
    if (packagesStart < 0) return []

    const packageSection = output
      .slice(packagesStart)
      .split(/\n\s*Bundles\b/i)[0]
      .replace(/\bPackages\b[\s\S]*?\bArmory\s+Command Name\s+Version\s+Type\s+Help\b/i, ' ')
      .replace(/[=\u2500\u2501-]{3,}/g, ' ')
      .replace(/\s+/g, ' ')
      .trim()

    const rowPattern = /(\S+)\s+(\S+)\s+(\S+)\s+(Alias|Extension)\s+([\s\S]*?)(?=\s+\S+\s+\S+\s+\S+\s+(?:Alias|Extension)\s+|$)/gi
    return Array.from(packageSection.matchAll(rowPattern), (match) => {
      const command = match[2]
      const installed = installedCommands.has(command)
      return {
        armory: match[1],
        command,
        version: match[3],
        type: match[4],
        help: match[5].trim(),
        installed,
        installedStr: installed ? 'Installed' : 'Not Installed',
      }
    })
  }

  function parseArmoryWarnings(output) {
    const lines = stripTerminalFormatting(output)
      .split('\n')
      .map((line) => line.trim())
      .filter((line) => line.includes('[!]') || /failed to (?:parse|download)/i.test(line))
    return [...new Set(lines)].join('\n')
  }

  async function installPackage(pkg, force = false) {
    installing = pkg.command
    try {
      const armory = pkg.armory ? ' --armory ' + quote(pkg.armory) : ''
      const forceFlag = force ? ' --force' : ''
      const output = await RunSessionCommand(
        '',
        'armory install' + forceFlag + armory + ' ' + quote(pkg.command),
      )
      const cleaned = stripTerminalFormatting(output).trim()
      if (/\b(?:could not install|failed to install|no package|package not found)\b/i.test(cleaned)) {
        await dialog.alert(cleaned, 'Armory Install Error')
      }
      await loadArmory(true)
    } catch (err) {
      await dialog.alert(
        errorMessage(err, 'Failed to install package: '),
        'Armory Install Error',
      )
    } finally {
      installing = ''
    }
  }

  async function installAll() {
    const pending = packages.filter((pkg) => !pkg.installed)
    if (pending.length === 0) {
      await dialog.alert('All available packages are already installed.', 'Armory')
      return
    }
    if (!(await dialog.confirm(
      'Install all ' + pending.length + ' available packages?',
      'Install All Packages',
    ))) return

    installing = '__all__'
    const failures = []
    try {
      for (const pkg of pending) {
        const armory = pkg.armory ? ' --armory ' + quote(pkg.armory) : ''
        try {
          const output = await RunSessionCommand(
            '',
            'armory install' + armory + ' ' + quote(pkg.command),
          )
          const cleaned = stripTerminalFormatting(output).trim()
          if (/\b(?:could not install|failed to install|no package|package not found)\b/i.test(cleaned)) {
            failures.push(pkg.command + ': ' + cleaned)
          }
        } catch (err) {
          failures.push(pkg.command + ': ' + errorMessage(err))
        }
      }
      await loadArmory(true)
      if (failures.length > 0) {
        await dialog.alert(
          (pending.length - failures.length) + ' installed, ' + failures.length + ' failed.\n\n' + failures.join('\n\n'),
          'Armory Install Results',
        )
      }
    } finally {
      installing = ''
    }
  }
</script>

<Panel {onclose} title="Armory / Script Manager" icon="shield" size="5xl">
  <Toolbar class="justify-end">
    <Button
      color="primary"
      size="sm"
      icon="download"
      onclick={installAll}
      disabled={loading || installing !== '' || packages.every((pkg) => pkg.installed)}
    >
      {installing === '__all__' ? 'Installing All...' : 'Install All'}
    </Button>
    <Button
      color="dark"
      size="sm"
      icon="refresh"
      onclick={() => loadArmory(true)}
      disabled={loading || installing !== ''}
    >
      Refresh
    </Button>
  </Toolbar>

  {#if warning}
    <div class="flex gap-2 mb-3 p-2 border border-yellow-700/50 rounded bg-yellow-500/10 text-yellow-500 text-sm">
      <Icon name="alert-triangle" size={16} class="shrink-0" />
      <pre class="m-0 whitespace-pre-wrap overflow-wrap-anywhere font-inherit">{warning}</pre>
    </div>
  {/if}

  <div class="h-vh-60 flex flex-col">
    <DataTable
      data={packages}
      columns={tableColumns}
      keyField="command"
      defaultSort={{key: 'command', dir: 'asc'}}
      error={error && packages.length === 0 ? error : null}
      emptyState={loading
        ? { icon: 'shield', title: 'Fetching packages…', description: 'Reading available packages from the Armory.' }
        : { icon: 'shield', title: 'No packages', description: 'Armory returned no packages. Try refreshing.' }}
    >
      {#snippet children(pkg, col)}
          {#if col.key === 'armory'}
            <strong>{pkg.armory}</strong>
          {:else if col.key === 'command'}
            <span class="font-mono">{pkg.command}</span>
          {:else if col.key === 'version'}
            <span class="font-mono">{pkg.version}</span>
          {:else if col.key === 'installedStr'}
            {#if pkg.installed}
              <span class="text-success-500">Installed</span>
            {:else}
              <span class="text-fg-muted">Not Installed</span>
            {/if}
          {:else if col.key === 'action'}
            {#if pkg.installed}
              <Button
                color="dark"
                size="xs"
                onclick={() => installPackage(pkg, true)}
                disabled={installing !== ''}
              >
                {installing === pkg.command ? 'Reinstalling...' : 'Reinstall'}
              </Button>
            {:else if installing === pkg.command}
              <Button color="primary" size="xs" icon="loader" disabled>Installing...</Button>
            {:else}
              <Button
                color="primary"
                size="xs"
                onclick={() => installPackage(pkg)}
                disabled={installing !== ''}
              >Install</Button>
            {/if}
          {:else}
            {pkg[col.key]}
          {/if}
        {/snippet}
      </DataTable>
    </div>
</Panel>
