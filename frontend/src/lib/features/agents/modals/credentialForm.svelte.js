import { credentialFields } from '$utils/credentials.js'

// createCredentialForm owns the credential state shared by credential-based
// command modals and resets it from the modal's saved initial values.
export function createCredentialForm() {
  const fields = $state(credentialFields())

  function reset(values = {}) {
    Object.assign(fields, credentialFields(values))
  }

  return { fields, reset }
}
