<script>
  import CollapsibleGroup from '$components/forms/CollapsibleGroup.svelte'
  import CheckboxField from '$components/forms/CheckboxField.svelte'
  import TextField from '$components/forms/TextField.svelte'

  let {
    limitDomainJoined = $bindable(false),
    limitHostname = $bindable(''),
    limitUsername = $bindable(''),
    limitDatetime = $bindable(''),
    limitFileExists = $bindable(''),
    limitLocale = $bindable(''),
  } = $props()
</script>

<CollapsibleGroup title="Execution constraints" open={false}>
  <p class="text-fg-muted text-xs mb-2">
    Implant refuses to run unless these constraints are met on the target host - useful for scoped engagements.
  </p>
  <CheckboxField bind:checked={limitDomainJoined} label="Only run on domain-joined hosts" />
  <TextField bind:value={limitHostname} label="Hostname must match" placeholder="TARGET-HOST01" />
  <TextField bind:value={limitUsername} label="Username must match" placeholder="CORP\\jsmith" />
  <TextField bind:value={limitDatetime} label="Do not run after (ISO date)" placeholder="2026-12-31T00:00:00Z" description="Kill date - implant self-destructs after this timestamp" />
  <TextField bind:value={limitFileExists} label="Only run if this file exists" placeholder="C:\\Windows\\System32\\wbem\\wbemtest.exe" />
  <TextField bind:value={limitLocale} label="Locale must match" placeholder="en-US" />
</CollapsibleGroup>
