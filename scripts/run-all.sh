#!/usr/bin/env bash

echo "Please Run Me on the root dir, not in scripts dir."

gateway_directory="output/gateway"
service_directory="output/services"

log_directory="log"

mkdir -p "$log_directory"

if [ ! -d "output" ]; then
    echo "Output dir does not exist, please run build script first."
fi

for gateway_file in "$gateway_directory"/*; do
    if [[ -x "$gateway_file" && -f "$gateway_file" ]]; then
        echo "Running $gateway_file"
        gateway_log_file="$log_directory"/"$(basename gateway_file)".log
        ./"$gateway_file" >> "$gateway_log_file" 2>&1 &
    fi
done

for service in "$service_directory"/*; do
    for service_file in "$service"/*; do
        if [[ -x "$service_file" && -f "$service_file" ]]; then
            echo "Running $service_file"
            service_log_file="$log_directory"/"$(basename service_file)".log
            ./"$service_file" >> "$service_log_file" 2>&1 &
        fi
    done
done
