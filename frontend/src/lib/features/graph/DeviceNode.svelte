<script>
  import Icon from '$components/ui/Icon.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import { Handle, Position } from '@xyflow/svelte'

  let { data, selected = false } = $props()
  let horizontal = $derived(data.direction === 'LR')
</script>

<div
  class="node w-64 h-28 py-2 px-3 border-l-4 border-l-device-500 cursor-default {selected ? 'node--selected-device' : ''}"
>
  <Handle type="target" position={horizontal ? Position.Left : Position.Top} class="handle" />
  <div class="flex items-center gap-2">
    <Icon name="laptop" size={16} />
    <span class="font-bold flex-1 truncate" title={data.ip}>{data.ip}</span>
    <span class="w-2 h-2 rounded-full shrink-0 bg-device-500 node-glow-device"></span>
  </div>
  <div class="text-2xs mt-1 truncate" title={data.hostname || 'Unknown hostname'}>
    {data.hostname || 'Unknown hostname'}
  </div>
  <div class="text-2xs my-1 truncate text-fg-muted" title={`Vendor: ${data.vendor || 'unknown'} / OS: ${data.osHint || 'unknown'}`}>
    {data.vendor ? `Vendor: ${data.vendor}` : 'Vendor unknown'} · {data.osHint || 'OS unknown'}
  </div>
  <div class="flex items-center justify-between gap-2">
    <Badge variant="discovered" size="graph">{(data.method || 'discovered').toUpperCase()}</Badge>
    <span class="text-3xs truncate text-fg-muted" title={data.mac || 'MAC unavailable'}>{data.mac || 'MAC unavailable'}</span>
  </div>
</div>
