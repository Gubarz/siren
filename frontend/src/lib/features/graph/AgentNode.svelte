<script>
  import Icon from '$components/ui/Icon.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import { Handle, Position } from '@xyflow/svelte'

  let { data, selected = false } = $props()
  let horizontal = $derived(data.direction === 'LR')
</script>

<div
  class={`node agent ${data.kind} h-28 w-64 border-l-4 px-2 py-2 ${selected ? 'node--selected-agent' : ''} ${data.dead ? 'opacity-60' : ''}`}
  style={selected
    ? `border-color: var(--color-brand);`
    : `border-left-color: var(--color-${data.kind === 'beacon' ? 'beacon' : 'success'}-500);`}
>
  <Handle type="target" position={horizontal ? Position.Left : Position.Top} class="handle" />
  <div class="flex items-center gap-2">
    <Icon name={data.icon} size={16} />
    <span class="min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap font-bold" title={data.agentID}>{data.agentID}</span>
    <span class={`h-2 w-2 shrink-0 rounded-full ${data.dead ? 'bg-danger-500' : 'bg-success-500 node-glow-success'}`}></span>
  </div>
  <div class="mt-1 overflow-hidden text-ellipsis whitespace-nowrap text-2xs" title={data.implantName}>{data.implantName}</div>
  <div class={`my-1 overflow-hidden text-ellipsis whitespace-nowrap text-2xs ${data.priv === 'high' ? 'font-semibold text-danger-500' : 'text-fg-muted'}`}>{data.user}@{data.host}</div>
  <div class="flex items-center justify-between gap-2">
    <Badge variant={data.kind} size="graph">{data.kind}</Badge>
    <span class="overflow-hidden text-ellipsis whitespace-nowrap text-3xs text-fg-muted">{data.addr || ''}</span>
  </div>
  <Handle type="source" position={horizontal ? Position.Right : Position.Bottom} class="handle" />
</div>
