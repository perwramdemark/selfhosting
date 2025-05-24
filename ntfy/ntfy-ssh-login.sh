#!/bin/bash
if [ "${PAM_TYPE}" = "open_session" ]; then
  curl \
    -H prio:high \
    -H tags:warning \
    -d "SSH login: ${PAM_USER} on houdini from ${PAM_RHOST}" \
    ntfy.w8k.site/ssh-logins
fi
