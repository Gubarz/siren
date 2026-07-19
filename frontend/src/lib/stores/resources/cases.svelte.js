import { createResource } from '../lib/createResource.svelte.js'
import { ListCases } from '../../api/cases.js'

// Cases are stored client-side, so we don't need a keep-alive interval —
// the store only refetches when the case-updated event fires, which the
// Go side emits after Create/Update/Delete/Add/Remove.
export const cases = createResource({
  name: 'cases',
  fetch: () => ListCases().then((r) => r || []),
  events: ['case-updated'],
})
