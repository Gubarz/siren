import { createResource } from '../lib/createResource.svelte.js'
import { listShellcodeEncoders } from '../../api/shellcode.js'

export const shellcodeEncoders = createResource({
  name: 'shellcodeEncoders',
  fetch: () => listShellcodeEncoders(),
})
