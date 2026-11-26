# FAQ & Troubleshooting

Start with `ship doctor`.

## "no .ship/config.yaml found"

Run `ship init` in your project. Ship looks in the current directory and its parents.

## SSH connection fails

- Confirm the key at `server.key_path` is in `~/.ssh/authorized_keys` on the server.
- For a passphrase-protected key, `ssh-add <path>` avoids being asked each time.
- **Unknown server**: run `ship connect` and check the fingerprint before accepting it.
- **HOST KEY CHANGED**: someone may be intercepting the connection. If you know you rebuilt the server, run the `ssh-keygen -R` command shown in the error.

## Docker or Compose missing

Ship does not install them. Follow the [Docker Engine install guide](https://docs.docker.com/engine/install/), and add your user to the `docker` group.

## "port 80 on the server is already in use"

Something Ship did not start is listening on port 80, such as a host nginx or another app. Ship will not stop it. Free the port, or set `app.public_port` (for example 8090) and open it in your firewall. Running several Ship apps on one server is not fully supported yet.

## The build fails

The build output is shown as it runs and saved to `~/ship/<app-name>/deploy.log`. Your running containers are not changed. Common causes: the generated Dockerfile does not suit your project layout (add your own Dockerfile and keep it when asked), or the server is out of disk or memory.

A file you need may have been excluded from the upload; see [what is uploaded](configuration.md#what-is-uploaded) and `.shipignore`. `ship logs --build` shows the last build output again.

## "the app container is not staying up"

The app started and then exited or restarted; the last log lines are included in the error. Check the values in `~/ship/<app-name>/.env` on the server and that `app.port` matches the port your app listens on.

## "another deployment of this app appears to be running"

A second `ship deploy` holds the lock. It is released automatically when that command ends or its connection drops.

## Health check fails

Ship requests port 80 on the server, using your domain as the `Host` header. Any status below 500 counts as up. Check your firewall allows port 80. This check does not verify your DNS.

## Where do my SSH keys and API keys live?

- SSH keys: wherever `server.key_path` points. Used only to sign the SSH handshake.
- AI API key: `.ship/config.yaml` (mode `0600`), sent only to your provider.

Still stuck? Open a [GitHub issue](https://github.com/Koded0214h/ship/issues). Report security problems privately, as described in [SECURITY.md](../SECURITY.md).
