<script>
  import Icon from '$components/ui/Icon.svelte'
  import { Handle, Position } from '@xyflow/svelte'

  let { data } = $props()
  let horizontal = $derived(data.direction === 'LR')
</script>

<div
  class="node flex h-8 items-center justify-center gap-2 border-dashed px-2 py-1 text-xs whitespace-nowrap"
  style="width: {data.width}px;"
>
  <Handle type="target" position={horizontal ? Position.Left : Position.Top} class="handle" />
  <Icon name="satellite-dish" size={16} />
  <span title={data.label}>{data.label}</span>
  <Handle type="source" position={horizontal ? Position.Right : Position.Bottom} class="handle" />
</div>
