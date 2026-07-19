// Client-side operator tags — persisted per teamserver on the Go side, no
// sliver-server RPC involved. Backing service is internal/tags/tags.go.

import {
  GetAgentTags,
  SetAgentTags,
  GetAllAgentTags,
  ListKnownTags,
} from '../../../wailsjs/go/main/App.js';

export {
  GetAgentTags,
  SetAgentTags,
  GetAllAgentTags,
  ListKnownTags,
};
