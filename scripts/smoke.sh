#!/usr/bin/env bash

set -euo pipefail

mkdir -p ./dist
workspace="$(mktemp -d)"
trap 'rm -rf "$workspace"' EXIT

go test ./...
go vet ./...
go build -o ./dist/cartero ./cmd/cartero
./dist/cartero --plain --root "$workspace" workspace init
./dist/cartero init "$workspace/campaign.yaml"
./dist/cartero --plain --root "$workspace" validate -f "$workspace/campaign.yaml"
./dist/cartero --plain --root "$workspace" preview -f "$workspace/campaign.yaml"
./dist/cartero --plain --root "$workspace" template list

cat > "$workspace/audience.csv" <<'EOF'
email,display_name,department,title
analyst@example.com,Finance Analyst,Finance,Analyst
manager@example.com,Finance Manager,Finance,Manager
EOF
./dist/cartero --plain --root "$workspace" audience import --segment finance-emea --csv "$workspace/audience.csv"

cat > "$workspace/reported.eml" <<'EOF'
From: alerts@example.com
Subject: Review payroll account changes

Please review the attached account changes before end of day.
EOF
./dist/cartero --plain --root "$workspace" import clone -f "$workspace/reported.eml"
./dist/cartero --plain --root "$workspace" event record --campaign "Q2 Awareness Rehearsal" --email analyst@example.com --type reported
./dist/cartero --plain --root "$workspace" report export --format json
./dist/cartero --plain --root "$workspace" doctor
./dist/cartero --plain --root "$workspace" plugin list
