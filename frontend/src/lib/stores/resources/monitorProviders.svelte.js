import { createResource } from '../lib/createResource.svelte.js';
import { listProviders } from '../../api/monitor.js';

export const monitorProviders = createResource({
  name: 'monitorProviders',
  fetch: () => listProviders(),
  events: ['monitoring-started', 'monitoring-stopped', 'monitoring-provider-added', 'monitoring-provider-removed'],
  pollInterval: 15000,
});
