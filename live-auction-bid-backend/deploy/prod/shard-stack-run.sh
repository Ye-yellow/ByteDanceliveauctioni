#!/usr/bin/env bash
set -euo pipefail

SERVER_DIR="${SERVER_DIR:-/opt/live-auction}"
COMPOSE_FILE="${COMPOSE_FILE:-$SERVER_DIR/backend/deploy/prod/docker-compose.shard-stack.yml}"
ACTION="${1:-status}"
export LIVE_AUCTION_ENV_FILE="${LIVE_AUCTION_ENV_FILE:-$SERVER_DIR/.env}"

require_file() {
  if [[ ! -f "$1" ]]; then
    echo "Missing required file: $1" >&2
    exit 1
  fi
}

compose() {
  docker compose -f "$COMPOSE_FILE" "$@"
}

wait_url() {
  local label="$1"
  local url="$2"
  local i
  for i in $(seq 1 40); do
    if curl -fsS "$url" >/dev/null; then
      echo "$label ready: $url"
      return 0
    fi
    sleep 2
  done
  echo "$label did not become ready: $url" >&2
  return 1
}

up() {
  require_file "$COMPOSE_FILE"
  : "${AUCTION_INSTANCE_ID:?AUCTION_INSTANCE_ID is required}"
  : "${AUCTION_CLUSTER_REGISTRY_JSON:?AUCTION_CLUSTER_REGISTRY_JSON is required}"
  compose up -d --no-build mysql redis consul auction-backend
  wait_url "shard backend" "http://127.0.0.1:${AUCTION_HTTP_PUBLISH:-18080}/readyz"
  compose ps
}

down() {
  require_file "$COMPOSE_FILE"
  compose stop auction-backend consul redis mysql || true
  compose rm -f auction-backend consul redis mysql || true
}

status() {
  require_file "$COMPOSE_FILE"
  compose ps || true
  curl -fsS -o /dev/null -w "backend: %{http_code}\n" "http://127.0.0.1:${AUCTION_HTTP_PUBLISH:-18080}/readyz" || true
  curl -fsS "http://127.0.0.1:${AUCTION_HTTP_PUBLISH:-18080}/clusterz" || true
  echo
}

case "$ACTION" in
  up)
    up
    ;;
  down|rollback)
    down
    ;;
  status)
    status
    ;;
  *)
    echo "Usage: $0 {up|down|rollback|status}" >&2
    exit 2
    ;;
esac
