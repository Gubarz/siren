import { createResource } from '../lib/createResource.svelte.js'
import { listJobs } from '../../api/server.js'

export const jobs = createResource({
  name: 'jobs',
  fetch: () => listJobs(),
  events: ['job-started', 'job-stopped'],
})
