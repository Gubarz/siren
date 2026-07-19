import { createResource } from '../lib/createResource.svelte.js';
import { listBuilders } from '../../api/builders.js';

export const builders = createResource({
  name: 'builders',
  fetch: () => listBuilders(),
  pollInterval: 15000,
});
