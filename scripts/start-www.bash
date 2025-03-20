#!/usr/bin/env bash

echo "Starting web servers"
CODE_ROOT_DIR="$(pwd)/.."

if ! command -v python3 &> /dev/null
then
    echo "python3 could not be found, please install python3"
    exit 1
fi

if ! command -v tmux &> /dev/null
then
    echo "tmux could not be found, please install tmux"
    exit 1
fi

cd $CODE_ROOT_DIR || { echo "Failed to change directory to $CODE_ROOT_DIR"; exit 1; }

tmux new-session -d -s sramek-web-servers

# Start transportation web server
transportation_port=8066
echo "Starting sramek-transportation web server"

cmd="python3 -m http.server ${transportation_port} -b localhost --directory ./www/sramek-transportation/"
tmux send-keys -t sramek-web-servers "$cmd" C-m # C-m is the enter key

echo "Running sramek-transportation on port ${transportation_port}"

# Start garden center web server
garden_center_port=8067
echo "Starting sramek-garden-center web server"

tmux new-window -t sramek-web-servers
cmd="python3 -m http.server ${garden_center_port} -b localhost --directory ./www/sramek-garden-center/"
tmux send-keys -t sramek-web-servers:1 "$cmd" C-m # C-m is the enter key

echo "Running sramek-garden-center on port ${garden_center_port}"


echo "Web servers started. Use 'tmux attach -t sramek-web-servers' to view the output."