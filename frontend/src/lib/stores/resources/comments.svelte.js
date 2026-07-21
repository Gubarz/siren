import { createResource } from '../lib/createResource.svelte.js'
import { GetAllComments } from '../../api/comments.js'

export const comments = createResource({
  name: 'comments',
  fetch: () => GetAllComments().then((res) => res || {}),
  events: ['comments-updated'],
})
