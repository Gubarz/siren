<script>
  import { onMount } from 'svelte'
  import Modal from '$components/patterns/Modal.svelte'
  import GenerateBehaviorSection from './generate/GenerateBehaviorSection.svelte'
  import GenerateBuildSection from './generate/GenerateBuildSection.svelte'
  import GenerateBuilderSection from './generate/GenerateBuilderSection.svelte'
  import GenerateC2Section from './generate/GenerateC2Section.svelte'
  import GenerateCanariesSection from './generate/GenerateCanariesSection.svelte'
  import GenerateConstraintsSection from './generate/GenerateConstraintsSection.svelte'
  import GenerateModalActions from './generate/GenerateModalActions.svelte'
  import GenerateStatus from './generate/GenerateStatus.svelte'
  import GenerateSpoofSection from './generate/GenerateSpoofSection.svelte'
  import GenerateTargetSection from './generate/GenerateTargetSection.svelte'
  import { buildGenerateCommandPreview, buildGenerateRequest } from './generate/generateHelpers.js'
  import { GenerateImplantAdvanced, SaveProfileAdvanced, GetServerInfo } from '../../api/server.js'
  import { GenerateExternalBuild, GetExternalBuildConfig, SaveExternalBuild } from '../../api/builders.js'
  import { implantBuilds } from '$stores/resources/implantBuilds.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(implantBuilds)
  import { errorMessage } from '../../utils/errors.js'

  let {
    open = $bindable(false),
    onclose,
    initialValues = {},
  } = $props()

  let name = $state('')
  let goos = $state('windows')
  let goarch = $state('amd64')
  let format = $state('exe')

  let c2Urls = $state([''])
  let httpC2ConfigName = $state('default')

  let isBeacon = $state(false)
  let beaconInterval = $state(60)
  let beaconJitter = $state(30)
  let reconnectInterval = $state(60)
  let pollTimeout = $state(360)
  let maxConnectionErrors = $state(1000)
  let connectionStrategy = $state('')

  let debug = $state(false)
  let evasion = $state(false)
  let obfuscateSymbols = $state(true)
  let sgnEnabled = $state(false)
  let netGoEnabled = $state(false)
  let runAtLoad = $state(false)

  let trafficEncodersEnabled = $state(false)
  let trafficEncoders = $state([])
  let canaryDomainsText = $state('')

  let limitDomainJoined = $state(false)
  let limitHostname = $state('')
  let limitUsername = $state('')
  let limitDatetime = $state('')
  let limitFileExists = $state('')
  let limitLocale = $state('')

  let generating = $state(false)
  let savingProfile = $state(false)
  let resultPath = $state('')
  let error = $state('')
  let serverHost = $state('')
  let buildTarget = $state('server')
  let externalBuild = $state(null)
  let externalStatus = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    name = values.name || ''
    goos = values.goos || 'windows'
    goarch = values.goarch || 'amd64'
    format = values.format || 'exe'
    c2Urls = values.c2Urls?.length ? [...values.c2Urls] : ['']
    httpC2ConfigName = values.httpC2ConfigName || 'default'
    isBeacon = values.isBeacon ?? false
    beaconInterval = values.beaconInterval || 60
    beaconJitter = values.beaconJitter || 30
    reconnectInterval = values.reconnectInterval || 60
    pollTimeout = values.pollTimeout || 360
    maxConnectionErrors = values.maxConnectionErrors || 1000
    connectionStrategy = values.connectionStrategy || ''
    debug = values.debug || false
    evasion = values.evasion || false
    obfuscateSymbols = values.obfuscateSymbols ?? true
    sgnEnabled = values.sgnEnabled || false
    netGoEnabled = values.netGoEnabled || false
    runAtLoad = values.runAtLoad || false
    trafficEncodersEnabled = values.trafficEncodersEnabled || false
    trafficEncoders = values.trafficEncoders?.length ? [...values.trafficEncoders] : []
    canaryDomainsText = (values.canaryDomains || []).join(', ')
    limitDomainJoined = values.limitDomainJoined || false
    limitHostname = values.limitHostname || ''
    limitUsername = values.limitUsername || ''
    limitDatetime = values.limitDatetime || ''
    limitFileExists = values.limitFileExists || ''
    limitLocale = values.limitLocale || ''
    buildTarget = values.buildTarget || 'server'
    externalBuild = null
    externalStatus = ''
  }

  onMount(async () => {
    try {
      const info = await GetServerInfo()
      if (info?.host) {
        serverHost = info.host
        if (c2Urls.length === 1 && !c2Urls[0]) {
          c2Urls = [`mtls://${info.host}:${info.port || 443}`]
        }
      }
    } catch {
      // Leave the C2 field blank if server info is unavailable.
    }
  })

  let generateValues = $derived({
    name,
    goos,
    goarch,
    format,
    c2Urls,
    httpC2ConfigName,
    isBeacon,
    beaconInterval,
    beaconJitter,
    reconnectInterval,
    pollTimeout,
    maxConnectionErrors,
    connectionStrategy,
    debug,
    evasion,
    obfuscateSymbols,
    sgnEnabled,
    netGoEnabled,
    runAtLoad,
    trafficEncodersEnabled,
    trafficEncoders,
    canaryDomainsText,
    limitDomainJoined,
    limitHostname,
    limitUsername,
    limitDatetime,
    limitFileExists,
    limitLocale,
  })
  let cmdPreview = $derived(buildGenerateCommandPreview(generateValues))
  let presetValues = $derived(buildGenerateRequest(generateValues))
  let canGenerate = $derived(c2Urls.some(Boolean))
  let generateLabel = $derived(buildTarget === 'server' ? 'Generate' : 'Queue Builder Build')

  function closeModal() {
    open = false
    onclose?.()
  }

  function buildRequest() {
    return buildGenerateRequest(generateValues)
  }

  async function doGenerate() {
    if (!canGenerate) return
    generating = true
    error = ''
    resultPath = ''
    try {
      if (buildTarget === 'server') {
        const path = await GenerateImplantAdvanced(buildRequest())
        resultPath = path ? `Saved to ${path}` : 'Cancelled.'
      } else {
        externalBuild = await GenerateExternalBuild(JSON.stringify(buildRequest()), buildTarget, name)
        externalStatus = externalBuildSummary(externalBuild)
        resultPath = externalStatus
      }
      implantBuilds.refresh?.()
    } catch (e) {
      error = annotateGenerateError(errorMessage(e))
    } finally {
      generating = false
    }
  }

  async function refreshExternalConfig() {
    const buildID = externalBuildID(externalBuild)
    if (!buildID) return
    generating = true
    error = ''
    try {
      externalBuild = await GetExternalBuildConfig(buildID)
      externalStatus = externalBuildSummary(externalBuild, 'External build config refreshed.')
      resultPath = externalStatus
    } catch (e) {
      error = errorMessage(e, 'Refresh failed: ')
    } finally {
      generating = false
    }
  }

  async function saveExternalBinary({ name: fileName, data }) {
    const buildID = externalBuildID(externalBuild)
    if (!buildID) return
    generating = true
    error = ''
    try {
      await SaveExternalBuild(fileName, buildID, data)
      resultPath = `External build "${fileName}" saved.`
      externalStatus = resultPath
      implantBuilds.refresh?.()
    } catch (e) {
      error = errorMessage(e, 'Save failed: ')
    } finally {
      generating = false
    }
  }

  function externalBuildID(build) {
    return build?.Build?.ID || build?.build?.id || ''
  }

  function externalBuildSummary(build, prefix = 'External build queued.') {
    const buildID = externalBuildID(build)
    const target = buildTarget || 'builder'
    return `${prefix}${buildID ? ` Build ID: ${buildID}.` : ''} Target: ${target}.`
  }

  // The sliver server's Generate RPC swallows the go-build stdout/stderr and
  // hands back a bare "exit status 1" — the real compiler output only lands
  // in ~/.sliver/logs/sliver.log on the teamserver host. Nudge the operator
  // there instead of leaving them stranded on the useless error.
  function annotateGenerateError(msg) {
    if (!/exit status \d+/.test(msg)) return msg
    return `${msg}\n\nThe go compiler failed on the teamserver. The actual error is only in the server's log — grep for "--- stderr ---" in ~/.sliver/logs/sliver.log on the sliver host.`
  }

  async function doSaveProfile() {
    if (!name) {
      error = 'Profile name is required to save.'
      return
    }
    if (!canGenerate) return
    savingProfile = true
    error = ''
    resultPath = ''
    try {
      await SaveProfileAdvanced(buildRequest())
      resultPath = `Profile "${name}" saved.`
    } catch (e) {
      error = errorMessage(e)
    } finally {
      savingProfile = false
    }
  }

  function applyPreset(values) {
    if (!values) return
    if (values.name != null) name = values.name
    if (values.goos != null) goos = values.goos
    if (values.goarch != null) goarch = values.goarch
    if (values.format != null) format = values.format
    if (values.c2Urls?.length) c2Urls = [...values.c2Urls]
    if (values.httpC2ConfigName != null) httpC2ConfigName = values.httpC2ConfigName
    if (values.isBeacon != null) isBeacon = values.isBeacon
    if (values.beaconInterval != null) beaconInterval = values.beaconInterval
    if (values.beaconJitter != null) beaconJitter = values.beaconJitter
    if (values.reconnectInterval != null) reconnectInterval = values.reconnectInterval
    if (values.pollTimeout != null) pollTimeout = values.pollTimeout
    if (values.maxConnectionErrors != null) maxConnectionErrors = values.maxConnectionErrors
    if (values.connectionStrategy != null) connectionStrategy = values.connectionStrategy
    if (values.debug != null) debug = values.debug
    if (values.evasion != null) evasion = values.evasion
    if (values.obfuscateSymbols != null) obfuscateSymbols = values.obfuscateSymbols
    if (values.sgnEnabled != null) sgnEnabled = values.sgnEnabled
    if (values.netGoEnabled != null) netGoEnabled = values.netGoEnabled
    if (values.runAtLoad != null) runAtLoad = values.runAtLoad
    if (values.trafficEncodersEnabled != null) trafficEncodersEnabled = values.trafficEncodersEnabled
    if (values.trafficEncoders != null) trafficEncoders = [...values.trafficEncoders]
    if (values.canaryDomains?.length) canaryDomainsText = values.canaryDomains.join(', ')
    if (values.limitDomainJoined != null) limitDomainJoined = values.limitDomainJoined
    if (values.limitHostname != null) limitHostname = values.limitHostname
    if (values.limitUsername != null) limitUsername = values.limitUsername
    if (values.limitDatetime != null) limitDatetime = values.limitDatetime
    if (values.limitFileExists != null) limitFileExists = values.limitFileExists
    if (values.limitLocale != null) limitLocale = values.limitLocale
  }
</script>

<Modal bind:open title="Generate Implant" size="3xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    Build a Sliver implant. Beacon mode polls the C2 on an interval - quieter but higher-latency.
    Session mode holds a persistent connection - real-time but noisier. Multiple C2 URLs try in priority order.
  </p>

  <GenerateTargetSection
    bind:name
    bind:goos
    bind:goarch
    bind:format
  />
  <GenerateC2Section
    bind:c2Urls
    bind:httpC2ConfigName
    {serverHost}
  />
  <GenerateBehaviorSection
    bind:isBeacon
    bind:beaconInterval
    bind:beaconJitter
    bind:reconnectInterval
    bind:pollTimeout
    bind:maxConnectionErrors
    bind:connectionStrategy
  />
  <GenerateBuildSection
    bind:obfuscateSymbols
    bind:sgnEnabled
    bind:netGoEnabled
    bind:evasion
    bind:runAtLoad
    bind:debug
    bind:trafficEncodersEnabled
    bind:trafficEncoders
  />
  <GenerateBuilderSection
    bind:buildTarget
    {externalBuild}
    {externalStatus}
    externalBusy={generating}
    onrefreshconfig={refreshExternalConfig}
    onsavebinary={saveExternalBinary}
  />
  <GenerateConstraintsSection
    bind:limitDomainJoined
    bind:limitHostname
    bind:limitUsername
    bind:limitDatetime
    bind:limitFileExists
    bind:limitLocale
  />
  <GenerateCanariesSection bind:canaryDomainsText />
  <GenerateSpoofSection {goos} implantName={name} />
  <GenerateStatus {cmdPreview} {error} {resultPath} />

  {#snippet footer()}
    <GenerateModalActions
      {presetValues}
      onapply={applyPreset}
      oncancel={closeModal}
      onsave={doSaveProfile}
      ongenerate={doGenerate}
      {savingProfile}
      {generating}
      {canGenerate}
      {generateLabel}
    />
  {/snippet}
</Modal>
