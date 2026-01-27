#!/bin/bash

# Default values
NAMESPACE="cmwa"
POD_SELECTOR=""
POD_PATTERN=""
INTERVAL=1
OUTPUT_DIR="./pod_logs"
MAX_DURATION=3600  # 1 hour
COMPLETION_STATUS="Succeeded,Completed,Failed"

# Parse command line arguments
while getopts "n:s:p:i:d:t:c:" opt; do
  case $opt in
    n) NAMESPACE="$OPTARG" ;;
    s) POD_SELECTOR="$OPTARG" ;;
    p) POD_PATTERN="$OPTARG" ;;
    i) INTERVAL="$OPTARG" ;;
    d) OUTPUT_DIR="$OPTARG" ;;
    t) MAX_DURATION="$OPTARG" ;;
    c) COMPLETION_STATUS="$OPTARG" ;;
    \?) echo "Usage: $0 [-n namespace] [-s label-selector] [-p name-pattern] [-i interval] [-d output_dir] [-t max_duration] [-c completion_status]" >&2; exit 1 ;;
  esac
done

# Create output directory
mkdir -p "$OUTPUT_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BASE_LOG="${OUTPUT_DIR}/pod_monitor_${TIMESTAMP}"

# Validate inputs
if [[ -z "$POD_SELECTOR" && -z "$POD_PATTERN" ]]; then
    echo "Error: Must specify either a label selector (-s) or name pattern (-p)"
    exit 1
fi

# Function to find matching pods
find_pods() {
    if [[ -n "$POD_SELECTOR" ]]; then
        kubectl get pods -n "$NAMESPACE" -l "$POD_SELECTOR" --no-headers -o custom-columns=":metadata.name"
    else
        kubectl get pods -n "$NAMESPACE" --no-headers -o custom-columns=":metadata.name" | grep -E "$POD_PATTERN"
    fi
}

# Function to check pod health
check_pod_health() {
    local pod=$1
    echo "--- HEALTH CHECK ---" >> "${BASE_LOG}_${pod}.log"
    
    # Check readiness/liveness probes
    kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.conditions}' | jq . >> "${BASE_LOG}_${pod}.log"
    
    # Check container statuses
    echo "Container Statuses:" >> "${BASE_LOG}_${pod}.log"
    kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses}' | jq . >> "${BASE_LOG}_${pod}.log"
}

START_TIME=$(date +%s)

echo "Starting pod monitoring in namespace: $NAMESPACE"
echo "Logs will be saved to: $OUTPUT_DIR"

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
    
    if [ "$ELAPSED_TIME" -ge "$MAX_DURATION" ]; then
        echo "Maximum monitoring duration reached. Exiting."
        exit 0
    fi

    PODS=$(find_pods)
    
    if [[ -z "$PODS" ]]; then
        echo "No matching pods found. Waiting..."
    else
        for pod in $PODS; do
            echo "Monitoring pod: $pod"
            
            # Get pod status
            POD_STATUS=$(kubectl get pod -n "$NAMESPACE" "$pod" -o jsonpath='{.status.phase}')
            echo "[$(date)] Status: $POD_STATUS" >> "${BASE_LOG}_${pod}.log"
            
            # Resource usage
            echo "--- RESOURCE USAGE ---" >> "${BASE_LOG}_${pod}.log"
            kubectl top pod "$pod" -n "$NAMESPACE" >> "${BASE_LOG}_${pod}.log" 2>&1
            
            # Health check
            check_pod_health "$pod"
            
            # Pod details
            echo "--- POD DETAILS ---" >> "${BASE_LOG}_${pod}.log"
            kubectl describe pod "$pod" -n "$NAMESPACE" >> "${BASE_LOG}_${pod}.log"
            
            # Recent logs
            echo "--- RECENT LOGS ---" >> "${BASE_LOG}_${pod}.log"
            kubectl logs --tail=50 -n "$NAMESPACE" "$pod" >> "${BASE_LOG}_${pod}.log" 2>&1
            
            # Check for completion
            if [[ "$COMPLETION_STATUS" == *"$POD_STATUS"* ]]; then
                echo "Pod $pod reached status $POD_STATUS. Stopping monitoring."
                exit 0
            fi
        done
    fi
    
    sleep "$INTERVAL"
done
