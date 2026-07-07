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

## Transcribe A Local Audio File

Use the `transcribe` command to connect directly to a Whisper Live backend and print the final transcript for a local audio file.

Basic usage:

```bash
go run . transcribe --audio data/samples/sample.wav
```

Example with custom backend, model, language, and timeout:

```bash
go run . transcribe \
	--host 127.0.0.1 \
	--port 9090 \
	--audio data/samples/sample.wav \
	--model small \
	--lang en \
	--timeout-sec 180
```

Expected output shape:

```text
connecting to whisper live at 127.0.0.1:9090
streaming audio file: data/samples/sample.wav
final transcript:
<transcribed text>
```
