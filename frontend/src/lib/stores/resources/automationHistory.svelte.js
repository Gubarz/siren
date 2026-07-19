import { createResource } from '../lib/createResource.svelte.js'
import { GetAutomationHistory } from '../../api/automation.js'

export const automationHistory = createResource({
  name: 'automationHistory',
  fetch: () => GetAutomationHistory().then((history) => history || []),
  events: ['automation-run'],
})
