import { createResource } from '../lib/createResource.svelte.js';
import { listCrackFiles } from '../../api/crack.js';

// Refetched on crack-status so any file-touching activity (upload/delete)
// causes a refresh without the 10s polling burn.
export const crackFiles = createResource({
  name: 'crackFiles',
  fetch: () => listCrackFiles(),
  events: ['crack-status'],
  pollInterval: 60000,
});
