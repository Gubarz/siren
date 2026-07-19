<script>
  import CollapsibleGroup from '$components/forms/CollapsibleGroup.svelte'
  import TextField from '$components/forms/TextField.svelte'
  import SelectField from '$components/forms/SelectField.svelte'

  let {
    name = $bindable(''),
    goos = $bindable('windows'),
    goarch = $bindable('amd64'),
    format = $bindable('exe'),
  } = $props()

  const OS_OPTIONS = [
    { value: 'windows', label: 'Windows' },
    { value: 'linux', label: 'Linux' },
    { value: 'darwin', label: 'macOS (darwin)' },
  ]
  const ARCH_OPTIONS = [
    { value: 'amd64', label: 'amd64 (64-bit)' },
    { value: '386', label: '386 (32-bit)' },
    { value: 'arm64', label: 'arm64' },
  ]
  const FORMAT_OPTIONS = [
    { value: 'exe', label: 'Executable (.exe / ELF / Mach-O)' },
    { value: 'shared', label: 'Shared library (.dll / .so / .dylib)' },
    { value: 'shellcode', label: 'Shellcode (raw)' },
    { value: 'service', label: 'Windows service binary' },
  ]
</script>

<CollapsibleGroup title="Target" open={true}>
  <TextField
    bind:value={name}
    label="Implant name"
    placeholder="Auto-generated if blank (e.g. WITTY_CANDLE)"
    description="Used as the on-target process/service name and the loot/build filename"
  />
  <div class="grid grid-cols-3 gap-3">
    <SelectField bind:value={goos} label="OS" options={OS_OPTIONS} />
    <SelectField bind:value={goarch} label="Architecture" options={ARCH_OPTIONS} />
    <SelectField bind:value={format} label="Format" options={FORMAT_OPTIONS} />
  </div>
</CollapsibleGroup>
