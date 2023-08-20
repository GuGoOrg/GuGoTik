#!/usr/bin/env bash

echo "Please Run Me on the root dir, not in scripts dir."

if [ -d "output" ]; then
    echo "Output dir existed, deleting and recreating..."
    rm -rf output
fi
mkdir -p output/services

pushd src/services || exit

for i in *; do
    if [ "$i" != "health" ]; then
        name="$i"
        capName="${name^}"
        cd "$i" || exit
        go build -o "../../../output/services/$i/${capName}Service"
        cd ..
    fi
done

popd || exit

mkdir -p output/gateway

cd src/web || exit

go build -o "../../output/gateway/Gateway"

echo "OK!"
