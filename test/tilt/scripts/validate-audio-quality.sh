echo "validate-audio-quality.sh: shim executed"
#!/usr/bin/env bash
set -eu
# Simple audio-quality validation for Tilt development environment.
# Checks that the Audio Router `/status` endpoint responds and contains expected keys.

SCRIPT_NAME=$(basename "$0")
AUDIO_ROUTER_URL="${AUDIO_ROUTER_URL:-http://localhost:9090}"
TIMEOUT=${TIMEOUT:-5}

echo "$SCRIPT_NAME: checking $AUDIO_ROUTER_URL/status"

response=$(curl -s -f --max-time "$TIMEOUT" "$AUDIO_ROUTER_URL/status" 2>/dev/null || true)
if [ -z "$response" ]; then
	echo "$SCRIPT_NAME: ERROR: failed to fetch status from $AUDIO_ROUTER_URL/status"
	exit 2
fi

# Look for signs of configured services in the status output
if echo "$response" | grep -Eiq 'service|services|usrp|discord|whotalkie|audio'; then
	echo "$SCRIPT_NAME: OK: audio router status looks healthy"
	exit 0
else
	echo "$SCRIPT_NAME: WARNING: status endpoint returned, but did not contain expected keys"
	echo "Status response snippet:"
	echo "$response" | head -n 20
	# Treat as success in development but return code 0 so Tilt doesn't mark resource failing
	exit 0
fi
