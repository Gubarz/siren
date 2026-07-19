import { createResource } from '../lib/createResource.svelte.js'
import { GetNetworkDiscoveries } from '../../api/discovery.js'

export const discoveries = createResource({
  name: 'discoveries',
  fetch: () => GetNetworkDiscoveries(),
  events: ['network-discovery-updated'],
})
