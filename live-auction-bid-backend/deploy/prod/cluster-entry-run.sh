#!/usr/bin/env bash
set -euo pipefail

SERVER_DIR="${SERVER_DIR:-/opt/live-auction}"
BASE_COMPOSE="${BASE_COMPOSE:-$SERVER_DIR/docker-compose.yml}"
GATEWAY_COMPOSE="${GATEWAY_COMPOSE:-$SERVER_DIR/backend/deploy/prod/docker-compose.gateway.yml}"
MONITORING_COMPOSE="${MONITORING_COMPOSE:-$SERVER_DIR/backend/deploy/prod/docker-compose.cluster-monitoring.yml}"
NATS_COMPOSE="${NATS_COMPOSE:-$SERVER_DIR/backend/deploy/prod/docker-compose.nats.yml}"
SINGLE_NGINX="${SINGLE_NGINX:-$SERVER_DIR/live-auction.nginx.conf}"
GATEWAY_NGINX="${GATEWAY_NGINX:-$SERVER_DIR/backend/deploy/prod/live-auction.gateway.nginx.conf.template}"
NGINX_SITE="${NGINX_SITE:-/etc/nginx/sites-available/live-auction}"
ACTION="${1:-status}"
export LIVE_AUCTION_ENV_FILE="${LIVE_AUCTION_ENV_FILE:-$SERVER_DIR/.env}"

compose() {
  docker compose -f "$BASE_COMPOSE" -f "$GATEWAY_COMPOSE" "$@"
}

compose_monitoring() {
  docker compose -f "$BASE_COMPOSE" -f "$MONITORING_COMPOSE" "$@"
}

compose_nats() {
  docker compose -f "$BASE_COMPOSE" -f "$NATS_COMPOSE" "$@"
}

single_compose() {
  docker compose -f "$BASE_COMPOSE" "$@"
}

require_file() {
  if [[ ! -f "$1" ]]; then
    echo "Missing required file: $1" >&2
    exit 1
  fi
}

install_nginx() {
  local source="$1"
  require_file "$source"
  cp "$source" "$NGINX_SITE"
  ln -sf "$NGINX_SITE" /etc/nginx/sites-enabled/live-auction
  nginx -t
  systemctl reload nginx
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
  require_file "$BASE_COMPOSE"
  require_file "$GATEWAY_COMPOSE"
  require_file "$GATEWAY_NGINX"
  : "${AUCTION_CLUSTER_REGISTRY_JSON:?AUCTION_CLUSTER_REGISTRY_JSON is required}"
  compose up -d --no-build auction-shard-gateway
  wait_url "shard gateway" "http://127.0.0.1:${AUCTION_GATEWAY_PUBLISH:-18081}/healthz"
  install_nginx "$GATEWAY_NGINX"
  wait_url "unified entry" "http://127.0.0.1/healthz"
  curl -fsS "http://127.0.0.1/clusterz" >/dev/null
  compose ps auction-shard-gateway
}

monitoring_up() {
  require_file "$BASE_COMPOSE"
  require_file "$MONITORING_COMPOSE"
  compose_monitoring up -d --no-build prometheus grafana
  compose_monitoring ps prometheus grafana
}

nats_up() {
  require_file "$BASE_COMPOSE"
  require_file "$NATS_COMPOSE"
  compose_nats up -d --no-build nats
  wait_url "nats monitor" "http://127.0.0.1:${AUCTION_NATS_MONITOR_HTTP_PORT:-18222}/healthz"
  compose_nats ps nats
}

rollback() {
  require_file "$BASE_COMPOSE"
  if [[ -f "$SINGLE_NGINX" ]]; then
    install_nginx "$SINGLE_NGINX"
  else
    echo "Single-node nginx config not found at $SINGLE_NGINX; skipping nginx rollback" >&2
  fi
  compose stop auction-shard-gateway || true
  compose rm -f auction-shard-gateway || true
  if [[ -f "$NATS_COMPOSE" ]]; then
    compose_nats stop nats || true
    compose_nats rm -f nats || true
  fi
  single_compose up -d --no-build auction-backend
  single_compose up -d --no-build prometheus grafana || true
  wait_url "single backend" "http://127.0.0.1:18080/readyz"
  wait_url "single unified entry" "http://127.0.0.1/readyz"
  single_compose ps auction-backend
}

status() {
  if [[ -f "$GATEWAY_COMPOSE" ]]; then
    compose ps auction-shard-gateway || true
  fi
  if [[ -f "$NATS_COMPOSE" ]]; then
    compose_nats ps nats || true
  fi
  curl -fsS -o /dev/null -w "gateway: %{http_code}\n" "http://127.0.0.1:${AUCTION_GATEWAY_PUBLISH:-18081}/healthz" || true
  curl -fsS -o /dev/null -w "nats monitor: %{http_code}\n" "http://127.0.0.1:${AUCTION_NATS_MONITOR_HTTP_PORT:-18222}/healthz" || true
  curl -fsS -o /dev/null -w "unified healthz: %{http_code}\n" "http://127.0.0.1/healthz" || true
  curl -fsS "http://127.0.0.1/clusterz" || true
  echo
}

case "$ACTION" in
  up)
    up
    ;;
  monitoring-up)
    monitoring_up
    ;;
  nats-up)
    nats_up
    ;;
  down|rollback)
    rollback
    ;;
  status)
    status
    ;;
  *)
    echo "Usage: $0 {up|monitoring-up|nats-up|down|rollback|status}" >&2
    exit 2
    ;;
esac
