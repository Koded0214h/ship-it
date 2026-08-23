# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
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
