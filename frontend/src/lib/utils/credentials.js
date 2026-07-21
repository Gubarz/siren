export function plaintextCredentials(items = []) {
  return items.filter((credential) => credentialUsername(credential) && credentialPlaintext(credential))
}

export function credentialKey(credential, index = 0) {
  return credential?.ID || credential?.id || `${credentialUsername(credential)}:${index}`
}

export function credentialUsername(credential) {
  return stringField(credential, 'Username', 'username')
}

export function credentialPlaintext(credential) {
  return stringField(credential, 'Plaintext', 'plaintext')
}

export function credentialCollection(credential) {
  return stringField(credential, 'Collection', 'collection')
}

export function credentialPickerOptions(items = []) {
  return plaintextCredentials(items).map((credential, index) => ({
    value: credentialKey(credential, index),
    label: credentialLabel(credential),
  }))
}

export function credentialLabel(credential) {
  const username = credentialUsername(credential) || 'credential'
  const collection = credentialCollection(credential)
  return collection ? `${username} (${collection})` : username
}

export function credentialLoginFields(credential) {
  const parsed = parseCredentialUsername(credentialUsername(credential))
  return {
    username: parsed.username,
    domain: parsed.domain,
    password: credentialPlaintext(credential),
  }
}

export function parseCredentialUsername(value = '') {
  const username = String(value || '').trim()
  if (!username) return { username: '', domain: '' }

  const slash = username.lastIndexOf('\\')
  if (slash > 0 && slash < username.length - 1) {
    return {
      username: username.slice(slash + 1),
      domain: username.slice(0, slash),
    }
  }

  const at = username.lastIndexOf('@')
  if (at > 0 && at < username.length - 1) {
    return {
      username: username.slice(0, at),
      domain: username.slice(at + 1),
    }
  }

  return { username, domain: '' }
}

function stringField(source, pascal, camel) {
  const value = source?.[pascal] ?? source?.[camel] ?? ''
  return typeof value === 'string' ? value.trim() : String(value || '').trim()
}
