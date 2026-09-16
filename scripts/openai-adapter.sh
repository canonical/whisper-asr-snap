#!/bin/bash

set -euo pipefail

# TODO: introduce the export-shared-configs script
# Export the configuration for content sharing
# This must be done each time the server is started to expose the actual configuration
# $SNAP/bin/export-shared-configs.sh

engine="$(modelctl status --format=json | jq -r .engine)"

ensure_unix_socket_in_shared_content() {
    local share_dir="$SNAP_COMMON/share/provider"
    local unix_socket_path=$(modelctl get http.unix-socket)
    if [ -n "$unix_socket_path" ] && [[ "$unix_socket_path" != "$share_dir/"* ]]; then
        echo "Unix socket path ($unix_socket_path) is not in the expected share directory ($share_dir)"
        exit 1
    fi
}

ensure_unix_socket_in_shared_content

exec modelctl run -- "$SNAP/engines/$engine/openai-adapter.sh" "$@"
