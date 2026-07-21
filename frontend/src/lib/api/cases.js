// Case-file API — client-side rollup of an engagement's agents/loot/creds/
// hosts. Backing service is internal/casefile; nothing here talks to the
// sliver teamserver directly.

import {
  ListCases,
  GetCase,
  CreateCase,
  UpdateCase,
  DeleteCase,
  AddToCase,
  RemoveFromCase,
  GenerateCaseReport,
} from '../../../wailsjs/go/main/App.js';

export {
  ListCases,
  GetCase,
  CreateCase,
  UpdateCase,
  DeleteCase,
  AddToCase,
  RemoveFromCase,
  GenerateCaseReport,
};

export function ExportCaseReport(caseID) {
  return window.go.main.App.ExportCaseReport(caseID)
}
