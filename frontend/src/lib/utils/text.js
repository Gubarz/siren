// eslint-disable-next-line no-control-regex
const ANSI_SEQUENCE = /\u001b(?:\[[0-?]*[ -/]*[@-~]|\][^\u0007]*(?:\u0007|\u001b\\)?)/g;
const HELP_MARKUP = /\[\[\.[A-Za-z]+\]\]/g;
// eslint-disable-next-line no-control-regex
const CONTROL_CHARACTERS = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f-\u009f]/g;
const TERMINAL_METADATA = new RegExp(
  // eslint-disable-next-line no-control-regex
  '(?:\\x1b\\]|\\x9d)[\\s\\S]*?(?:\\x07|\\x1b\\\\)',
  'g',
);

// Remove terminal metadata such as OSC window titles and hyperlinks while
// preserving ordinary ANSI color/style sequences for terminal rendering.
export function stripTerminalMetadata(value) {
  return String(value ?? '').replace(TERMINAL_METADATA, '');
}

export function stripTerminalFormatting(value) {
  return String(value ?? '')
    .replace(ANSI_SEQUENCE, '')
    .replace(HELP_MARKUP, '')
    .replace(/\r\n?/g, '\n')
    .replace(CONTROL_CHARACTERS, '');
}

export function commandTooltip(command) {
  const unavailable = stripTerminalFormatting(command?.unavailable).trim();
  if (unavailable) return unavailable;

  const description = stripTerminalFormatting(command?.description);
  const lines = description
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);
  const about = lines.find((line) => /^about\s*:/i.test(line));
  const summary = (about ?? lines[0] ?? '')
    .replace(/^(about|command)\s*:\s*/i, '')
    .replace(/\s+/g, ' ')
    .trim();

  if (summary.length <= 240) return summary;
  return `${summary.slice(0, 237).trimEnd()}...`;
}
