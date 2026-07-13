#!/bin/bash

set -e

BACKEND_HOST="127.0.0.1" # TODO: replace with snap config value when not hardcoded to 0.0.0.0 anymore
BACKEND_PORT=$(snapctl get engine-port)
HOST=$(snapctl get server-host)
PORT=$(snapctl get server-port)

# TODO: get these from config, and allow multiple models and languages
DEFAULT_MODEL="base"
DEFAULT_LANG="en"
ALLOWED_MODELS="base" # use comma-separated list of models to allow multiple models, e.g. "base,small,medium"
ALLOWED_LANGUAGES="en" # use comma-separated list of languages to allow multiple languages, e.g. "auto,en,es,de"

echo "Launching proxy..."
echo "Using backend-host=$BACKEND_HOST, backend-port=$BACKEND_PORT, host=$HOST, port=$PORT, default-model=$DEFAULT_MODEL, default-lang=$DEFAULT_LANG, allowed-models=$ALLOWED_MODELS, allowed-languages=$ALLOWED_LANGUAGES"

$SNAP/bin/proxy serve \
    --backend-host "$BACKEND_HOST" \
    --backend-port "$BACKEND_PORT" \
    --host "$HOST" \
    --port "$PORT" \
    --model "$DEFAULT_MODEL" \
    --language "$DEFAULT_LANG" \
    --allowed-models "$ALLOWED_MODELS" \
    --allowed-languages "$ALLOWED_LANGUAGES"

echo "Proxy terminated."
