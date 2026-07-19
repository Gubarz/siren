import { createResource } from '../lib/createResource.svelte.js'
import { listImplantBuilds } from '../../api/server.js'

export const implantBuilds = createResource({
  name: 'implantBuilds',
  fetch: () => listImplantBuilds(),
})
