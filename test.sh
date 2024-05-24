#!/bin/bash

p2p_port=33333
ws_port=23333
rpc_port=13333
base_id="Hello"

for i in {1..10}
do
    id=$base_id$i
    echo "Running instance $i"
    go run main.go --remote tcp://192.168.1.110:33333 --id ${id} --p2p ${p2p_port} --ws ${ws_port} --rpc ${rpc_port} &
    ((p2p_port++))
    ((ws_port++))
    ((rpc_port++))
done