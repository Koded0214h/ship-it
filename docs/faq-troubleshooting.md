# FAQ & Troubleshooting

Start with `ship doctor` — it checks your config file, SSH connection, Docker on the server, your AI provider's API key, and (if you set a domain) DNS/HTTP reachability, in that order.

## "no .ship/config.yaml found"

You haven't run `ship init` in this project (or a parent directory of it). Config is looked up by walking up from the current directory, so make sure you're inside the project you initialized.

## SSH connection fails

- Confirm the key at `server.key_path` in `.ship/config.yaml` (or shown by `ship config`) exists and is the one added to `~/.ssh/authorized_keys` on your server.
- `ship init` prints the exact `authorized_keys` line when it generates a new key — if you skipped adding it, re-run `ship connect` after adding it.
- Ship uses `ssh.InsecureIgnoreHostKey()`, so host key mismatches aren't the problem — check the host/user/port instead.

## "Docker not found" on the server

`ship deploy` doesn't install Docker for you automatically today — `ship connect`/`ship doctor` will tell you it's missing. Install it yourself first: `apt-get install -y docker.io docker-compose-plugin` (Ubuntu/Debian), then re-run `ship deploy`.

## AI provider key invalid

`ship doctor` calls `Ping()` on your configured provider. If it fails, double check `ai.api_key` via `ship config` — a masked preview is shown. For Gemini specifically, keys should start with `AIza`; `ship config` warns if yours doesn't.

## Containers never come up / deploy times out

`ship deploy` polls for up to 3 minutes after `docker compose up -d --build`, then prints the tail of the build log. Most often this is a bad generated Dockerfile/Compose file for an unusual project layout, or the server running low on disk/memory (check with `ship connect`). SSH into the server and run `docker compose logs` in the app directory for the full picture.

## Health check fails but the app looks fine on the server

The health check does a plain HTTP GET to your domain (or the server's IP if you didn't set one) and only treats responses `>= 500` as failures — redirects and 4xxs count as "up". If it's still failing, check that Nginx is actually listening on port 80 and proxying to `app:<port>` inside the container network, and that no firewall is blocking inbound port 80/443.

## Where do my SSH keys and API keys live?

- SSH keys: wherever you pointed `server.key_path`, or `~/.ssh/ship_ed25519` if Ship generated one for you. Never sent anywhere — used only to sign the local SSH handshake.
- AI API key: stored in `.ship/config.yaml` (mode `0600`, gitignored) and sent only to your configured AI provider.

See [Architecture](architecture.md#where-credentials-go) for more.

Didn't find your issue here? Open a [GitHub issue](https://github.com/Koded0214h/ship/issues) — see [CONTRIBUTING.md](../CONTRIBUTING.md#reporting-bugs) for what to include.
