#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE_DIR="$(cd "$ROOT_DIR/.." && pwd)"
BACKEND_DIR="${BACKEND_DIR:-$ROOT_DIR}"
ADMIN_DIR="${ADMIN_DIR:-$WORKSPACE_DIR/live-auction-bid-frontend}"
H5_DIR="${H5_DIR:-$WORKSPACE_DIR/live-auction-user-h5}"

SERVER_HOST="${SERVER_HOST:-120.79.7.110}"
SERVER_USER="${SERVER_USER:-root}"
SERVER_DIR="${SERVER_DIR:-/opt/live-auction}"
SSH_KEY_SOURCE="${SSH_KEY:-/home/ye/OpenClaw/workspace/plans/yexieer.pem}"
SSH_KEY_RUNTIME="${SSH_KEY_RUNTIME:-/tmp/live-auction-yexieer.pem}"
WORK_DIR="${WORK_DIR:-/tmp/live-auction-deploy-fast}"
INSTALL_NGINX_CONF="${INSTALL_NGINX_CONF:-0}"

RUN_BACKEND=1
RUN_ADMIN=1
RUN_H5=1
RUN_TESTS=1

usage() {
  cat <<'EOF'
Usage: scripts/deploy-prod.sh [options]

Options:
  --frontend-only   Only build/upload admin + H5 static files.
  --backend-only    Only test/build/upload backend image.
  --skip-tests      Skip go test before backend image build.
  -h, --help        Show this help.

Environment:
  SSH_KEY=/path/to/key.pem
  SERVER_HOST=120.79.7.110
  SERVER_USER=root
  SERVER_DIR=/opt/live-auction
  INSTALL_NGINX_CONF=1
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --frontend-only)
      RUN_BACKEND=0
      RUN_ADMIN=1
      RUN_H5=1
      ;;
    --backend-only)
      RUN_BACKEND=1
      RUN_ADMIN=0
      RUN_H5=0
      ;;
    --skip-tests)
      RUN_TESTS=0
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

require_file() {
  if [[ ! -f "$1" ]]; then
    echo "Missing required file: $1" >&2
    exit 1
  fi
}

run() {
  echo
  echo "==> $*"
  "$@"
}

ensure_clean_git() {
  local repo_dir="$1"
  local name="$2"
  if [[ -n "$(git -C "$repo_dir" status --porcelain)" ]]; then
    echo "$name has uncommitted changes. Commit or stash them before deploy." >&2
    git -C "$repo_dir" status --short >&2
    exit 1
  fi
}

prepare_ssh_key() {
  require_file "$SSH_KEY_SOURCE"
  cp "$SSH_KEY_SOURCE" "$SSH_KEY_RUNTIME"
  chmod 600 "$SSH_KEY_RUNTIME"
}

remote() {
  ssh -i "$SSH_KEY_RUNTIME" -o IdentitiesOnly=yes -o BatchMode=yes "$SERVER_USER@$SERVER_HOST" "$@"
}

upload() {
  scp -i "$SSH_KEY_RUNTIME" -o IdentitiesOnly=yes -o BatchMode=yes "$@" "$SERVER_USER@$SERVER_HOST:$SERVER_DIR/uploads/"
}

prepare_packages() {
  rm -rf "$WORK_DIR"
  mkdir -p "$WORK_DIR"

  if [[ "$RUN_BACKEND" -eq 1 ]]; then
    ensure_clean_git "$BACKEND_DIR" "backend"
    if [[ "$RUN_TESTS" -eq 1 ]]; then
      run env GOCACHE=/tmp/live-auction-go-cache go -C "$BACKEND_DIR" test ./...
    fi
    run docker compose -f "$BACKEND_DIR/deploy/docker-compose.yml" build auction-backend
    run docker tag live-auction-bid-backend:local live-auction-bid-backend:prod
    run git -C "$BACKEND_DIR" archive --format=tar.gz -o "$WORK_DIR/backend-src.tar.gz" HEAD
    run bash -lc "docker save live-auction-bid-backend:prod | gzip -1 > '$WORK_DIR/backend-image.tar.gz'"
  fi

  if [[ "$RUN_ADMIN" -eq 1 ]]; then
    ensure_clean_git "$ADMIN_DIR" "admin frontend"
    run npm --prefix "$ADMIN_DIR" run build
    run tar -czf "$WORK_DIR/admin-dist.tar.gz" -C "$ADMIN_DIR/dist" .
  fi

  if [[ "$RUN_H5" -eq 1 ]]; then
    ensure_clean_git "$H5_DIR" "H5 frontend"
    run npm --prefix "$H5_DIR" run build
    run tar -czf "$WORK_DIR/h5-dist.tar.gz" -C "$H5_DIR/dist" .
  fi
}

deploy_remote() {
  run remote "mkdir -p '$SERVER_DIR/uploads' '$SERVER_DIR/www' '$SERVER_DIR/backend'"

  local upload_files=()
  [[ "$RUN_BACKEND" -eq 1 ]] && upload_files+=("$WORK_DIR/backend-src.tar.gz" "$WORK_DIR/backend-image.tar.gz")
  [[ "$RUN_ADMIN" -eq 1 ]] && upload_files+=("$WORK_DIR/admin-dist.tar.gz")
  [[ "$RUN_H5" -eq 1 ]] && upload_files+=("$WORK_DIR/h5-dist.tar.gz")

  run upload "${upload_files[@]}"

  local remote_script
  remote_script=$(cat <<EOF
set -euo pipefail
cd '$SERVER_DIR'

if [[ $RUN_BACKEND -eq 1 ]]; then
  rm -rf backend.new
  mkdir -p backend.new
  tar -xzf uploads/backend-src.tar.gz -C backend.new
  gzip -dc uploads/backend-image.tar.gz | docker load
  cp backend.new/deploy/prod/docker-compose.yml docker-compose.yml
  cp backend.new/deploy/.env .env
  cp backend.new/deploy/prod/live-auction.nginx.conf live-auction.nginx.conf
  if [[ '$INSTALL_NGINX_CONF' == '1' ]]; then
    cp live-auction.nginx.conf /etc/nginx/sites-available/live-auction
    ln -sf /etc/nginx/sites-available/live-auction /etc/nginx/sites-enabled/live-auction
  fi
  rm -rf backend
  mv backend.new backend
fi

if [[ $RUN_ADMIN -eq 1 ]]; then
  rm -rf www/admin.new
  mkdir -p www/admin.new
  tar -xzf uploads/admin-dist.tar.gz -C www/admin.new
  rm -rf www/admin
  mv www/admin.new www/admin
fi

if [[ $RUN_H5 -eq 1 ]]; then
  rm -rf www/h5.new
  mkdir -p www/h5.new
  tar -xzf uploads/h5-dist.tar.gz -C www/h5.new
  rm -rf www/h5
  mv www/h5.new www/h5
fi

chmod 755 '$SERVER_DIR' '$SERVER_DIR/www' '$SERVER_DIR/www/admin' '$SERVER_DIR/www/h5'
docker compose up -d --no-build
systemctl reload nginx

for i in \$(seq 1 30); do
  if curl -fsS http://127.0.0.1/readyz; then
    break
  fi
  if [[ "\$i" -eq 30 ]]; then
    echo "backend did not become ready" >&2
    exit 1
  fi
  sleep 2
done
curl -fsS -o /tmp/live-auction-h5.html -w "\\nH5 %{http_code} %{size_download}\\n" http://127.0.0.1/
curl -fsS -o /tmp/live-auction-admin.html -w "ADMIN %{http_code} %{size_download}\\n" http://127.0.0.1:8080/
docker compose ps
EOF
)

  run remote "bash -s" <<<"$remote_script"
}

prepare_ssh_key
prepare_packages
deploy_remote

echo
echo "Deploy complete:"
echo "  H5:    http://$SERVER_HOST/"
echo "  Admin: http://$SERVER_HOST:8080/"
