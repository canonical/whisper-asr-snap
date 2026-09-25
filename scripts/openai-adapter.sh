#!/bin/bash

set -euo pipefail

status_json=$(modelctl status --format=json)
engine=$(echo "$status_json" | jq -r .engine)

exec modelctl run --share-provider -- "$SNAP/engines/$engine/openai-adapter.sh" "$@"
