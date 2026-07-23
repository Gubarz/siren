<script>
  import CollapsibleGroup from '$components/forms/CollapsibleGroup.svelte'
  import TextField from '$components/forms/TextField.svelte'
  import SelectField from '$components/forms/SelectField.svelte'
  import { credentials } from '$stores/resources/credentials.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'
  import {
    credentialKey,
    credentialLoginFields,
    credentialPickerOptions,
    plaintextCredentials,
  } from '$utils/credentials.js'

  useResource(credentials)

  let {
    username = $bindable(''),
    password = $bindable(''),
    domain = $bindable(''),
    timeout = $bindable(''),
  } = $props()

  let selectedCredential = $state('')
  let usableCredentials = $derived(plaintextCredentials(credentials.data || []))
  let credentialOptions = $derived(credentialPickerOptions(credentials.data || []))

  function applyCredential(id) {
    selectedCredential = id
    const credential = usableCredentials.find((item, index) => credentialKey(item, index) === id)
    if (!credential) return
    const fields = credentialLoginFields(credential)
    username = fields.username
    password = fields.password
    domain = fields.domain
  }

  export function credentialFields(values = {}) {
    return {
      username: values['username'] || '',
      password: values['password'] || '',
      domain: values['domain'] || '',
      timeout: values['timeout'] || '',
    }
  }
</script>

<CollapsibleGroup title="Credentials" defaultOpen>
  {#if credentialOptions.length > 0}
    <SelectField
      bind:value={selectedCredential}
      label="Select from credential store"
      options={[{ value: '', label: '— None (enter manually) —' }, ...credentialOptions]}
      description="Pick an existing credential to auto-fill the fields below"
      onchange={(e) => {
        if (e?.target?.value !== undefined) applyCredential(e.target.value)
      }}
    />
  {/if}

  <TextField bind:value={username} label="Username" placeholder="DOMAIN\username or user@domain" />

  <TextField bind:value={password} label="Password" class="security-field" />

  <TextField bind:value={domain} label="Domain" placeholder="Leave blank to use the current domain" />

  <TextField bind:value={timeout} label="Timeout (seconds)" type="number" placeholder="Default" />
</CollapsibleGroup>
