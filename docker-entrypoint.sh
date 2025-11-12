#!/bin/sh
set -e

# Default to running kvserver if no command specified
if [ "$#" -eq 0 ]; then
    exec ./kvserver
fi

# Execute the provided command
exec "$@"
