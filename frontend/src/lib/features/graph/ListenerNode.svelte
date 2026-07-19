<script>
  import Icon from '$components/ui/Icon.svelte'
  import { Handle, Position } from '@xyflow/svelte'

  let { data } = $props()
  let horizontal = $derived(data.direction === 'LR')
</script>

<div class="node w-72 h-10 flex items-center gap-2 py-2 px-3 text-xs border-dashed">
  <Handle type="target" position={horizontal ? Position.Left : Position.Top} class="handle" />
  <Icon name="satellite-dish" size={16} />
  <span class="truncate" title={data.label}>{data.label}</span>
  <Handle type="source" position={horizontal ? Position.Right : Position.Bottom} class="handle" />
</div>
