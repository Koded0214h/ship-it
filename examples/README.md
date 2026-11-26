# Examples

One minimal app per stack. Each is detected, containerised with Ship's generated
Dockerfile, started and requested over HTTP:

```bash
go test -tags docker -v -timeout 40m ./internal/generator -run Examples
```

Every app answers `GET /` with `ok` and listens on `$PORT`.
