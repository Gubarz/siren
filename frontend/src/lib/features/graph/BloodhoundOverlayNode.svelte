<script>
  import Badge from '$components/ui/Badge.svelte';
  import Icon from '$components/ui/Icon.svelte';
  import { Handle, Position } from '@xyflow/svelte';

  let { data, selected = false } = $props();

  let horizontal = $derived(data.direction === 'LR');

  let nodeStyle = $derived(
    (selected ? 'border-color: var(--color-brand);' : '') +
    (data.tierZero
      ? ' border-left-color: var(--color-danger-500); background-image: linear-gradient(color-mix(in srgb, var(--color-danger-500) 12%, transparent), color-mix(in srgb, var(--color-danger-500) 12%, transparent));'
      : data.owned
        ? ' border-left-color: var(--color-success-500);'
        : '')
  );
</script>

<div
  class="node h-16 w-50 py-1 px-2 border-l-4 cursor-default {selected ? 'node--selected-agent' : ''}"
  style={nodeStyle}
>
  <Handle type="target" position={horizontal ? Position.Left : Position.Top} class="handle" />
  <div class="flex items-center gap-1">
    <Icon name="workflow" size={12} />
    <span class="font-bold flex-1 truncate text-xs" title={data.label}>{data.label}</span>
  </div>
  <div class="mt-1 flex items-center gap-1">
    {#if data.kind}
      <Badge variant={data.tierZero ? 'danger' : data.owned ? 'success' : 'default'} size="xs">{data.kind}</Badge>
    {/if}
    {#if data.tierZero}
      <Badge variant="danger" size="xs">T0</Badge>
    {/if}
    {#if data.distance >= 0}
      <span class="text-3xs text-fg-muted">{data.distance}h</span>
    {/if}
  </div>
  <Handle type="source" position={horizontal ? Position.Right : Position.Bottom} class="handle" />
</div>
