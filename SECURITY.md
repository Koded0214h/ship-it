# Security Policy

Ship runs on your machine and connects to your own servers over SSH. Your SSH
keys and server credentials are never sent anywhere — only project metadata
needed to generate a deployment plan may be sent to your configured AI
provider (Anthropic, OpenAI, or Gemini).

## Reporting a Vulnerability

If you find a security issue — in the CLI, the generated deployment configs
(Docker/Nginx/CI), or how credentials are handled — please **do not** open a
public issue.

Instead, report it privately via [GitHub Security Advisories](../../security/advisories/new)
for this repository. Include:

- A description of the issue and its impact
- Steps to reproduce, or a proof of concept
- Affected version (`ship --version`)

We'll acknowledge reports as soon as we can and work with you on a fix and
disclosure timeline.

## Supported Versions

Ship is pre-1.0 and moving quickly. Only the latest released version is
supported with security fixes.
