import { createResource } from '../lib/createResource.svelte.js'
import { listBeacons } from '../../api/agents.js'

export const beacons = createResource({
  name: 'beacons',
  fetch: () => listBeacons(),
  events: ['beacon-registered', 'beacon-checkin'],
  pollInterval: 5000,
})
