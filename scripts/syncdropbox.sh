#!/bin/bash

/usr/bin/rclone sync mydropbox:/ /srv/dev-disk-by-uuid-0690e9fe-6a27-4996-832c-be833cab6446/Dropbox --log-file=/home/houdini/scripts/rclone.log --log-level NOTICE

# Check the exit status of rclone command
if [ $? -eq 0 ]; then
  curl --silent --output /dev/null http://houdini:3001/api/push/QEWxc3vt76?status=up&msg=OK&ping= >> /dev/null 2>&1
else
  curl --silent --output /dev/null http://houdini:3001/api/push/QEWxc3vt76?status=down&msg=Sync%20NOK&ping= >> /dev/null 2>&1
  exit 1
fi
