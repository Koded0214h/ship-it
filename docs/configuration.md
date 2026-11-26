# Configuration

Ship stores per-project config at `.ship/config.yaml`, created by `ship init` and editable via `ship config`. Commands walk up from the current directory to find it, and the directory containing `.ship/` is treated as the project root: that is what `ship deploy` inspects and uploads, even if you run it from a subdirectory.

`.ship/` is added to your `.gitignore` by `ship init`. It contains your AI API key, so it should never be committed. If you clone a repository that already contains a `.ship/config.yaml`, read it before running Ship: it decides which server your code is sent to.

## Schema

```yaml
app:
  name: my-api      # required. Letters, digits, '.', '_', '-'. Used as the remote directory name
  domain: ""        # optional hostname (no scheme or path). Blank deploys on the server's IP
  port: 8000        # the port your app listens on inside its container. 0 = detect

server:
  host: ""          # required. IP or hostname of your VPS
  user: ""          # required. SSH user, who must be able to run docker
  key_path: ""      # path to your private SSH key (passphrase-protected keys are not supported yet)
  port: 22          # SSH port, 22 if unset

ai:                 # optional. Leave empty to use the built-in templates
  provider: ""      # "anthropic" (or "claude"), "openai", or "gemini" (or "google")
  api_key: ""
  model: ""         # blank uses the provider's default
```

## Validation

Every command that talks to your server checks that `app.name`, `server.host` and `server.user` are set, and that names, hostnames and ports contain only valid values. The AI section is never required.

`ship init` fills `app.port` from detection. If it later differs from the detected default, `ship deploy` asks which one to use.

## What is uploaded

`ship deploy` uploads the project root, except:

- **Secrets:** `.env`, `.env.*`, `*.env`, `*.pem`, `*.key`, `*.p12`, `*.pfx`, private SSH key names, `.npmrc`, `.pypirc`, `.netrc`. Files ending `.example`, `.sample`, `.template` or `.dist` are uploaded.
- **Never source:** `.git`, `.ship`, `node_modules`, `__pycache__`, virtualenvs, `.venv`, `venv`, `.idea`, `.next`.
- **Top-level build output:** `dist`, `build`; `target` for Rust and Maven projects; `vendor` for PHP and Ruby.
- Files over 30 MB, and symlinks that are broken, absolute or point outside the project.

Skipped secrets, build output, large files and symlinks are listed during the deploy. There is no override yet.

## What lives on the server

Everything is in `~/ship/<app-name>/`: your source, the Dockerfile, `docker-compose.yml`, `nginx.conf`, `.dockerignore`, `deploy.log` and `.env`.

`.env` is created only if it does not exist, with a random database password, and is never overwritten. Edit it on the server to set your own values, then redeploy. Files from earlier deploys are not deleted, including secrets uploaded by `v0.1.0`; remove those by hand.

## Editing config

`ship config` shows your current settings (the API key is masked) and lets you update one section at a time. Changing AI provider requires that provider's key.
