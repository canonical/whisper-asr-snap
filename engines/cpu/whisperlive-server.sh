#!/bin/bash

set -e

HOST=$(modelctl get whisperlive.host)
PORT=$(modelctl get whisperlive.port)
BACKEND="faster_whisper" # options: "tensorrt", "faster_whisper", "openvino"

NPROC=$(nproc)
CLIENT_MAX_CONNECTION_TIME=2147483647 #int32 max, about 64 years

echo "Activating python venv..."
source activate 

echo "Launching engine..."

set -x
python3 "$SERVER_RUN_SCRIPT" \
    --cache_path "$MODEL_DIR" \
    --host "$HOST" \
    --port "$PORT" \
    --backend "$BACKEND" \
    --max_connection_time "$CLIENT_MAX_CONNECTION_TIME" \
    --omp_num_threads "$NPROC"

set +x
echo "Engine terminated."
