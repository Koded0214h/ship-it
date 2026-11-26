# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Security
- SSH host keys are verified against `known_hosts`; new servers must be confirmed and changed keys are refused.
- `.env` files and private keys are no longer uploaded or baked into images. `v0.1.0` uploaded them; remove any left in `~/ship/<app>` and rotate what was exposed.
- App name, domain, host, user and ports are validated before being used in remote commands.
- Generated database passwords are random instead of `changeme`.
- The local project path is no longer sent to AI providers.

### Fixed
- Deploys no longer stop and remove every container on the server, or prune the build cache.
- A failed build or crashing app is reported as a failure, with a non-zero exit code.
- Images are built before running containers are replaced.
- `app.port` is now used; deploys run from the project root.
- An existing server `.env` and SSH key are never overwritten.
- Generated Nginx no longer answers the app's `/health` routes.
- Detection of capitalised Python requirements, Node `pg`, Rails `bullet` and non-Postgres database URLs.

### Changed
- Deploy shows the exact files before asking for confirmation, and rejects plans that reach into the host.
- Docker with Compose v2 must already be installed.
- Existing app names with spaces or other special characters are now rejected.

### Removed
- GitHub Actions workflow generation, which did not work.
- HTTPS options and claims, until certificate issuance exists.

### Added
- `ship deploy --no-ai`, and an offer to use the template when the AI provider fails.
- Deterministic fallback for language/framework detection when no AI provider is configured.
- Open-source project scaffolding: `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, issue/PR templates.

## [0.1.0] - 2026-06-29

Initial public release.

### Added
- `ship init` — detect the project's language/framework and scaffold a deployment config.
- `ship deploy` — AI-generated deployment plan (Dockerfile, docker-compose, nginx config) executed over SSH.
- `ship doctor` — one-shot health check against the deployed app.
- Support for multiple AI providers.
- Domain + optional SSL configuration in `.ship/config.yaml`.

[Unreleased]: https://github.com/Koded0214h/ship-it/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Koded0214h/ship-it/releases/tag/v0.1.0
