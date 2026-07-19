import { quote } from '../../../utils/shell.js'

export function buildGenerateRequest(values) {
  const canaries = values.canaryDomainsText
    .split(/[\s,]+/)
    .map((domain) => domain.trim())
    .filter(Boolean)
  const trafficEncoders = (values.trafficEncoders || []).filter(Boolean)

  return {
    name: values.name,
    goos: values.goos,
    goarch: values.goarch,
    format: values.format,
    c2Urls: values.c2Urls.filter(Boolean),
    httpC2ConfigName: values.httpC2ConfigName,
    isBeacon: values.isBeacon,
    beaconInterval: Number(values.beaconInterval) || 0,
    beaconJitter: Number(values.beaconJitter) || 0,
    reconnectInterval: Number(values.reconnectInterval) || 0,
    pollTimeout: Number(values.pollTimeout) || 0,
    maxConnectionErrors: Number(values.maxConnectionErrors) || 0,
    connectionStrategy: values.connectionStrategy,
    debug: values.debug,
    evasion: values.evasion,
    obfuscateSymbols: values.obfuscateSymbols,
    sgnEnabled: values.sgnEnabled,
    netGoEnabled: values.netGoEnabled,
    runAtLoad: values.runAtLoad,
    trafficEncodersEnabled: values.trafficEncodersEnabled && trafficEncoders.length > 0,
    trafficEncoders,
    canaryDomains: canaries,
    limitDomainJoined: values.limitDomainJoined,
    limitHostname: values.limitHostname,
    limitUsername: values.limitUsername,
    limitDatetime: values.limitDatetime,
    limitFileExists: values.limitFileExists,
    limitLocale: values.limitLocale,
  }
}

export function buildGenerateCommandPreview(values) {
  const parts = ['generate']
  parts.push('--os', values.goos)
  parts.push('--arch', values.goarch)
  parts.push('--format', values.format)

  if (values.name) parts.push('--name', quote(values.name))

  for (const url of values.c2Urls) {
    if (!url) continue
    const lower = url.toLowerCase()
    if (lower.startsWith('mtls://')) parts.push('--mtls', quote(url.replace(/^mtls:\/\//i, '')))
    else if (lower.startsWith('https://') || lower.startsWith('http://')) parts.push('--http', quote(url))
    else if (lower.startsWith('dns://')) parts.push('--dns', quote(url.replace(/^dns:\/\//i, '')))
    else if (lower.startsWith('wg://')) parts.push('--wg', quote(url.replace(/^wg:\/\//i, '')))
    else if (lower.startsWith('tcp-pivot://')) parts.push('--tcp-pivot', quote(url.replace(/^tcp-pivot:\/\//i, '')))
    else if (lower.startsWith('namedpipe://')) parts.push('--named-pipe', quote(url.replace(/^namedpipe:\/\//i, '')))
  }

  if (values.httpC2ConfigName && values.httpC2ConfigName !== 'default') {
    parts.push('--http-c2-profile', quote(values.httpC2ConfigName))
  }
  if (values.isBeacon) {
    parts.push('beacon')
    if (values.beaconInterval) parts.push('--seconds', String(values.beaconInterval))
    if (values.beaconJitter) parts.push('--jitter', String(values.beaconJitter))
  }
  if (values.reconnectInterval && values.reconnectInterval !== 60) parts.push('--reconnect', String(values.reconnectInterval))
  if (values.pollTimeout && values.pollTimeout !== 360) parts.push('--poll-timeout', String(values.pollTimeout))
  if (values.maxConnectionErrors && values.maxConnectionErrors !== 1000) parts.push('--max-errors', String(values.maxConnectionErrors))
  if (values.connectionStrategy) parts.push('--strategy', values.connectionStrategy)
  if (values.debug) parts.push('--debug')
  if (values.evasion) parts.push('--evasion')
  if (!values.obfuscateSymbols) parts.push('--skip-symbols')
  if (values.sgnEnabled) parts.push('--disable-sgn=false')
  if (values.netGoEnabled) parts.push('--netgo')
  if (values.runAtLoad) parts.push('--run-at-load')
  if (values.trafficEncodersEnabled && values.trafficEncoders?.length) {
    parts.push('--traffic-encoders', quote(values.trafficEncoders.join(',')))
  }
  if (values.canaryDomainsText.trim()) parts.push('--canary', quote(values.canaryDomainsText.replace(/\s+/g, '')))
  if (values.limitDomainJoined) parts.push('--limit-domainjoined')
  if (values.limitHostname) parts.push('--limit-hostname', quote(values.limitHostname))
  if (values.limitUsername) parts.push('--limit-username', quote(values.limitUsername))
  if (values.limitDatetime) parts.push('--limit-datetime', quote(values.limitDatetime))
  if (values.limitFileExists) parts.push('--limit-fileexists', quote(values.limitFileExists))
  if (values.limitLocale) parts.push('--limit-locale', quote(values.limitLocale))

  return parts.join(' ')
}
