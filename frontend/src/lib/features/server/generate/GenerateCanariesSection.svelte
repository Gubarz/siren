<script>
  import CollapsibleGroup from '$components/forms/CollapsibleGroup.svelte'
  import TextField from '$components/forms/TextField.svelte'

  let {
    canaryDomainsText = $bindable(''),
  } = $props()
</script>

<CollapsibleGroup title="Canaries" open={false}>
  <TextField
    bind:value={canaryDomainsText}
    label="Canary domains (comma-separated)"
    placeholder="something.thisisa.canary.example.com"
    description="If the sample is analyzed and these are resolved, the teamserver gets a canary event"
  />
</CollapsibleGroup>
