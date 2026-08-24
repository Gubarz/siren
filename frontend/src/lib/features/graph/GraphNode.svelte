<script>
  import AgentNode from './AgentNode.svelte'
  import ListenerNode from './ListenerNode.svelte'
  import DeviceNode from './DeviceNode.svelte'
  import ServerNode from './ServerNode.svelte'
  import BloodhoundOverlayNode from './BloodhoundOverlayNode.svelte'

  let { data, selected = false } = $props()
</script>

{#if data.variant === 'agent'}
  <AgentNode {data} {selected} />
{:else if data.variant === 'listener'}
  <ListenerNode {data} />
{:else if data.variant === 'device'}
  <DeviceNode {data} {selected} />
{:else if data.variant === 'bloodhound'}
  <BloodhoundOverlayNode {data} {selected} />
{:else}
  <ServerNode {data} />
{/if}
