import { createResource } from '../lib/createResource.svelte.js'
import { ListAutomationRules } from '../../api/automation.js'

export const automation = createResource({
  name: 'automation',
  fetch: () => ListAutomationRules().then((rules) => rules || []),
  events: ['automation-updated'],
})
