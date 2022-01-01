#!/bin/bash

. /root/state_env_vars.sh

haproxy_print_stats() {
    echo "show stat" | sudo nc -U /var/run/haproxy/admin.sock | cut -d "," -f 1-2,5-10,34-36 | column -s, -t
}

haproxy_run_cmd() {
    echo "${1}" | sudo nc -U /var/run/haproxy/admin.sock
}

haproxy_set_server_addr() {
    haproxy_run_cmd "set server ${1} addr ${2}"
}

haproxy_enable_server() {
    haproxy_run_cmd "set server ${1} state ready"
}

haproxy_disable_server() {
    haproxy_run_cmd "set server ${1} state maint"
}

haproxy_enable_deploy() {
    haproxy_run_cmd "set server be_usermux_mux/${1} state ready"
}

haproxy_disable_deploy() {
    haproxy_run_cmd "set server be_usermux_mux/${1} state maint"
}

haproxy_weight_deploy() {
    haproxy_run_cmd "set weight be_usermux_mux/${1} ${2}"
}

haproxy_save_state() {
    # refresh local state file
    haproxy_run_cmd "show servers state" > /etc/haproxy/haproxy.state

    # save state file to the state storage bucket
    yes | azcopy --source /etc/haproxy/haproxy.state \
      --destination "${AZURE_STATE_STORAGE_PATH}" \
      --dest-key "${AZURE_STATE_ACCESS_KEY}"
}

CUR_DEPLOY=""
# make an array to hold our list of deploy names
declare -a DEPLOY_LIST=()

haproxy_get_deploy_info() {
    # fetch state for current deployment from state storage
    # azcopy --source "${AZURE_STATE_STORAGE_PREFIX}deploy-${TARGET_DEPLOY}-muxlist" \
    #   --source-key "${AZURE_STATE_ACCESS_KEY}"
    #   --destination "/etc/haproxy/state/" \

    CUR_DEPLOY=$(echo "$(haproxy_run_cmd "show servers state")" | python -c '
import sys, json
state_headers = None
active_deploy_id = None

for server_state in sys.stdin:
  state_array = server_state.split(" ")
  if len(state_array) <= 1:
    continue

  if state_headers is None:
    state_headers = state_array[1:]  # remove stray hash
    continue

  server_status_dict = dict(zip(state_headers, state_array))
  if server_status_dict.get("srv_name", "").startswith("deploy_"):
    # if the 2 bit is set, the server is enabled
    if int(server_status_dict.get("srv_op_state", 0)) & 2:
      active_deploy_id = server_status_dict.get("srv_name")[-1:]
print(active_deploy_id)
')

    DEPLOY_LIST=("A" "B")
    # set target to deploy that isnt enabled
    TARGET_DEPLOY="A" && [ "${CUR_DEPLOY}" = "A" ] && TARGET_DEPLOY="B"
}

haproxy_get_release_ips() {
    #cuts out only sever ip, state flags
    echo "$(haproxy_run_cmd "show servers state")" | python -c '
import sys, json
state_headers = None
active_deploy_id = None
active_servers = []

for server_state in sys.stdin:
  state_array = server_state.split(" ")
  if len(state_array) <= 1:
    continue

  if state_headers is None:
    state_headers = state_array[1:]  # remove stray hash
    continue

  server_status_dict = dict(zip(state_headers, state_array))

  # first figure out which deploy we are on, then return the ips for that deploy
  if server_status_dict.get("srv_name", "").startswith("deploy_"):
    # if the 2 bit is set, the server is enabled
    if int(server_status_dict.get("srv_op_state", 0)) & 2:
      active_deploy_id = "be_usermux_{}".format(server_status_dict.get("srv_name")[-1:])
  # if were looking at a mux and its in the active deploy
  elif server_status_dict.get("srv_name", "").startswith("usermux") and \
         server_status_dict.get("be_name", "") == active_deploy_id:
    if int(server_status_dict.get("srv_op_state", 0)) & 2:
      active_servers.append(server_status_dict.get("srv_addr"))
print json.dumps({"muxIPs": active_servers})
'
}


haproxy_swap_deploy() {
    NEW_DEPLOY_IP_LIST="${1}"
    haproxy_get_deploy_info

    # # if cur deploy is target deploy, we have nothing to do
    # if [ "$TARGET_DEPLOY" = "$CUR_DEPLOY" ]; then
    #     return
    # fi

    for DEPLOY_NAME in "${DEPLOY_LIST[@]}"; do
        IS_TARGET_DEPLOY=false; [ "$DEPLOY_NAME" = "$TARGET_DEPLOY" ] && IS_TARGET_DEPLOY=true
        IS_CURRENT_DEPLOY=false; [ "$DEPLOY_NAME" = "$CUR_DEPLOY" ] && IS_CURRENT_DEPLOY=true

        if $IS_TARGET_DEPLOY; then
            # look up IPs for new deploy and set addresses
            # TODO: loop through NEW_DEPLOY_IP_LIST with index counter and set server addresses
            n=0
            IFS=',' read -ra ADDR <<< "$NEW_DEPLOY_IP_LIST"
            for ip in "${ADDR[@]}"; do
                (( n++ ))

                be_servername="be_usermux_${DEPLOY_NAME}/usermux${n}"
                haproxy_set_server_addr "${be_servername}" "${ip}"
                haproxy_enable_server "${be_servername}"
            done

            echo "CHANGING DEPLOY TO $DEPLOY_NAME"
            haproxy_enable_deploy "deploy_${DEPLOY_NAME}"
        fi

        if $IS_CURRENT_DEPLOY; then
            echo "DISABLING DEPLOY $DEPLOY_NAME"
            haproxy_disable_deploy "deploy_${DEPLOY_NAME}"
        fi
    done



    # save state
    haproxy_save_state

}

# haproxy_set_server_addr "be_usermux_A/usermux1" "10.46.1.16"
# haproxy_set_server_addr "be_usermux_A/usermux2" "10.46.1.17"

# haproxy_enable_server "be_usermux_A/usermux1"
# haproxy_enable_server "be_usermux_A/usermux2"

# haproxy_enable_deploy "deploy_A"
# haproxy_disable_deploy "deploy_B"
