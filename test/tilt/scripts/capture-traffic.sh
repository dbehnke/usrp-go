echo "capture-traffic.sh: shim executed"
#!/usr/bin/env bash
set -eu
# Network capture helper used in Tilt dev. Runs tcpdump for a short interval if available.

SCRIPT_NAME=$(basename "$0")
OUT_DIR=${OUT_DIR:-/tmp}
CAP_FILE="$OUT_DIR/tilt_network_capture_$(date +%s).pcap"
DURATION=${DURATION:-10}

if ! command -v tcpdump >/dev/null 2>&1; then
	echo "$SCRIPT_NAME: tcpdump not installed; skipping capture"
	exit 0
fi

echo "$SCRIPT_NAME: capturing network traffic to $CAP_FILE for $DURATION seconds"
sudo tcpdump -i any -w "$CAP_FILE" -nn 'udp or tcp' &
pid=$!
sleep $DURATION
kill $pid 2>/dev/null || true
wait $pid 2>/dev/null || true

if [ -f "$CAP_FILE" ]; then
	echo "$SCRIPT_NAME: capture written to $CAP_FILE"
	# Keep small captures in dev; do not fail
	exit 0
else
	echo "$SCRIPT_NAME: capture failed or file missing"
	exit 0
fi
