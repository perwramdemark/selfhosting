#!/bin/bash

# ntfy topic
NTFY_TOPIC="https://ntfy.w8k.site/scripts"

# home directory for scripts
HOME_DIR=/home/houdini/scripts

# List of source directories
SOURCE_DIRS=(
  "/home/houdini/bazarr/backup"
  "/home/houdini/radarr/Backups"
  "/home/houdini/sonarr/Backups"
  "/home/houdini/prowlarr/Backups"
  "/home/houdini/readarr/Backups"
  "/home/houdini/paperless/export"
)

# Base destination directory
DEST_DIR="/srv/dev-disk-by-uuid-0690e9fe-6a27-4996-832c-be833cab6446/houdini/backups"

# Check if destination directory exists
if [ ! -d "$DEST_DIR" ]; then
  ERROR_MSG="Destination directory $DEST_DIR does not exist. Backup aborted."
  echo "$(date): $ERROR_MSG" >> $HOME_DIR/backup_copy.log
  curl -sS -o /dev/null -X POST "$NTFY_TOPIC" -d "❌ Error when trying to copy backups on houdini $ERROR_MSG"
  exit 1
fi

# Track success and errors
ERRORS=()

for SRC in "${SOURCE_DIRS[@]}"; do
  if [ -d "$SRC" ]; then
    APP_NAME=$(basename "$(dirname "$SRC")")
    APP_DEST="$DEST_DIR/$APP_NAME"

    mkdir -p "$APP_DEST"
    if [ $? -ne 0 ]; then
      ERR_MSG="Failed to create destination directory: $APP_DEST"
      ERRORS+=("$ERR_MSG")
      echo "$(date): $ERR_MSG" >> $HOME_DIR/backup_copy.log
      continue
    fi

    shopt -s globstar nullglob
    ALL_FILES=("$SRC"/**/*.*)

    if [ ${#ALL_FILES[@]} -gt 0 ]; then
      cp -u "${ALL_FILES[@]}" "$APP_DEST"

      if [ $? -ne 0 ]; then
        ERRORS+=("Failed to copy files from $SRC")
         echo "$(date): Failed to copy files from $SRC" >> $HOME_DIR/backup_copy.log
      else
        echo "$(date): Copied files from $SRC to $APP_DEST" >> $HOME_DIR/backup_copy.log
      fi
    else
      echo "$(date): No files found in $SRC" >> $HOME_DIR/backup_copy.log
    fi

  else
    ERRORS+=("Source directory not found: $SRC")
    echo "$(date): WARNING - $SRC not found" >> $HOME_DIR/backup_copy.log
  fi
done

# Notify via ntfy
if [ ${#ERRORS[@]} -eq 0 ]; then
  curl -sS -o /dev/null -X POST "$NTFY_TOPIC" -d "✅ Houdini backups copied successfully to NAS drives."
else
  MSG=$(printf "❌ Backup issues when running backup from houdini:\n\n%s\n" "$(printf '%s\n' "${ERRORS[@]}")")
  curl -sS -o /dev/null -X POST "$NTFY_TOPIC" -d "${MSG}"
fi
