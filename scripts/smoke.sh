#!/usr/bin/env bash
set -euo pipefail
PORT=${PORT:-$((30000 + RANDOM % 20000))}; DB=$(mktemp -t mockbridge-smoke).db; LOG=$(mktemp -t mockbridge-smoke).log
cleanup(){ kill ${PID:-0} 2>/dev/null || true; rm -f "$DB" "$LOG"; }; trap cleanup EXIT
MOCKBRIDGE_ADDRESS=":$PORT" MOCKBRIDGE_DB_PATH="$DB" ./bin/mockbridge >"$LOG" 2>&1 & PID=$!
for _ in $(seq 1 50); do curl -fsS "http://127.0.0.1:$PORT/api/mock/_ping" >/dev/null 2>&1 && break; sleep .1; done
curl -fsS "http://127.0.0.1:$PORT/admin/api/health" | grep -q '"code":0'; echo '[PASS] health'
curl -fsS -X POST "http://127.0.0.1:$PORT/admin/api/contracts" -H 'Content-Type: application/json' \
 -d '{"key":"smoke/hello","name":"Smoke","priority":100,"enabled":true}' | grep -q '"code":0'; echo '[PASS] create contract'
# Discover scenario id because SQLite IDs can differ.
SID=$(curl -fsS "http://127.0.0.1:$PORT/admin/api/contracts/smoke%2Fhello" | sed -n 's/.*"scenarios":\[{"id":\([0-9]*\).*/\1/p')
curl -fsS -X PUT "http://127.0.0.1:$PORT/admin/api/contracts/smoke%2Fhello/scenarios/$SID" -H 'Content-Type: application/json' \
 -d '{"name":"default","match_rules":{"method":"GET","path":"/api/mock/hello/{name}"},"response":{"status":200,"headers":[{"name":"Content-Type","value":"application/json"}],"body":"{\"hello\":\"{{path.name}}\"}"},"delay":{},"fault":{},"switch":{}}' | grep -q '"code":0'; echo '[PASS] configure scenario'
curl -fsS "http://127.0.0.1:$PORT/api/mock/hello/world" | grep -q '"hello":"world"'; echo '[PASS] mock response'
sleep .3
curl -fsS "http://127.0.0.1:$PORT/admin/api/records?contract_key=smoke%2Fhello" | grep -q '"matched":true'; echo '[PASS] record stored'
