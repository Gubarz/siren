<script>
  import CollapsibleGroup from '$components/forms/CollapsibleGroup.svelte'
  import CheckboxField from '$components/forms/CheckboxField.svelte'
  import TextField from '$components/forms/TextField.svelte'
  import SelectField from '$components/forms/SelectField.svelte'

  let {
    isBeacon = $bindable(false),
    beaconInterval = $bindable(60),
    beaconJitter = $bindable(30),
    reconnectInterval = $bindable(60),
    pollTimeout = $bindable(360),
    maxConnectionErrors = $bindable(1000),
    connectionStrategy = $bindable(''),
  } = $props()

  const STRATEGY_OPTIONS = [
    { value: '', label: 'Default (sequential)' },
    { value: 's', label: 'Sequential' },
    { value: 'r', label: 'Random' },
  ]
</script>

<CollapsibleGroup title="Behavior" open={true}>
  <CheckboxField
    bind:checked={isBeacon}
    label="Beacon mode"
    description="Polls C2 on an interval instead of holding a persistent session. Quieter, higher-latency."
  />
  {#if isBeacon}
    <div class="grid grid-cols-2 gap-3">
      <TextField bind:value={beaconInterval} label="Interval (seconds)" type="number" description="Base sleep time between check-ins" />
      <TextField bind:value={beaconJitter} label="Jitter (seconds)" type="number" description="Randomization +/-jitter on each check-in" />
    </div>
  {/if}
  <div class="grid grid-cols-2 gap-3">
    <TextField bind:value={reconnectInterval} label="Reconnect interval (s)" type="number" description="How long to wait before retrying a dropped connection" />
    <TextField bind:value={pollTimeout} label="Poll timeout (s)" type="number" description="HTTP long-poll timeout" />
  </div>
  <div class="grid grid-cols-2 gap-3">
    <TextField bind:value={maxConnectionErrors} label="Max connection errors" type="number" description="Give up after this many consecutive failures (implant self-destructs)" />
    <SelectField bind:value={connectionStrategy} label="Connection strategy" options={STRATEGY_OPTIONS} description="How to pick between C2 URLs" />
  </div>
</CollapsibleGroup>
