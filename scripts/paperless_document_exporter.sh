#!/bin/bash

LOG_FILE="/home/houdini/scripts/paperless_document_exporter.log"
echo "[$(date)] Starting document export..." >> "$LOG_FILE"

cd /home/houdini/stacks/paperless-ngx || {
    echo "[$(date)] Failed to cd into /home/houdini/stacks/paperless-ngx" >> "$LOG_FILE"
    exit 1
}

docker compose exec webserver document_exporter ../export --compare-json --no-progress-bar >> "$LOG_FILE" 2>&1

if [ $? -eq 0 ]; then
    echo "[$(date)] Document export completed successfully." >> "$LOG_FILE"
else
    echo "[$(date)] Document export FAILED." >> "$LOG_FILE"
fi

