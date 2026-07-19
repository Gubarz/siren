import { createResource } from '../lib/createResource.svelte.js';
import { listCrackstations } from '../../api/crack.js';

// Event-driven refresh — sliver emits crackstation-{connected,disconnected}
// and crack-benchmark / crack-status through the sliver-event pipe, so
// the store subscribes there and only keeps a 60s keepalive as backstop.
export const crackstations = createResource({
  name: 'crackstations',
  fetch: () => listCrackstations(),
  events: [
    'crackstation-connected',
    'crackstation-disconnected',
    'crack-benchmark',
    'crack-status',
  ],
  pollInterval: 60000,
});
