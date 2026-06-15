#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_PATH="${CONFIG_PATH:-$ROOT_DIR/config.yaml}"
ADDR="${ADDR:-0.0.0.0:8080}"

if [[ ! -f "$CONFIG_PATH" ]]; then
  cp "$ROOT_DIR/config.example.yaml" "$CONFIG_PATH"
  echo "created config: $CONFIG_PATH"
fi

HOST_IP="$(ipconfig getifaddr en0 2>/dev/null || true)"
if [[ -z "$HOST_IP" ]]; then
  HOST_IP="$(ipconfig getifaddr en1 2>/dev/null || true)"
fi
PORT="${ADDR##*:}"

cd "$ROOT_DIR"

if [[ -n "$HOST_IP" ]]; then
  echo "portal will be available on this machine: http://127.0.0.1:$PORT"
  echo "portal will be available on your LAN:      http://$HOST_IP:$PORT"
else
  echo "portal will listen on: http://$ADDR"
  echo "could not detect LAN IP automatically; check System Settings > Wi-Fi/Ethernet"
fi

echo "using config: $CONFIG_PATH"
echo "press Ctrl+C to stop"

exec go run ./cmd/portal -config "$CONFIG_PATH" -addr "$ADDR"
