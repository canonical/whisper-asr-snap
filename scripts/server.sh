#!/bin/bash

set -e

BACKEND_HOST=$(modelctl get engine-host)
BACKEND_PORT=$(modelctl get engine-port)
SERVER_UNIX_SOCKET=$(modelctl get server-unix-socket)

# TODO: get these from config, and allow multiple models and languages
DEFAULT_MODEL="base"
DEFAULT_LANG="en"
ALLOWED_MODELS="base" # use comma-separated list of models to allow multiple models, e.g. "base,small,medium"
ALLOWED_LANGUAGES="en" # use comma-separated list of languages to allow multiple languages, e.g. "auto,en,es,de"
echo "Launching adapter..."

set -x
$SNAP/bin/whisperlive-adapter serve \
    --backend-host "$BACKEND_HOST" \
    --backend-port "$BACKEND_PORT" \
    --unix-socket "$SERVER_UNIX_SOCKET" \
    --model "$DEFAULT_MODEL" \
    --language "$DEFAULT_LANG" \
    --allowed-models "$ALLOWED_MODELS" \
    --allowed-languages "$ALLOWED_LANGUAGES"

set +x
echo "Adapter terminated."
