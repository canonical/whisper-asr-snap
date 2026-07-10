# UbuSTT Proxy

Rely on existing transcription projects while exposing an UbuSTT-compliant API.

## Building

To build the UbuSTT proxy:

```bash
go build ./cmd/proxy
```

## Run The Server

The server can bind both to a TCP Socket or a Unix Domain Socket:

### TCP mode

Run with default host and port (`127.0.0.1:8080`):

```bash
go run ./cmd/proxy serve
```

Run with custom host and port:

```bash
go run ./cmd/proxy serve --host 0.0.0.0 --port 8080
```

Quick health check:

```bash
curl http://127.0.0.1:8080/
```

### Unix socket mode

Run bound to a Unix domain socket path:

```bash
go run ./cmd/proxy serve --unix-socket /tmp/ubustt-proxy.sock
```

When `--unix-socket` is set, `--host` and `--port` are ignored.

Quick health check over Unix socket:

```bash
curl --unix-socket /tmp/ubustt-proxy.sock http://localhost/
```

### Backend configuration

The server will reach a WhisperLive backend at its default address of `127.0.0.1:9090`. 
This value can be changed by setting `--whisper-host` and `--backend-port`.

The default model and language can be changed by setting `--model` and `--language`.
To restrict which models and languages are allowed, use `--allowed-models` and `--allowed-languages`.

```bash
go run ./cmd/proxy serve \
	--unix-socket /tmp/ubustt-proxy.sock \
	--whisper-host 127.0.0.1 \
	--backend-port 9090 \
	--model small \
	--language en \
	--allowed-models "small,tiny" \
	--allowed-languages "auto,en,fr"
```

## Debugging

This project includes a debug entry point to run inference directly against the backend or through a running instance of UbuSTT proxy.

### Prompting the backend

The `use-backend` command connects directly to a backend and prints the final transcript for a local audio file. This procedure requires `ffmpeg`.

Basic usage:

```bash
go run ./cmd/debug use-backend --audio data/samples/jfk.flac
```

Example with custom model, language, and timeout:

```bash
go run ./cmd/debug use-backend \
	--host 127.0.0.1 \
	--port 9090 \
	--audio data/samples/jfk.flac \
	--model small \
	--lang en \
	--timeout-sec 180
```

Expected output shape:

```text
connecting to whisper live at 127.0.0.1:9090
streaming audio file: data/samples/jfk.flac
final transcript:
<transcribed text>
```

### Prompting UbuSTT

The `use-proxy` command to stream a local audio file to a running UbuSTT proxy server and print the transcription events as they arrive. The procedure requires `ffmpeg`.

You can optionally set `--realtime-factor <float>` value to speed up audio processing.

Over TCP:

```bash
go run ./cmd/debug use-proxy --url ws://127.0.0.1:8080/ws --audio data/samples/jfk.flac
```

Over a Unix domain socket:

```bash
go run ./cmd/debug use-proxy --unix-socket /tmp/ubustt-proxy.sock --audio data/samples/jfk.flac
```

If the `--unix-socket` argument is used, `--url` is ignored.
