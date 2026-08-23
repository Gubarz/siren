<script>
  import Badge from '$components/ui/Badge.svelte';
  import { enrichmentChip } from './enrichment.js';

  let { enrichment = null } = $props();

  let chip = $derived(enrichmentChip(enrichment));
</script>

{#if chip}
  <span title={chip.title}>
    <Badge variant={chip.kind === 'tierZero' ? 'danger' : chip.kind === 'owned' ? 'success' : 'default'}>
      {chip.label}
    </Badge>
  </span>
{/if}
