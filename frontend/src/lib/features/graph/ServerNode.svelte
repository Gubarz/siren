<script>
  import Icon from '$components/ui/Icon.svelte'
  import { Handle, Position } from '@xyflow/svelte'

  let { data } = $props()
  let horizontal = $derived(data.direction === 'LR')
</script>

<div class="node server flex h-11 w-56 items-center gap-2 border-brand px-3 py-2 font-bold whitespace-nowrap">
  <Icon name="server" size={18} />
  <span class="truncate">{data.label}</span>
  <Handle type="source" position={horizontal ? Position.Right : Position.Bottom} class="handle" />
</div>
