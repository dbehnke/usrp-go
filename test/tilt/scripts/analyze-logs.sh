echo "analyze-logs.sh: shim executed"
#!/usr/bin/env bash
set -eu
# Lightweight log analysis for Tilt development. Scans recent pod logs for ERROR/WARN

SCRIPT_NAME=$(basename "$0")
NAMESPACE=${NAMESPACE:-default}
POD_LIST=$(kubectl get pods -o name -n "$NAMESPACE" 2>/dev/null || true)

if [ -z "$POD_LIST" ]; then
	echo "$SCRIPT_NAME: No pods found in namespace $NAMESPACE"
	exit 0
fi

echo "$SCRIPT_NAME: scanning logs for ERROR/WARN entries (this may take a moment)"
issues=0
for pod in $POD_LIST; do
	name=${pod#pod/}
	echo "--- $name ---"
	if kubectl logs "$name" -n "$NAMESPACE" --tail=200 2>/dev/null | egrep -i "error|warn|panic" >/tmp/${name}.matches; then
		echo "Found potential issues in $name:"; sed -n '1,80p' /tmp/${name}.matches
		issues=$((issues+1))
	else
		echo "No obvious errors in $name"
	fi
done

if [ $issues -gt 0 ]; then
	echo "$SCRIPT_NAME: Completed with $issues pods showing possible issues"
	# Do not fail Tilt on log noise; exit 0 but print a summary
	exit 0
else
	echo "$SCRIPT_NAME: No issues detected in recent logs"
	exit 0
fi
