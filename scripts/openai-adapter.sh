#!/bin/bash

set -euo pipefail


engine="$(modelctl status --format=json | jq -r .engine)"

share_provider() {
    local status_json=$(modelctl status --format=json)
    local provider_env_content="SNAP_NAME=$SNAP_NAME\nSNAP_INSTANCE_NAME=$SNAP_INSTANCE_NAME\n"

    # if status_json.entrypoints has a "openai" entry
    if [ "$(echo "$status_json" | jq -e '.entrypoints | has("openai")')" = "true" ]; then
        local openai_base_url=$(echo "$status_json" | jq -r '.entrypoints.openai.url')
        provider_env_content+="OPENAI_BASE_URL=$openai_base_url\n"
    fi

    # if status_json.entrypoints has a "openai-unix" entry
    if [ "$(echo "$status_json" | jq -e '.entrypoints | has("openai-unix")')" = "true" ]; then
        local full_socket_path=$(echo "$status_json" | jq -r '.entrypoints."openai-unix"."unix-socket"')
        local socket_filename=$(basename "$full_socket_path")
        local socket_url=$(echo "$status_json" | jq -r '.entrypoints."openai-unix"."unix-socket-url"')
        provider_env_content+="OPENAI_UNIX_BASE_URL=$socket_filename\n"
        provider_env_content+="OPENAI_UNIX_SOCKET_URL=$socket_url\n"
    fi

    local share_dir="$SNAP_COMMON/share/provider"
    if [ ! -d "$share_dir" ]; then
        echo "Share directory does not exist, creating it: $share_dir"
        mkdir -p "$share_dir"
    fi
    
    local env_file_path="$share_dir/provider.env"
    echo -e "$provider_env_content" > "$env_file_path"
}

ensure_unix_socket_in_shared_content() {
    local share_dir="$SNAP_COMMON/share/provider"
    local unix_socket_path=$(modelctl get http.unix-socket)
    if [ -n "$unix_socket_path" ] && [[ "$unix_socket_path" != "$share_dir/"* ]]; then
        echo "Unix socket path ($unix_socket_path) is not in the expected share directory ($share_dir)"
        exit 1
    fi
}

ensure_unix_socket_in_shared_content
share_provider

# TODO: use modelctl run --share-provider instead of share_provider() once every feature is implemented
exec modelctl run -- "$SNAP/engines/$engine/openai-adapter.sh" "$@"
