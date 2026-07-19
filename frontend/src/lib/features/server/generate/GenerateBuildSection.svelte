<script>
  import { onMount } from 'svelte'
  import CollapsibleGroup from '$components/forms/CollapsibleGroup.svelte'
  import CheckboxField from '$components/forms/CheckboxField.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import { trafficEncoders as trafficEncoderResource } from '$stores/resources/trafficEncoders.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(trafficEncoderResource)

  let {
    obfuscateSymbols = $bindable(true),
    sgnEnabled = $bindable(false),
    netGoEnabled = $bindable(false),
    evasion = $bindable(false),
    runAtLoad = $bindable(false),
    debug = $bindable(false),
    trafficEncodersEnabled = $bindable(false),
    trafficEncoders = $bindable([]),
  } = $props()

  let encoderList = $derived($trafficEncoderResource.data || [])

  onMount(() => {
    trafficEncoderResource.refresh()
  })

  function setEncoder(name, checked) {
    if (checked) {
      trafficEncoders = [...new Set([...trafficEncoders, name])]
      trafficEncodersEnabled = true
    } else {
      trafficEncoders = trafficEncoders.filter((current) => current !== name)
      if (trafficEncoders.length === 0) trafficEncodersEnabled = false
    }
  }
</script>

<CollapsibleGroup title="Evasion &amp; build" open={false}>
  <CheckboxField bind:checked={obfuscateSymbols} label="Obfuscate Go symbols" description="Strip identifying strings from the binary (recommended, slower build)" />
  <CheckboxField bind:checked={sgnEnabled} label="SGN shellcode encoding" description="Encode shellcode with Shikata Ga Nai (shellcode format only)" />
  <CheckboxField bind:checked={netGoEnabled} label="NetGo (pure-Go networking)" description="Use Go's pure-Go DNS resolver instead of libc - avoids CGO and looks more legit" />
  <CheckboxField bind:checked={evasion} label="Compile-time evasion features" description="Extra anti-analysis / anti-EDR compile-time features (varies by OS)" />
  <CheckboxField bind:checked={runAtLoad} label="Run at load (macOS/Linux)" description="Auto-execute the payload when the shared library is loaded" />
  <CheckboxField bind:checked={debug} label="Debug build" description="Includes debug logging in the binary - DO NOT ship to real targets" />
  <CheckboxField bind:checked={trafficEncodersEnabled} label="Enable selected WASM traffic encoders" description="Embed selected HTTP C2 traffic encoders in the implant" />
  {#if trafficEncodersEnabled}
    <div class="ml-5 grid gap-1 border-l border-line pl-3">
      {#each encoderList as encoder}
        <Checkbox
          checked={trafficEncoders.includes(encoder.name)}
          label={encoder.name}
          onchange={(event) => setEncoder(encoder.name, event.currentTarget.checked)}
        />
      {:else}
        <span class="text-xs text-fg-muted">No traffic encoders installed.</span>
      {/each}
    </div>
  {/if}
</CollapsibleGroup>
