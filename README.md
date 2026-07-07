# UbuSTT Proxy

Rely on existing transcription projects while exposing an UbuSTT-compliant API.

## Run The Server

The server can bind both to a TCP Socket or a Unix Domain Socket:

### TCP mode

Run with default host and port (`127.0.0.1:8080`):

```bash
go run . serve
```

Run with custom host and port:

```bash
go run . serve --host 0.0.0.0 --port 9000
```

Quick health check:

```bash
curl http://127.0.0.1:8080/
```

### Unix socket mode

Run bound to a Unix domain socket path:

```bash
go run . serve --unix-socket /tmp/ubustt-proxy.sock
```

When `--unix-socket` is set, it overrides `--host` and `--port`.

Quick health check over Unix socket:

```bash
curl --unix-socket /tmp/ubustt-proxy.sock http://localhost/
```
