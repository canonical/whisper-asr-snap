#!/bin/bash

set -e

PORT=$(snapctl get engine-port)
NPROC=$(nproc)
CLIENT_MAX_CONNECTION_TIME=2147483647 #int32 max, about 64 years

#TODO: get backend from snap config
BACKEND="faster_whisper" #Available backends: "tensorrt", "faster_whisper", "openvino"

echo "Activating python venv..."
source activate 

echo "Launching engine..."
echo "Using port=$PORT, nproc=$NPROC, backend=$BACKEND"

# NOTE: host is hardcoded to 0.0.0.0 in this run script
# TODO: rewrite the run script to accept host as a parameter, and get it from snap config
python3 $SNAP/bin/run_server.py \
    --cache_path "$SNAP/models" \
    --port "$PORT" \
    --backend "$BACKEND" \
    --max_connection_time "$CLIENT_MAX_CONNECTION_TIME" \
    --omp_num_threads "$NPROC" 

echo "Engine terminated."