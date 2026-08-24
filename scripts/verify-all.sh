#!/usr/bin/env bash
# Verify all registered routes: regenerate imports, build, then live-fetch every
# route Example and report item counts. Usage: ./scripts/verify-all.sh [port]
set -uo pipefail

PORT="${1:-12891}"
cd "$(dirname "$0")/.."

echo "== regenerating route imports =="
go run scripts/generate-routes.go || exit 1

echo "== building =="
go build -o /tmp/rsshub-verify ./cmd || exit 1

echo "== extracting examples =="
grep -rhoE 'Example:\s+"[^"]+"' routes/ | sed -E 's/Example:\s+"([^"]+)"/\1/' | sort -u > /tmp/rsshub-examples.txt
total=$(wc -l < /tmp/rsshub-examples.txt)
echo "routes to verify: $total"

sed "s/port: \"1200\"/port: \"$PORT\"/" config.yaml > /tmp/rsshub-verify-config.yaml
SYNDI_CONFIG=/tmp/rsshub-verify-config.yaml /tmp/rsshub-verify > /tmp/rsshub-verify.log 2>&1 &
SRV=$!
sleep 3

ok=0; empty=0; fail=0
: > /tmp/rsshub-verify-results.txt
while IFS= read -r ex; do
  code=$(curl -s -o /tmp/rsshub-body.xml -w '%{http_code}' -m 45 "http://127.0.0.1:$PORT/$ex")
  if [ "$code" != "200" ]; then
    echo "FAIL $code $ex" >> /tmp/rsshub-verify-results.txt
    fail=$((fail+1))
    continue
  fi
  n=$(grep -c '<item>' /tmp/rsshub-body.xml 2>/dev/null || echo 0)
  if [ "${n:-0}" -gt 0 ]; then
    echo "OK $n $ex" >> /tmp/rsshub-verify-results.txt
    ok=$((ok+1))
  else
    echo "EMPTY $n $ex" >> /tmp/rsshub-verify-results.txt
    empty=$((empty+1))
  fi
done < /tmp/rsshub-examples.txt

kill $SRV 2>/dev/null
echo "=== results: ok=$ok empty=$empty fail=$fail total=$total ==="
sort /tmp/rsshub-verify-results.txt | awk '{print $1}' | uniq -c
