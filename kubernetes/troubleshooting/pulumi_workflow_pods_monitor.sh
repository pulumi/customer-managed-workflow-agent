#!/bin/sh

# Default values
NAMESPACE="cmwa"
INTERVAL=1
MAX_LOGS=10
LOG_DIR="logs"
LABEL_SELECTOR="app.kubernetes.io/component=pulumi-workflow"

# Parse command line arguments
while getopts "n:i:m:d:l:" opt; do
  case $opt in
    n) NAMESPACE="$OPTARG" ;;
    i) INTERVAL="$OPTARG" ;;
    m) MAX_LOGS="$OPTARG" ;;
    d) LOG_DIR="$OPTARG" ;;
    l) LABEL_SELECTOR="$OPTARG" ;;
    \?) echo "Usage: $0 [-n namespace] [-i interval] [-m max_logs] [-d log_dir] [-l label_selector]" >&2; exit 1 ;;
  esac
done

# Create log directory if it doesn't exist
mkdir -p "$LOG_DIR"

# Log rotation function
rotate_logs() {
    cd "$LOG_DIR" || exit
    TOTAL_LOGS=$(ls pulumi_workflow_pods_monitor_*.log 2>/dev/null | wc -l)
    while [ "$TOTAL_LOGS" -ge "$MAX_LOGS" ]; do
        OLDEST_LOG=$(ls -t pulumi_workflow_pods_monitor_*.log | tail -1)
        rm "$OLDEST_LOG"
        TOTAL_LOGS=$((TOTAL_LOGS - 1))
    done
}

# Ensure namespace is set
if [ -z "$NAMESPACE" ]; then
  echo "Error: Namespace must be set."
  exit 1
fi

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOGFILE="$LOG_DIR/pulumi_workflow_pods_monitor_${TIMESTAMP}.log"

echo "Starting Pulumi workflow pods monitoring. Log file: $LOGFILE"
echo "Monitoring pods in namespace: $NAMESPACE"
echo "Using label selector: $LABEL_SELECTOR"
echo "Started at: $(date)" > "$LOGFILE"
echo "Target namespace: $NAMESPACE" >> "$LOGFILE"
echo "Label selector: $LABEL_SELECTOR" >> "$LOGFILE"
echo "" >> "$LOGFILE"

# Infinite loop to monitor the pod
while true; do
  rotate_logs
  
  PODS=$(kubectl get pods -n "$NAMESPACE" -l "$LABEL_SELECTOR" --no-headers -o custom-columns=":metadata.name")
  
  if [ -z "$PODS" ]; then
    echo "$(date): No matching pulumi-workflow pods found" >> "$LOGFILE"
  else
    for pod in $PODS; do
      echo "===============================================" >> "$LOGFILE"
      echo "$(date): Pod information for $pod" >> "$LOGFILE"
      echo "===============================================" >> "$LOGFILE"
      
      echo "--- RESOURCE USAGE ---" >> "$LOGFILE"
      kubectl top pod "$pod" -n "$NAMESPACE" >> "$LOGFILE" 2>&1 || echo "Failed to get resource usage" >> "$LOGFILE"
      
      echo "--- NETWORK CONNECTIVITY ---" >> "$LOGFILE"
      kubectl exec -n "$NAMESPACE" "$pod" -- nc -vz kubernetes.default.svc 443 >> "$LOGFILE" 2>&1 || echo "Failed to check network connectivity" >> "$LOGFILE"
      
      echo "--- DESCRIBE OUTPUT ---" >> "$LOGFILE"
      kubectl describe pod "$pod" -n "$NAMESPACE" >> "$LOGFILE" 2>&1
      
      echo "--- LOGS OUTPUT ---" >> "$LOGFILE"
      kubectl logs -n "$NAMESPACE" "$pod" >> "$LOGFILE" 2>&1 || echo "Failed to get logs" >> "$LOGFILE"

      echo "--- EVENTS OUTPUT ---" >> "$LOGFILE"
      kubectl get events -n "$NAMESPACE" --field-selector involvedObject.name="$pod" >> "$LOGFILE" 2>&1 || echo "Failed to get events" >> "$LOGFILE"
      
      echo "" >> "$LOGFILE"
    done
  fi
  
  sleep "$INTERVAL"
done
