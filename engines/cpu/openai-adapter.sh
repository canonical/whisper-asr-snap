#!/bin/bash

set -e

BACKEND_HOST=$(modelctl get whisper-live.ws.host)
BACKEND_PORT=$(modelctl get whisper-live.ws.port)
ADAPTER_HOST=$(modelctl get http.host)
ADAPTER_PORT=$(modelctl get http.port)

DEFAULT_LANG="en"
ALLOWED_LANGUAGES="en" # use comma-separated list of languages to allow multiple languages, e.g. "auto,en,es,de"

get_installed_models_aliases() {
    local aliases=()
    while IFS= read -r line; do
        # Split the jq output into the component name and description
        local first_component_name="${line%%|*}"
        local alias="${line#*|}"

        # Check if the component is installed by checking if its directory exists
        if [[ -d "$SNAP_COMPONENTS/$first_component_name" ]]; then
            aliases+=("$alias")
        fi
    done < <(
        # Output format per item: "first-component-name|alias"
        modelctl list-models --format=json | jq -r '.models[] | "\(.components[0])|\(.alias)"'
    )
    # Return a comma-separated list of aliases
    (IFS=,; echo "${aliases[*]}") 
}

echo "Launching adapter..."

# Get the alias of the active model
active_model_alias=$(modelctl list-models --format=json | jq -r '. as $root | .models[] | select(.name == $root."active-model") | .alias')

# Get installed models
installed_model_aliases_str=$(get_installed_models_aliases)

set -x
$SNAP/bin/whisperlive-adapter serve \
    --backend-host "$BACKEND_HOST" \
    --backend-port "$BACKEND_PORT" \
    --host "$ADAPTER_HOST" \
    --port "$ADAPTER_PORT" \
    --model "$active_model_alias" \
    --language "$DEFAULT_LANG" \
    --allowed-models "$installed_model_aliases_str" \
    --allowed-languages "$ALLOWED_LANGUAGES"

set +x
echo "Adapter terminated."
