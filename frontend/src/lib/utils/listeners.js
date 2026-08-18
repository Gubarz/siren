export function listenerProtocol(job) {
  if (!job) return ''
  const candidates = [
    job.Protocol ?? job.protocol,
    job.Name ?? job.name,
    job.Description ?? job.description,
  ].map((v) => String(v ?? '').toLowerCase())
  for (const text of candidates) {
    if (text.includes('mtls')) return 'mtls'
    if (text.includes('https')) return 'https'
    if (text.includes('http')) return 'http'
    if (text.includes('dns')) return 'dns'
    if (text.includes('wireguard') || /\bwg\b/.test(text)) return 'wg'
  }
  return ''
}

function isWildcardHost(host) {
  return ['', '0.0.0.0', '::', '[::]', '*'].includes(String(host || '').trim())
}

function extractHost(value) {
  const text = String(value ?? '').trim()
  if (!text) return ''
  try {
    const parsed = new URL(text)
    return parsed.hostname || ''
  } catch {
    // Most Sliver job descriptions are prose, not URLs.
  }
  const bracketed = text.match(/\[([^\]]+)\](?::\d+)?/)
  if (bracketed) return bracketed[1]
  const hostPort = text.match(/(?:^|\s)([a-z0-9_.:-]+):\d+(?:\s|$)/i)
  if (hostPort) return hostPort[1]
  return ''
}

export function listenerHost(job, fallback = '') {
  if (!job) return fallback || ''
  const domains = job.Domains ?? job.domains ?? []
  const firstDomain = Array.isArray(domains) ? domains.find(Boolean) : ''
  if (firstDomain && !isWildcardHost(firstDomain)) return firstDomain

  const candidates = [
    job.Host ?? job.host,
    job.BindHost ?? job.bindHost,
    job.BindAddr ?? job.bindAddr,
    job.ListenAddr ?? job.listenAddr,
    job.Description ?? job.description,
  ]
  for (const candidate of candidates) {
    const host = extractHost(candidate) || String(candidate ?? '').trim()
    if (host && !isWildcardHost(host)) return host
  }
  return fallback || ''
}

export function formatListenerC2(listener, fallbackHost = '127.0.0.1') {
  if (!listener) return `mtls://${fallbackHost}:443`
  const proto = listener.protocol || 'mtls'
  const host = listener.host || fallbackHost || '127.0.0.1'
  if (proto === 'dns') {
    const domain = (listener.domains || []).find(Boolean) || host
    return `dns://${domain}`
  }
  return `${proto}://${host}:${listener.port || 443}`
}
