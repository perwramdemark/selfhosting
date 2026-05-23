#!/bin/bash

TARGET_CONTAINER="$1"
CHECK_INTERVAL=10  # sekunder
# ntfy topic
NTFY_TOPIC="https://ntfy.w8k.site/scripts"

if [ -z "$TARGET_CONTAINER" ]; then
    echo "Usage: watchdog <container-name>"
    exit 1
fi

echo "Starting watchdog for container: $TARGET_CONTAINER"

while true; do
    # Kontrollera om containern körs
    if ! docker ps --format '{{.Names}}' | grep -q "^${TARGET_CONTAINER}$"; then
        echo "$(date): $TARGET_CONTAINER är inte igång. Startar om..."
        docker start "$TARGET_CONTAINER"
        curl -sS -o /dev/null -X POST "$NTFY_TOPIC" -d "✅ ${TARGET_CONTAINER} was successfully restarted."
    fi
    sleep $CHECK_INTERVAL
done

