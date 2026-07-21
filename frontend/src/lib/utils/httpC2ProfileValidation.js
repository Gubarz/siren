const ROOT_FIELDS = {
  name: ['Name', 'name'],
  serverConfig: ['ServerConfig', 'serverConfig', 'server_config'],
  implantConfig: ['ImplantConfig', 'implantConfig', 'implant_config'],
}

const IMPLANT_FIELDS = {
  userAgent: ['UserAgent', 'userAgent', 'user_agent'],
  nonceQueryArgChars: ['NonceQueryArgChars', 'nonceQueryArgChars', 'nonce_query_args'],
  nonceQueryLength: ['NonceQueryLength', 'nonceQueryLength', 'nonce_query_length'],
  nonceMode: ['NonceMode', 'nonceMode', 'nonce_mode'],
  extraUrlParameters: ['ExtraURLParameters', 'extraURLParameters', 'extraUrlParameters', 'url_parameters', 'URLParameters'],
  headers: ['Headers', 'headers'],
  minFileGen: ['MinFileGen', 'minFileGen', 'min_files'],
  maxFileGen: ['MaxFileGen', 'maxFileGen', 'max_files'],
  minPathGen: ['MinPathGen', 'minPathGen', 'min_paths'],
  maxPathGen: ['MaxPathGen', 'maxPathGen', 'max_paths'],
  minPathLength: ['MinPathLength', 'minPathLength', 'min_path_length'],
  maxPathLength: ['MaxPathLength', 'maxPathLength', 'max_path_length'],
  extensions: ['Extensions', 'extensions'],
  pathSegments: ['PathSegments', 'pathSegments'],
  files: ['Files', 'files'],
  paths: ['Paths', 'paths'],
}

const SERVER_FIELDS = {
  headers: ['Headers', 'headers'],
  cookies: ['Cookies', 'cookies'],
}

const ITEM_FIELDS = {
  name: ['Name', 'name'],
  value: ['Value', 'value'],
  method: ['Method', 'method'],
  probability: ['Probability', 'probability'],
  isFile: ['IsFile', 'isFile', 'is_file'],
}

const METHODS = new Set(['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'])
const NONCE_MODES = new Set(['Url', 'UrlParam'])
const DEFAULT_HOST = 'example.com'

export function validateHTTPC2ProfileText(text) {
  const result = emptyResult()
  const trimmed = String(text || '').trim()
  if (!trimmed) {
    result.errors.push('Profile JSON is empty.')
    return finish(result)
  }

  let profile
  try {
    profile = JSON.parse(trimmed)
  } catch (err) {
    result.errors.push(`Invalid JSON: ${err.message}`)
    return finish(result)
  }

  result.validJson = true
  result.profile = profile
  validateProfile(profile, result)
  result.sampleRequest = buildSampleRequest(profile)
  return finish(result)
}

function emptyResult() {
  return {
    validJson: false,
    canSave: false,
    profile: null,
    errors: [],
    warnings: [],
    summary: {
      cookies: 0,
      extensions: 0,
      pathSegments: 0,
      headers: 0,
      urlParameters: 0,
    },
    sampleRequest: '',
  }
}

function finish(result) {
  result.canSave = result.validJson && result.errors.length === 0
  return result
}

function validateProfile(profile, result) {
  if (!profile || typeof profile !== 'object' || Array.isArray(profile)) {
    result.errors.push('Profile root must be a JSON object.')
    return
  }

  const name = stringValue(field(profile, ROOT_FIELDS.name))
  const server = objectValue(field(profile, ROOT_FIELDS.serverConfig))
  const implant = objectValue(field(profile, ROOT_FIELDS.implantConfig))

  if (!name) result.errors.push('Profile Name is required.')
  if (!server) result.errors.push('ServerConfig is required.')
  if (!implant) result.errors.push('ImplantConfig is required.')
  if (!server || !implant) return

  const cookies = list(field(server, SERVER_FIELDS.cookies)).map(cookieName).filter(Boolean)
  const extensions = list(field(implant, IMPLANT_FIELDS.extensions)).map(stringValue).filter(Boolean)
  const serverHeaders = list(field(server, SERVER_FIELDS.headers))
  const implantHeaders = list(field(implant, IMPLANT_FIELDS.headers))
  const urlParameters = list(field(implant, IMPLANT_FIELDS.extraUrlParameters))
  const pathSegments = normalizedPathSegments(implant)

  result.summary = {
    cookies: cookies.length,
    extensions: extensions.length,
    pathSegments: pathSegments.length,
    headers: serverHeaders.length + implantHeaders.length,
    urlParameters: urlParameters.length,
  }

  if (cookies.length < 1) result.errors.push('ServerConfig.Cookies must include at least one cookie.')
  warnDuplicates(cookies, 'cookie', result)

  if (extensions.length < 1) result.errors.push('ImplantConfig.Extensions must include at least one extension.')
  for (const extension of extensions) {
    if (extension.startsWith('.')) {
      result.warnings.push(`Extension "${extension}" starts with a dot; Sliver strips leading dots during validation.`)
    }
    if (/[^a-zA-Z0-9._-]/.test(extension)) {
      result.warnings.push(`Extension "${extension}" contains characters Sliver will strip from generated file names.`)
    }
  }
  warnDuplicates(extensions.map((value) => value.toLowerCase()), 'extension', result)

  const userAgent = stringValue(field(implant, IMPLANT_FIELDS.userAgent))
  if (userAgent.includes('`')) {
    result.errors.push('ImplantConfig.UserAgent cannot contain a backtick because it breaks implant compilation.')
  }

  validateRangePair(implant, IMPLANT_FIELDS.minFileGen, IMPLANT_FIELDS.maxFileGen, 'file count', 1, result)
  validateRangePair(implant, IMPLANT_FIELDS.minPathGen, IMPLANT_FIELDS.maxPathGen, 'path count', 0, result)
  validateRangePair(implant, IMPLANT_FIELDS.minPathLength, IMPLANT_FIELDS.maxPathLength, 'path length', 1, result)

  const nonceLength = numberValue(field(implant, IMPLANT_FIELDS.nonceQueryLength))
  if (nonceLength !== null && nonceLength < 1) {
    result.warnings.push('NonceQueryLength is below 1; Sliver will generate very short nonce values.')
  }
  const nonceMode = stringValue(field(implant, IMPLANT_FIELDS.nonceMode))
  if (nonceMode && !NONCE_MODES.has(nonceMode)) {
    result.warnings.push(`NonceMode "${nonceMode}" is unusual; expected Url or UrlParam.`)
  }

  if (pathSegments.length < 1) {
    result.warnings.push('No path segments were found; generated requests may have little path variation.')
  } else {
    const files = pathSegments.filter((segment) => segment.isFile)
    const paths = pathSegments.filter((segment) => !segment.isFile)
    if (files.length < 1) result.warnings.push('No file path segments were found.')
    if (paths.length < 1) result.warnings.push('No directory path segments were found.')
    warnDuplicates(pathSegments.map((segment) => segment.value.toLowerCase()).filter(Boolean), 'path segment', result)
  }

  validateNameValueList(serverHeaders, 'server header', result)
  validateNameValueList(implantHeaders, 'implant header', result)
  validateNameValueList(urlParameters, 'URL parameter', result)
  warnHeaderCollisions([...serverHeaders, ...implantHeaders], result)
}

function validateRangePair(source, minNames, maxNames, label, minimum, result) {
  const min = numberValue(field(source, minNames))
  const max = numberValue(field(source, maxNames))
  if (min !== null && min < minimum) {
    result.warnings.push(`Minimum ${label} is below ${minimum}; Sliver will coerce it upward.`)
  }
  if (min !== null && max !== null && max < min) {
    result.warnings.push(`Maximum ${label} is lower than the minimum; Sliver will coerce it upward.`)
  }
}

function validateNameValueList(items, label, result) {
  for (const [index, item] of items.entries()) {
    const name = stringValue(field(item, ITEM_FIELDS.name))
    const value = stringValue(field(item, ITEM_FIELDS.value))
    const method = stringValue(field(item, ITEM_FIELDS.method)).toUpperCase()
    const probability = numberValue(field(item, ITEM_FIELDS.probability))
    const location = `${label} #${index + 1}`

    if (!name) result.warnings.push(`${location} is missing a name.`)
    if (!value) result.warnings.push(`${location} is missing a value.`)
    if (method && !METHODS.has(method)) result.warnings.push(`${location} uses unusual method "${method}".`)
    if (probability !== null && (probability < 0 || probability > 100)) {
      result.warnings.push(`${location} probability should be between 0 and 100.`)
    }
  }
}

function warnHeaderCollisions(headers, result) {
  const seen = new Map()
  for (const header of headers) {
    const name = stringValue(field(header, ITEM_FIELDS.name)).toLowerCase()
    if (!name) continue
    const method = stringValue(field(header, ITEM_FIELDS.method)).toUpperCase() || '*'
    const key = `${method}:${name}`
    seen.set(key, (seen.get(key) || 0) + 1)
  }
  for (const [key, count] of seen.entries()) {
    if (count > 1) {
      const [method, name] = key.split(':')
      result.warnings.push(`Duplicate ${method} header "${name}" appears ${count} times.`)
    }
  }
}

function warnDuplicates(values, label, result) {
  const seen = new Map()
  for (const value of values) {
    const key = String(value || '').trim()
    if (!key) continue
    seen.set(key, (seen.get(key) || 0) + 1)
  }
  for (const [value, count] of seen.entries()) {
    if (count > 1) result.warnings.push(`Duplicate ${label} "${value}" appears ${count} times.`)
  }
}

function buildSampleRequest(profile) {
  const server = objectValue(field(profile, ROOT_FIELDS.serverConfig))
  const implant = objectValue(field(profile, ROOT_FIELDS.implantConfig))
  if (!server || !implant) return ''

  const segments = normalizedPathSegments(implant)
  const directory = segments.find((segment) => !segment.isFile)?.value || 'api'
  const file = segments.find((segment) => segment.isFile)?.value || `poll.${firstExtension(implant)}`
  const nonce = sampleNonce(implant)
  const nonceMode = stringValue(field(implant, IMPLANT_FIELDS.nonceMode)) || 'UrlParam'
  const urlParams = list(field(implant, IMPLANT_FIELDS.extraUrlParameters))
    .filter((param) => appliesToGet(param))
    .map((param) => `${encodeURIComponent(stringValue(field(param, ITEM_FIELDS.name)))}=${encodeURIComponent(stringValue(field(param, ITEM_FIELDS.value)))}`)
    .filter((param) => !param.startsWith('='))
  const nonceName = firstNonceName(implant)

  let path = joinPath(directory, file)
  if (nonceMode === 'Url') {
    path = joinPath(path, nonce)
  } else {
    urlParams.unshift(`${encodeURIComponent(nonceName)}=${encodeURIComponent(nonce)}`)
  }

  const query = urlParams.length ? `?${urlParams.join('&')}` : ''
  const lines = [`GET ${path}${query} HTTP/1.1`, `Host: ${DEFAULT_HOST}`]
  const userAgent = stringValue(field(implant, IMPLANT_FIELDS.userAgent))
  if (userAgent) lines.push(`User-Agent: ${userAgent}`)

  const headers = [...list(field(implant, IMPLANT_FIELDS.headers)), ...list(field(server, SERVER_FIELDS.headers))]
  for (const header of headers.filter((item) => appliesToGet(item)).slice(0, 8)) {
    const name = stringValue(field(header, ITEM_FIELDS.name))
    const value = stringValue(field(header, ITEM_FIELDS.value))
    if (name && value) lines.push(`${name}: ${value}`)
  }

  const cookies = list(field(server, SERVER_FIELDS.cookies)).map(cookieName).filter(Boolean)
  if (cookies.length) lines.push(`Cookie: ${cookies.slice(0, 3).map((name) => `${name}=...`).join('; ')}`)
  return lines.join('\n')
}

function normalizedPathSegments(implant) {
  const rawSegments = list(field(implant, IMPLANT_FIELDS.pathSegments))
  if (rawSegments.length) {
    return rawSegments
      .map((segment) => ({
        isFile: Boolean(field(segment, ITEM_FIELDS.isFile)),
        value: stringValue(field(segment, ITEM_FIELDS.value)),
      }))
      .filter((segment) => segment.value)
  }

  const files = list(field(implant, IMPLANT_FIELDS.files)).map(stringValue).filter(Boolean)
  const paths = list(field(implant, IMPLANT_FIELDS.paths)).map(stringValue).filter(Boolean)
  return [
    ...files.map((value) => ({ isFile: true, value })),
    ...paths.map((value) => ({ isFile: false, value })),
  ]
}

function firstExtension(implant) {
  return list(field(implant, IMPLANT_FIELDS.extensions)).map(stringValue).find(Boolean) || 'js'
}

function firstNonceName(implant) {
  const chars = stringValue(field(implant, IMPLANT_FIELDS.nonceQueryArgChars)) || 'abcdefghijklmnopqrstuvwxyz'
  return chars.slice(0, 3) || 'n'
}

function sampleNonce(implant) {
  const chars = stringValue(field(implant, IMPLANT_FIELDS.nonceQueryArgChars)) || 'abcdefghijklmnopqrstuvwxyz'
  const length = Math.max(1, numberValue(field(implant, IMPLANT_FIELDS.nonceQueryLength)) || 1)
  return Array.from({ length }, (_, index) => chars[index % chars.length] || 'a').join('')
}

function appliesToGet(item) {
  const method = stringValue(field(item, ITEM_FIELDS.method)).toUpperCase()
  return !method || method === 'GET'
}

function cookieName(cookie) {
  if (typeof cookie === 'string') return cookie.trim()
  return stringValue(field(cookie, ITEM_FIELDS.name))
}

function joinPath(...parts) {
  const path = parts
    .map((part) => stringValue(part).replace(/^\/+|\/+$/g, ''))
    .filter(Boolean)
    .join('/')
  return `/${path || 'index'}`
}

function field(source, names) {
  if (!source || typeof source !== 'object') return undefined
  for (const name of names) {
    if (Object.prototype.hasOwnProperty.call(source, name)) return source[name]
  }
  return undefined
}

function list(value) {
  return Array.isArray(value) ? value : []
}

function objectValue(value) {
  return value && typeof value === 'object' && !Array.isArray(value) ? value : null
}

function stringValue(value) {
  return typeof value === 'string' ? value.trim() : value === null || value === undefined ? '' : String(value).trim()
}

function numberValue(value) {
  if (value === null || value === undefined || value === '') return null
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}
