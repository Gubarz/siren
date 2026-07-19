import { createResource } from '../lib/createResource.svelte.js'
import { listSessions } from '../../api/agents.js'

export const sessions = createResource({
  name: 'sessions',
  fetch: () => listSessions(),
  events: ['session-connected', 'session-disconnected', 'session-updated'],
})
