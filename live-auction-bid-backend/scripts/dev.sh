#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "Start backend: cd $ROOT/backend && go run ./cmd/auction-server"
echo "Start frontend: cd $ROOT/frontend && npm install && npm run dev"
