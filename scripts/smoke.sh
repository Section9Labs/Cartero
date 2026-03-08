#!/usr/bin/env bash

set -euo pipefail

mkdir -p ./dist

go test ./...
go vet ./...
go build -o ./dist/cartero ./cmd/cartero
./dist/cartero validate --plain -f ./configs/campaign.example.yaml
./dist/cartero preview --plain -f ./configs/campaign.example.yaml
./dist/cartero doctor --plain
