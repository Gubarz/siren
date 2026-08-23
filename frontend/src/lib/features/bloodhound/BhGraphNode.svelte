<script>
  import Badge from '$components/ui/Badge.svelte';
  import { Handle, Position } from '@xyflow/svelte';

  let { data, selected = false } = $props();

  let nodeStyle = $derived(
    selected
      ? 'border-color: var(--color-brand);'
      : data.tierZero
        ? 'border-left-color: var(--color-danger-500); background-image: linear-gradient(color-mix(in srgb, var(--color-danger-500) 14%, transparent), color-mix(in srgb, var(--color-danger-500) 14%, transparent));'
        : data.owned
          ? 'border-left-color: var(--color-success-500);'
          : ''
  );
</script>

<div class="node h-10 w-42 border-l-4 px-2" style={nodeStyle}>
  <Handle type="target" position={Position.Left} class="handle" />
  <div class="flex h-full items-center gap-2">
    <span class="min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap font-bold" title={data.label}>{data.label}</span>
    {#if data.kind}
      <Badge variant={data.tierZero ? 'danger' : data.owned ? 'success' : 'default'} size="graph">{data.kind}</Badge>
    {/if}
  </div>
  <Handle type="source" position={Position.Right} class="handle" />
</div>
