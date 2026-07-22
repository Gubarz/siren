// Client-side operator tags — persisted per teamserver on the Go side, no
// sliver-server RPC involved. Backing service is internal/tags/tags.go.

import {
  GetAgentTags,
  GetEntityTags,
  SetAgentTags,
  SetEntityTags,
  GetAllAgentTags,
  GetAllEntityTags,
  ListKnownTags,
  GetAllAgentColors,
  GetAllEntityColors,
  GetEntityColor,
  SetAgentColor,
  SetEntityColor,
} from '../../../wailsjs/go/gui/App.js';

export {
  GetAgentTags,
  GetEntityTags,
  SetAgentTags,
  SetEntityTags,
  GetAllAgentTags,
  GetAllEntityTags,
  ListKnownTags,
  GetAllAgentColors,
  GetAllEntityColors,
  GetEntityColor,
  SetAgentColor,
  SetEntityColor,
};
