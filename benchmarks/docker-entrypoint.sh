#!/bin/sh
set -e

# Get hostname for Raft binding
HOSTNAME=$(hostname)

# Build command with flags from environment variables
CMD="./kvserver"

if [ -n "$NODE_ID" ]; then
    CMD="$CMD -node-id=$NODE_ID"
fi

if [ -n "$HTTP_PORT" ]; then
    CMD="$CMD -http=0.0.0.0:$HTTP_PORT"
fi

if [ -n "$RAFT_PORT" ]; then
    # Use hostname for Raft address so it's advertisable
    CMD="$CMD -raft=$HOSTNAME:$RAFT_PORT"
fi

if [ -n "$KV_PORT" ]; then
    CMD="$CMD -port=:$KV_PORT"
fi

if [ -n "$RAFT_DIR" ]; then
    CMD="$CMD -data=$RAFT_DIR"
fi

if [ -n "$JOIN_ADDR" ]; then
    CMD="$CMD -join=$JOIN_ADDR"
fi

if [ -n "$PARTITION_COUNT" ]; then
    CMD="$CMD -partitions=$PARTITION_COUNT"
fi

echo "Starting: $CMD"
exec $CMD
