#!/bin/bash

set -euo pipefail

# Python venv with huggingface_hub
sudo apt-get install -y python3-venv

python3 -m venv .venv
source .venv/bin/activate

pip install --upgrade pip
pip install -U huggingface_hub

mkdir -p components

# Faster Whisper Base
hf download Systran/faster-whisper-base \
    --local-dir components/model-faster-whisper-base/
