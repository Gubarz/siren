import { createResource } from '../lib/createResource.svelte.js'
import { GetAgentNotes } from '../../api/agents.js'

export const sessionNotes = createResource({
  name: 'sessionNotes',
  fetch: () => GetAgentNotes().then(r => r || {}),
  events: ['agent-notes-updated', 'comments-updated'],
})
