# Contributing to Ship

Thanks for your interest in contributing. Ship is early-stage, and there's a lot of room to help — from fixing bugs to shaping how AI-generated deployments should behave.

## Ground rules

- Be respectful — see [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
- Open an issue before starting large or breaking changes, so we can align on direction before you invest time.
- Small fixes (typos, docs, small bugs) can go straight to a PR.

## Project layout

```
cmd/            Cobra CLI commands (init, connect, deploy, doctor, logs, config)
internal/ai/    AI provider clients (Anthropic, OpenAI, Gemini) + the deployment-plan prompt
internal/config/  Reads/writes the per-project .ship/config.yaml
internal/detector/  Framework & dependency detection for a project
internal/generator/ Turns an AI deployment plan into Dockerfile/Compose/Nginx/CI files
internal/ssh/       SSH connection + remote command execution
internal/sshkey/    SSH key generation/lookup
internal/ui/        Shared lipgloss/huh styling helpers
frontend/           Marketing site (React + Vite), not shipped in the CLI binary
```

## Development setup

**Requirements:** Go 1.26+ (see `go.mod`), and Node 18+ if you're working on `frontend/`.

```bash
git clone https://github.com/Koded0214h/ship.git
cd ship
go build -o ship .
./ship --help
```

To iterate quickly without rebuilding:

```bash
go run . <command>
```

`ship init` will prompt for an AI provider and API key, writing them to `.ship/config.yaml` inside whatever project directory you run it in. Any of Anthropic, OpenAI, or Gemini works — you only need a key for the provider you're testing against.

### Frontend (marketing site)

```bash
cd frontend
npm install
npm run dev
```

## Making changes

1. Fork the repo and create a branch off `main`, named `<type>/<short-description>` — e.g. `fix/nginx-ssl-fallback`, `feat/shipignore-support`, `docs/config-schema`. Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`.
2. Keep PRs focused — one logical change per PR is easier to review than a bundle of unrelated fixes.
3. Run `go build ./...` and `go vet ./...` before opening a PR. If you touched `frontend/`, run `npm run lint` and `npm run build` there too.
4. Write a clear PR description: what changed and why, not just what. Follow the PR template — it asks for a linked issue, a summary, and how you tested it.
5. Link the issue your PR addresses. PRs that don't address an open issue (or a clearly-scoped bug you found) may be asked to open one first, so we can agree on direction before reviewing the code.
6. Open the PR against `main` — direct pushes to `main` aren't allowed, and CI must pass before it can merge.

There isn't a Go test suite yet — if you're adding non-trivial logic (especially in `internal/detector` or `internal/generator`), tests are welcome and appreciated, but not a hard gate today. That said, PRs that only add coverage (see issues labeled `testing`) are very welcome.

## Maintainers and merging

Anyone can open a PR. Right now, [@Koded0214h](https://github.com/Koded0214h) is the sole maintainer and the only person who merges to `main`. `main` is protected: no direct pushes, PRs required, CI must pass before merge.

A PR is expected to be rejected or sent back for changes if it:

- doesn't address an open issue (open one first for anything non-trivial),
- lacks tests for new non-trivial logic where tests are reasonably expected,
- introduces a breaking change without prior discussion in an issue,
- is poorly documented (unclear PR description, no comments on non-obvious logic),
- bundles unrelated changes together.

**Becoming a maintainer:** there's no formal process yet, given the project's size — it'll happen organically as contributors show sustained, trustworthy involvement (multiple merged PRs, thoughtful issue triage/review). If you're interested, say so in an issue or PR discussion.

## Where to focus

Good first areas:

- Framework/dependency detection (`internal/detector`) — more languages and frameworks recognized correctly.
- Generated Dockerfile/Compose/Nginx quality (`internal/generator`, and the system prompt in `internal/ai/provider.go`).
- CLI UX and error messages (`internal/ui`, `cmd/`).
- Docs and examples.

Check open issues, especially any labeled `good first issue` or `help wanted`.

## Reporting bugs

Open a GitHub issue with:

- What you ran (command + relevant flags)
- What you expected vs. what happened
- Your OS/arch and `ship --version`
- Relevant output (redact secrets, hostnames, and API keys)

For security vulnerabilities, see [SECURITY.md](SECURITY.md) — please don't open a public issue.

## Commit messages

No strict format is enforced, but a short, descriptive summary line (`fix: handle missing Dockerfile port`, `feat: detect Bun projects`) makes history easier to scan.
