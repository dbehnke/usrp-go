#!/usr/bin/env bash
set -euo pipefail
# Small shim that prefers the Docker Compose plugin ("docker compose").
# Falls back to the standalone "docker-compose" binary when available.
# If neither is present (common in Docker-in-Docker images), the shim
# will attempt to download a standalone Docker Compose v2 binary to
# /tmp and execute it (requires curl or wget and network access).
# Usage: ./scripts/docker_compose.sh [args...]

DOCKER_COMPOSE_BIN="/tmp/docker-compose-standalone"

run_docker_compose_plugin() {
  # Use the Compose plugin via the docker CLI if available.
  if command -v docker >/dev/null 2>&1; then
    # 'docker compose version' returns 0 when the plugin is available.
    if docker compose version >/dev/null 2>&1; then
      exec docker compose "$@"
    fi
  fi
  return 1
}

run_local_docker_compose() {
  if command -v docker-compose >/dev/null 2>&1; then
    exec docker-compose "$@"
  fi
  return 1
}

download_standalone_compose() {
  # Try to download an x86_64 / aarch64 compatible docker-compose v2 binary from GitHub.
  # Pick a reasonably recent, stable release. This is a best-effort fallback.
  local version="v2.20.2"
  local os
  local arch

  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64) arch="x86_64" ;;
    aarch64|arm64) arch="aarch64" ;;
    *) arch="$arch" ;;
  esac

  local url="https://github.com/docker/compose/releases/download/${version}/docker-compose-${os}-${arch}"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$DOCKER_COMPOSE_BIN" "$url" || return 1
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$DOCKER_COMPOSE_BIN" "$url" || return 1
  else
    return 1
  fi

  chmod +x "$DOCKER_COMPOSE_BIN"
  return 0
}

print_diagnostic() {
  echo "Error: neither 'docker compose' (plugin) nor 'docker-compose' found in PATH." >&2
  echo "Detected environment:" >&2
  echo "  docker: $(command -v docker || echo '(not found)')" >&2
  echo "  docker-compose: $(command -v docker-compose || echo '(not found)')" >&2
  echo "If you are running inside a container (Docker-in-Docker), ensure the container image includes the Docker CLI and the Compose plugin, or mount the host Docker socket and provide a compose binary." >&2
  echo "As a fallback the repository provides a shim that can download a standalone Docker Compose binary if network access and curl/wget are available." >&2
}

# Try plugin first
if run_docker_compose_plugin "$@"; then
  exit 0
fi

# Then try local docker-compose
if run_local_docker_compose "$@"; then
  exit 0
fi

# Best-effort: attempt to download a standalone docker-compose v2 binary and run it
if download_standalone_compose; then
  exec "$DOCKER_COMPOSE_BIN" "$@"
fi

print_diagnostic
exit 127
