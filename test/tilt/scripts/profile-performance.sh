echo "profile-performance.sh: shim executed"
#!/usr/bin/env bash
set -eu
# Simple performance profiling harness used in dev. Sends repeated requests to measure latency.

SCRIPT_NAME=$(basename "$0")
AUDIO_ROUTER_URL="${AUDIO_ROUTER_URL:-http://localhost:9090/status}"
ITERATIONS=${ITERATIONS:-20}
SLEEP=${SLEEP:-1}

echo "$SCRIPT_NAME: profiling $AUDIO_ROUTER_URL for $ITERATIONS requests"
total=0
success=0
for i in $(seq 1 $ITERATIONS); do
	start=$(date +%s%3N)
	if curl -s -f --max-time 5 "$AUDIO_ROUTER_URL" >/dev/null 2>&1; then
		success=$((success+1))
	fi
	end=$(date +%s%3N)
	dt=$((end-start))
	total=$((total+dt))
	sleep $SLEEP
done

if [ $success -eq 0 ]; then
	echo "$SCRIPT_NAME: no successful requests"
	exit 1
fi
avg=$((total / success))
echo "$SCRIPT_NAME: $success/$ITERATIONS successful, avg latency ${avg}ms"
exit 0
