#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web/portal"
CONFIG_PATH="${CONFIG_PATH:-$ROOT_DIR/config.yaml}"
BACKEND_ADDR="${BACKEND_ADDR:-0.0.0.0:8080}"
BACKEND_PROXY_HOST="${BACKEND_PROXY_HOST:-127.0.0.1}"
FRONTEND_HOST="${FRONTEND_HOST:-0.0.0.0}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"
AUTO_NPM_INSTALL="${AUTO_NPM_INSTALL:-1}"

backend_pid=""

cleanup() {
  if [[ -n "$backend_pid" ]] && kill -0 "$backend_pid" 2>/dev/null; then
    echo
    echo "stopping portal backend..."
    kill "$backend_pid" 2>/dev/null || true
    wait "$backend_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

if [[ ! -f "$CONFIG_PATH" ]]; then
  cp "$ROOT_DIR/config.example.yaml" "$CONFIG_PATH"
  echo "created config: $CONFIG_PATH"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required but was not found in PATH" >&2
  exit 1
fi
if ! command -v npm >/dev/null 2>&1; then
  echo "npm is required but was not found in PATH" >&2
  exit 1
fi

HOST_IP="$(ipconfig getifaddr en0 2>/dev/null || true)"
if [[ -z "$HOST_IP" ]]; then
  HOST_IP="$(ipconfig getifaddr en1 2>/dev/null || true)"
fi
if [[ -z "$HOST_IP" ]] && command -v hostname >/dev/null 2>&1; then
  HOST_IP="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
fi

BACKEND_PORT="${BACKEND_ADDR##*:}"
API_PROXY_TARGET="${API_PROXY_TARGET:-http://$BACKEND_PROXY_HOST:$BACKEND_PORT}"

cd "$ROOT_DIR"
echo "starting portal backend on $BACKEND_ADDR..."
go run ./cmd/portal -config "$CONFIG_PATH" -addr "$BACKEND_ADDR" &
backend_pid="$!"

sleep 1
if ! kill -0 "$backend_pid" 2>/dev/null; then
  echo "portal backend failed to start" >&2
  wait "$backend_pid"
fi

cd "$WEB_DIR"
if [[ ! -d node_modules && "$AUTO_NPM_INSTALL" != "0" ]]; then
  if [[ -f package-lock.json ]]; then
    echo "installing frontend dependencies with npm ci..."
    npm ci
  else
    echo "installing frontend dependencies with npm install..."
    npm install
  fi
fi

cat <<EOF

portal backend API:
  local: http://127.0.0.1:$BACKEND_PORT/api/health

portal web frontend:
  local: http://127.0.0.1:$FRONTEND_PORT
EOF

if [[ -n "$HOST_IP" ]]; then
  cat <<EOF
  LAN:   http://$HOST_IP:$FRONTEND_PORT
EOF
else
  echo "  LAN:   could not detect LAN IP automatically"
fi

cat <<EOF

Open the portal web frontend URL above. The frontend proxies /api requests to:
  $API_PROXY_TARGET
Press Ctrl+C to stop both backend and frontend.
EOF

VITE_API_PROXY_TARGET="$API_PROXY_TARGET" exec npm run dev -- --host "$FRONTEND_HOST" --port "$FRONTEND_PORT"
