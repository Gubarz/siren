<script>
  import { quote } from '../../../../utils/shell.js'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import SelectField from '../../../../components/forms/SelectField.svelte'
  import CredentialPicker from '../CredentialPicker.svelte'
  import CommandModalFrame from './CommandModalFrame.svelte'
  import { createCredentialForm } from '../credentialForm.svelte.js'

  let { open = $bindable(false), onexecute, ...rest } = $props()

  const credentials = createCredentialForm()
  let logonType = $state('LOGON_NEW_CREDENTIALS')

  function resetForm(values) {
    credentials.reset(values)
    logonType = values['logon-type'] || 'LOGON_NEW_CREDENTIALS'
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['make-token']
    if (credentials.fields.username) parts.push('--username', quote(credentials.fields.username))
    if (credentials.fields.password) parts.push('--password', quote(credentials.fields.password))
    if (credentials.fields.domain) parts.push('--domain', quote(credentials.fields.domain))
    if (logonType) parts.push('--logon-type', logonType)
    if (credentials.fields.timeout) parts.push('--timeout', String(credentials.fields.timeout))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<CommandModalFrame
  bind:open
  title="Make Token"
  size="xl"
  {cmdPreview}
  commandPath="make-token"
  currentValues={{ username: credentials.fields.username, domain: credentials.fields.domain, 'logon-type': logonType }}
  onapply={(values) => {
    if (values['username'] != null) credentials.fields.username = values['username']
    if (values['domain'] != null) credentials.fields.domain = values['domain']
    if (values['logon-type'] != null) logonType = values['logon-type']
  }}
  primaryLabel="Make Token"
  onprimary={execute}
  primaryDisabled={!credentials.fields.username || !credentials.fields.password}
  onreset={resetForm}
  {...rest}
>
  <p class="text-fg-muted text-sm mb-4">
    Create a new access token from clear-text credentials. Typical use: <em>net-only</em> logon so subsequent network commands authenticate as the given user (Kerberos / SMB / WinRM) without changing local identity.
    Credentials are validated locally.
  </p>

  <CredentialPicker bind:username={credentials.fields.username} bind:password={credentials.fields.password} bind:domain={credentials.fields.domain} bind:timeout={credentials.fields.timeout} />

  <CollapsibleGroup title="Advanced" open={false}>
    <SelectField
      bind:value={logonType}
      label="Logon type"
      options={[
        { value: 'LOGON_NEW_CREDENTIALS', label: 'NEW_CREDENTIALS (net-only, quietest)' },
        { value: 'LOGON_INTERACTIVE', label: 'INTERACTIVE' },
        { value: 'LOGON_NETWORK', label: 'NETWORK' },
        { value: 'LOGON_BATCH', label: 'BATCH' },
        { value: 'LOGON_SERVICE', label: 'SERVICE' },
        { value: 'LOGON_NETWORK_CLEARTEXT', label: 'NETWORK_CLEARTEXT' },
      ]}
      description="NEW_CREDENTIALS (runas /netonly) is the default — validates only on remote access, doesn't log a local logon event."
    />
  </CollapsibleGroup>
</CommandModalFrame>
