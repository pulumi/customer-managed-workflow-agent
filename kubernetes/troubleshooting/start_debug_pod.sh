#!/usr/bin/env bash

# Default values
NAMESPACE="cmwa"
LABEL="app.kubernetes.io/component=pulumi-workflow"
CONTAINER_NAME="pulumi-workflow"
DEBUG_POD_SUFFIX="debug"

# Parse command line arguments
while getopts "n:l:c:s:" opt; do
  case $opt in
    n) NAMESPACE="$OPTARG" ;;
    l) LABEL="$OPTARG" ;;
    c) CONTAINER_NAME="$OPTARG" ;;
    s) DEBUG_POD_SUFFIX="$OPTARG" ;;
    \?) echo "Usage: $0 [-n namespace] [-l label] [-c container-name] [-s debug-suffix]" >&2; exit 1 ;;
  esac
done

# Validate namespace
if [ -z "$NAMESPACE" ]; then
  echo "Error: Namespace cannot be empty."
  exit 1
fi

echo "Searching for pods in namespace '$NAMESPACE' with label '$LABEL'..."

# Check if any matching pods exist
POD_COUNT=$(kubectl get pod -n "$NAMESPACE" -l "$LABEL" --no-headers | wc -l)

if [ "$POD_COUNT" -eq 0 ]; then
  echo "Error: No pods found matching label '$LABEL' in namespace '$NAMESPACE'"
  exit 1
fi

# Get the first matching pod name
POD=$(kubectl get pod -n "$NAMESPACE" -l "$LABEL" -o name | head -n 1)
POD_NAME=$(echo "$POD" | sed 's|pod/||')

echo "Found pod: $POD_NAME"

CONTAINER="containers[?(@.name==\"$CONTAINER_NAME\")]"
ORIGINAL_CMD=$(kubectl get "$POD" -n "$NAMESPACE" -o jsonpath="{.spec.$CONTAINER.command[*]} {.spec.$CONTAINER.args[*]}")
ORIGINAL_CMD="${ORIGINAL_CMD% }"

if [ -z "$ORIGINAL_CMD" ]; then
  echo "Error: Pod $POD_NAME has no container named '$CONTAINER_NAME' with a command"
  exit 1
fi

echo "Original command parsed: $ORIGINAL_CMD"

# Create a debug pod name
DEBUG_POD_NAME="${POD_NAME}-${DEBUG_POD_SUFFIX}"

echo "Creating debug pod: $DEBUG_POD_NAME"

# Create a copy of the deploy pod but replace the entrypoint with a shell
kubectl debug -n "$NAMESPACE" "$POD" -it --copy-to="$DEBUG_POD_NAME" --container="$CONTAINER_NAME" -- sh -c "cd /mnt/ && echo 'Original command was: $ORIGINAL_CMD' && echo 'Running in interactive mode. To execute original workflow, run:' && echo '$ORIGINAL_CMD' && bash"