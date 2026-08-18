<script>
  import { onMount } from 'svelte'
  import CollapsibleGroup from '$components/forms/CollapsibleGroup.svelte'
  import SelectField from '$components/forms/SelectField.svelte'
  import Button from '$components/ui/Button.svelte'
  import { jobs } from '$stores/resources/jobs.svelte.js'
  import { httpC2Profiles } from '$stores/resources/httpC2Profiles.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(jobs, httpC2Profiles)
  import { profileName } from '../../../api/httpc2.js'
  import { listenerHost, listenerProtocol } from '../../../utils/listeners.js'
  import GenerateC2Row from './GenerateC2Row.svelte'

  let {
    c2Urls = $bindable(['']),
    httpC2ConfigName = $bindable('default'),
    serverHost = '',
  } = $props()

  const C2_PROTOCOLS = ['mtls', 'http', 'https', 'dns', 'wg']

  onMount(() => {
    jobs.refresh?.()
    httpC2Profiles.refresh?.()
  })

  let c2Listeners = $derived.by(() => {
    const list = jobs?.data || []
    return list
      .map((job) => {
        const protocol = listenerProtocol(job)
        return {
          id: job.ID ?? job.id,
          name: job.Name ?? job.name,
          protocol,
          port: job.Port ?? job.port,
          host: listenerHost(job, serverHost),
          domains: job.Domains ?? job.domains ?? [],
          description: job.Description ?? job.description ?? '',
        }
      })
      .filter((listener) => C2_PROTOCOLS.includes(listener.protocol))
  })
  let hasHttpChannel = $derived(c2Urls.some((url) => /^https?:\/\//i.test(url || '')))
  let httpC2Options = $derived.by(() => {
    const names = new Set(['default'])
    for (const profile of httpC2Profiles.data || []) {
      const name = profileName(profile)
      if (name) names.add(name)
    }
    if (httpC2ConfigName) names.add(httpC2ConfigName)
    return [...names].sort().map((name) => ({ value: name, label: name }))
  })

  function setC2Urls(next) {
    c2Urls = next.length ? next : ['']
  }

  function updateC2(index, value) {
    const next = [...c2Urls]
    next[index] = value
    c2Urls = next
  }

  function addC2() {
    setC2Urls([...c2Urls, ''])
  }

  function removeC2(index) {
    if (c2Urls.length === 1) return
    setC2Urls(c2Urls.filter((_, currentIndex) => currentIndex !== index))
  }

  function moveC2Up(index) {
    if (index === 0) return
    const next = [...c2Urls]
    const previous = next[index - 1]
    next[index - 1] = next[index]
    next[index] = previous
    setC2Urls(next)
  }

  function moveC2Down(index) {
    if (index === c2Urls.length - 1) return
    const next = [...c2Urls]
    const following = next[index + 1]
    next[index + 1] = next[index]
    next[index] = following
    setC2Urls(next)
  }

  function setProto(index, prefix) {
    const current = c2Urls[index] || ''
    const stripped = current.replace(/^(mtls|https?|dns|wg|tcp-pivot|namedpipe):\/\//i, '')
    updateC2(index, prefix + stripped)
  }

  function pickListener(index, listener) {
    if (listener.protocol === 'dns') {
      updateC2(index, `dns://${listener.host || ''}`)
    } else {
      updateC2(index, `${listener.protocol}://${listener.host || serverHost || '<server>'}:${listener.port}`)
    }
  }
</script>

<CollapsibleGroup title="C2 channels (priority order)" open={true}>
  <p class="text-fg-muted text-xs mb-2">
    Implant tries each URL in order until one connects. Include the protocol prefix - <code>mtls://</code>, <code>https://</code>, <code>dns://</code>, <code>wg://</code>.
  </p>
  {#each c2Urls as c2Url, i}
    <GenerateC2Row
      {c2Url}
      index={i}
      total={c2Urls.length}
      {c2Listeners}
      onupdate={updateC2}
      onproto={setProto}
      onlistener={pickListener}
      onmoveup={moveC2Up}
      onmovedown={moveC2Down}
      onremove={removeC2}
    />
  {/each}
  <Button color="alternative" size="xs" icon="plus" onclick={addC2}>Add C2 URL</Button>
  {#if hasHttpChannel}
    <SelectField
      bind:value={httpC2ConfigName}
      label="HTTP C2 profile"
      options={httpC2Options}
      description="Which malleable HTTP C2 profile to embed for the http(s):// channels above"
    />
  {/if}
</CollapsibleGroup>
