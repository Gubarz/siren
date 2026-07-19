import { createResource } from '../lib/createResource.svelte.js'
import { listTrafficEncoders } from '../../api/trafficEncoders.js'

export const trafficEncoders = createResource({
  name: 'trafficEncoders',
  fetch: () => listTrafficEncoders(),
})
