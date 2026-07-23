#!/bin/bash

set -e

BACKEND_HOST=$(modelctl get engine-host)
BACKEND_PORT=$(modelctl get engine-port)
SERVER_UNIX_SOCKET=$(modelctl get server-unix-socket)

DEFAULT_LANG="en"
ALLOWED_LANGUAGES="en" # use comma-separated list of languages to allow multiple languages, e.g. "auto,en,es,de"

echo "Launching adapter..."

# Get the alias of the active model
active_model_alias=$(modelctl list-models --format=json | jq -r '. as $root | .models[] | select(.name == $root."active-model") | .alias')

# Get installed models
installed_model_aliases=()
while IFS= read -r line; do
    # Split the jq output into the component name and description
    first_component_name="${line%%|*}"
    alias="${line#*|}"

    # Check if the component directory actually exists
    if [[ -d "$SNAP_COMPONENTS/$first_component_name" ]]; then
        installed_model_aliases+=("$alias")
    fi
done < <(
    # Output format per item: "first-component-name|alias"
    modelctl list-models --format=json | jq -r '.models[] | "\(.components[0])|\(.alias)"'
)
installed_model_aliases_str=$(IFS=,; echo "${installed_model_aliases[*]}")

set -x
$SNAP/bin/whisperlive-adapter serve \
    --backend-host "$BACKEND_HOST" \
    --backend-port "$BACKEND_PORT" \
    --unix-socket "$SERVER_UNIX_SOCKET" \
    --model "$active_model_alias" \
    --language "$DEFAULT_LANG" \
    --allowed-models "$installed_model_aliases_str" \
    --allowed-languages "$ALLOWED_LANGUAGES"

set +x
echo "Adapter terminated."
