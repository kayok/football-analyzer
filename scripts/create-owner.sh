#!/usr/bin/env bash
set -euo pipefail
if [[ -z "${OWNER_EMAIL:-}" ]]; then
  echo 'Set OWNER_EMAIL in .env first.' >&2
  exit 1
fi
read -r -s -p 'Owner password (9–72 ASCII characters, no spaces): ' owner_password </dev/tty
printf '\n' >&2
printf '%s' "$owner_password" | (cd backend && go run ./cmd/manage create-owner)
unset owner_password
