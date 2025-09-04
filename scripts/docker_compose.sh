#!/usr/bin/env bash
set -euo pipefail
# Small shim that prefers the Docker Compose plugin ("docker compose")
# but falls back to the standalone "docker-compose" binary when necessary.
# Usage: ./scripts/docker_compose.sh [args...]

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  exec docker compose "$@"
elif command -v docker-compose >/dev/null 2>&1; then
  exec docker-compose "$@"
else
  echo "Error: neither 'docker compose' nor 'docker-compose' found in PATH" >&2
  exit 127
fi
