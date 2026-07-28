#!/bin/bash

set -e

BACKEND_HOST=$(modelctl get whisper-live.ws.host)
BACKEND_PORT=$(modelctl get whisper-live.ws.port)
ADAPTER_HOST=$(modelctl get http.host)
ADAPTER_PORT=$(modelctl get http.port)

DEFAULT_LANG=$(modelctl get transcription-language)

echo "Launching adapter..."

# Get the alias of the active model
active_model_alias=$(modelctl list-models --format=json | jq -r '. as $root | .models[] | select(.name == $root."active-model") | .alias')

set -x
$SNAP/bin/whisperlive-adapter serve \
    --backend-host "$BACKEND_HOST" \
    --backend-port "$BACKEND_PORT" \
    --host "$ADAPTER_HOST" \
    --port "$ADAPTER_PORT" \
    --model "$active_model_alias" \
    --language "$DEFAULT_LANG" \
    --allowed-models "$active_model_alias" \
    --allowed-languages "${MODEL_SUPPORTED_LANGUAGES//\"/}"

set +x
echo "Adapter terminated."
