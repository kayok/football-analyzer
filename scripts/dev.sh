#!/usr/bin/env bash
set -euo pipefail
(cd backend && go build -o ../bin/api ./cmd/api)
./bin/api &
api_pid=$!
(cd frontend; exec npm run dev) &
web_pid=$!
trap 'kill "$api_pid" "$web_pid" 2>/dev/null || true; wait || true' EXIT INT TERM
wait -n "$api_pid" "$web_pid"
