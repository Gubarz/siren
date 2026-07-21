# Security Policy

## Supported Versions

Only the latest release and the `main` branch receive security fixes.

| Version | Supported |
| --- | --- |
| Latest release | Yes |
| Older releases | No |

## Reporting a Vulnerability

**Please do not open public issues for security vulnerabilities.**

Report them privately through [GitHub Security Advisories](https://github.com/Gubarz/sliver-gui/security/advisories/new).

Include:

- The version you are running (`sliver-gui --version`, or the commit hash)
- Your OS and architecture
- The Sliver teamserver version you were connected to, if relevant
- Steps to reproduce, and the security impact you see

This project is maintained by a single developer in spare time. Expect an
acknowledgment within a few days, but there is no formal response-time SLA.

## Scope

**In scope:** the sliver-gui application itself — the Go backend, the Svelte
frontend, the console subprocess handling, and the build/CI configuration in
this repository.

**Out of scope:**

- The Sliver framework (teamserver, implants, RPC). Report those to
  [Bishop Fox](https://github.com/BishopFox/sliver/security).
- Vulnerabilities in third-party dependencies. Report them upstream, and feel
  free to open a regular issue here so the dependency gets bumped.

## Authorized Use

sliver-gui is an offensive-security tool. Use it solely on systems you own or
have explicit written permission to test.

## Data Handling

- All application data — operator configs, event history, cases, notes,
  automation rules — is stored locally. There is **no telemetry**, no crash
  reporting, and no phone-home of any kind.
- Operator config files contain the private keys that authenticate you to a
  teamserver. The app only reads them locally; protecting them is your
  responsibility.
- Automation rules execute JavaScript with the full privileges of the
  application. Only import or enable scripts you trust.
- Interactive consoles run as subprocesses with the application's privileges.
