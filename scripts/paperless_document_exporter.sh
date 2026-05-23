#!/bin/bash

NTFY_TOPIC="https://ntfy.w8k.site/scripts"
LOG_FILE="/home/houdini/scripts/paperless_document_exporter.log"

echo "[$(date)] Starting document export..." >> "$LOG_FILE"

cd /home/houdini/stacks/paperless-ngx || {
    echo "[$(date)] Failed to cd into /home/houdini/stacks/paperless-ngx" >> "$LOG_FILE"
    exit 1
}

docker compose exec webserver document_exporter ../export --compare-json --no-progress-bar >> "$LOG_FILE" 2>&1

if [ $? -eq 0 ]; then
  echo "[$(date)] Document export completed successfully." >> "$LOG_FILE"
  curl -sS -o /dev/null -X POST "$NTFY_TOPIC" -d "✅ Export of documents from Paperless run successfully."
else
  curl -sS -o /dev/null -X POST "$NTFY_TOPIC" -d "❌ Error when trying to export documents from paperless"
  echo "[$(date)] Document export FAILED." >> "$LOG_FILE"
fi

