#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo "--- Smart Process Audit (Column-Aware) ---"
echo "------------------------------------------"

for container in $(docker ps -q); do
    name=$(docker inspect --format '{{.Name}}' "$container" | sed 's/\///')
    
    # 1. Capture the raw ps output
    ps_output=$(docker exec "$container" ps aux 2>/dev/null)
    
    if [[ -z "$ps_output" ]]; then
        # Fallback to metadata if ps fails completely
        metadata_user=$(docker inspect --format '{{.Config.User}}' "$container")
        if [[ -z "$metadata_user" || "$metadata_user" == "0" || "$metadata_user" == "root" ]]; then
            echo -e "${RED}[!] WARNING:${NC} ${name} is ROOT (via Metadata)"
        else
            echo -e "${GREEN}[✓] SAFE:${NC} ${name} is ${metadata_user} (via Metadata)"
        fi
        continue
    fi

    # 2. Find which column 'USER' is in
    # This handles both (USER PID...) and (PID USER...) formats
    user_col=$(echo "$ps_output" | head -n 1 | awk '{for(i=1;i<=NF;i++) if($i=="USER") print i}')
    
    # 3. Extract all users from that column, excluding headers and root
    # We also exclude 'ps' and 'awk' processes
    effective_user=$(echo "$ps_output" | awk -v col="$user_col" 'NR>1 {print $col}' | grep -vE "(root|USER|ps|awk)" | head -n 1)

    # 4. Final Verdict
    if [[ -z "$effective_user" ]]; then
        echo -e "${RED}[!] WARNING:${NC} ${name} is truly running as ROOT."
    else
        echo -e "${GREEN}[✓] SAFE:${NC} ${name} app is running as: ${effective_user}"
    fi
done
