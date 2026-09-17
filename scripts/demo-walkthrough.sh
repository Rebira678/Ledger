#!/usr/bin/env bash
# Ledger — scripted demo walkthrough (master prompt Phase 9).
# Boots a throwaway Postgres + the Ledger server, then walks the API end-to-end.
# Usage: ./scripts/demo-walkthrough.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PG_CONTAINER=ledger-demo-pg
PORT=15432
B="http://localhost:8080"

cleanup() {
  [[ -n "${SERVER_PID:-}" ]] && kill "$SERVER_PID" 2>/dev/null || true
  docker rm -f "$PG_CONTAINER" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "▶ 1. Starting throwaway Postgres ($PG_CONTAINER)…"
docker rm -f "$PG_CONTAINER" >/dev/null 2>&1 || true
docker run -d --name "$PG_CONTAINER" -e POSTGRES_USER=ledger -e POSTGRES_PASSWORD=ledger \
  -e POSTGRES_DB=ledger -p "$PORT":5432 postgres:16-alpine >/dev/null
for i in $(seq 1 60); do
  docker exec "$PG_CONTAINER" pg_isready -U ledger >/dev/null 2>&1 && break
  sleep 1
done

echo "▶ 2. Building and starting the Ledger server…"
go -C "$ROOT" build -o /tmp/ledger-demo-server ./cmd/ledger-server
export LEDGER_DATABASE_URL="postgres://ledger:ledger@localhost:$PORT/ledger?sslmode=disable"
export LEDGER_JWT_SIGNING_KEY="demo-signing-key-0123456789abcdef-0123456789abcdef"
export LEDGER_HTTP_ADDR=":8080"
export LEDGER_ENV="development"
# Give Postgres an extra beat after pg_isready (it restarts once during init).
sleep 4
/tmp/ledger-demo-server > /tmp/ledger-demo-server.log 2>&1 &
SERVER_PID=$!
for i in $(seq 1 20); do
  curl -sf "$B/healthz" >/dev/null 2>&1 && break
  sleep 1
done

echo "▶ 3. Liveness + readiness:"
curl -s "$B/healthz"; echo
curl -s "$B/readyz"; echo

echo "▶ 4. Register + login:"
curl -s -X POST "$B/v1/auth/register" -H 'Content-Type: application/json' \
  -d '{"email":"abel@example.com","password":"s3curepass!"}' | head -c 200; echo
LOGIN=$(curl -s -X POST "$B/v1/auth/login" -H 'Content-Type: application/json' \
  -d '{"email":"abel@example.com","password":"s3curepass!"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json;print(json.load(sys.stdin)['access_token'])")
echo "   access token acquired."

echo "▶ 5. Pair an Android device:"
PAIR=$(curl -s -X POST "$B/v1/devices/pair" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"device_model":"Pixel 6a","os_version":"14"}')
DKEY=$(echo "$PAIR" | python3 -c "import sys,json;print(json.load(sys.stdin)['device_api_key'])")
echo "   device key issued (dk_live_…)."

echo "▶ 6. Ingest a CBE credit SMS (201 expected):"
curl -s -w "\n   HTTP %{http_code}\n" -X POST "$B/v1/ingest/sms" \
  -H "Authorization: Bearer $TOKEN" -H "X-Device-Key: $DKEY" -H 'Content-Type: application/json' \
  -d '{"sender_id":"CBE","body":"CBE: You have received 500.00 ETB from ALMAZ TESFAYE (Acc 1000123456789) on 14/09/2026 07:41. Ref: TXN12345678. Balance: 4,820.50 ETB","received_at":"2026-09-14T07:41:00Z","client_message_id":"c8f1e2a0"}'

echo "▶ 7. Re-send the SAME message — idempotent 409 (contract §3):"
curl -s -o /dev/null -w "   HTTP %{http_code} (409 expected)\n" -X POST "$B/v1/ingest/sms" \
  -H "Authorization: Bearer $TOKEN" -H "X-Device-Key: $DKEY" -H 'Content-Type: application/json' \
  -d '{"sender_id":"CBE","body":"CBE: You have received 500.00 ETB from ALMAZ TESFAYE (Acc 1000123456789) on 14/09/2026 07:41. Ref: TXN12345678. Balance: 4,820.50 ETB","received_at":"2026-09-14T07:41:00Z","client_message_id":"c8f1e2a0"}'

echo "▶ 8. Ingest a Telebirr debit:"
curl -s -w "\n   HTTP %{http_code}\n" -X POST "$B/v1/ingest/sms" \
  -H "Authorization: Bearer $TOKEN" -H "X-Device-Key: $DKEY" -H 'Content-Type: application/json' \
  -d '{"sender_id":"TELEBIRR","body":"Telebirr: You have paid 45.00 ETB to SAFARICOM DATA (Ref: T987654321) on 14/09/2026 08:15. Balance: 1,205.00 ETB","received_at":"2026-09-14T08:15:00Z","client_message_id":"tb1"}'

echo "▶ 9. Unknown sender → review queue, 202 (FR-3.3):"
curl -s -w "\n   HTTP %{http_code}\n" -X POST "$B/v1/ingest/sms" \
  -H "Authorization: Bearer $TOKEN" -H "X-Device-Key: $DKEY" -H 'Content-Type: application/json' \
  -d '{"sender_id":"MYSTERY-BANK","body":"totally unknown format","received_at":"2026-09-14T08:20:00Z","client_message_id":"mx1"}'

echo "▶ 10. Low-confidence transaction queues a clarifying question:"
curl -s -X POST "$B/v1/ingest/sms" -H "Authorization: Bearer $TOKEN" -H "X-Device-Key: $DKEY" \
  -H 'Content-Type: application/json' \
  -d '{"sender_id":"CBE","body":"CBE: You have paid 800.00 ETB to YILMA TESFAYE on 14/09/2026 08:00. Ref: TXC1.","received_at":"2026-09-14T08:00:00Z","client_message_id":"cc1"}' >/dev/null
CLAR=$(curl -s "$B/v1/agent/clarifications" -H "Authorization: Bearer $TOKEN")
echo "   $CLAR"
CLR_ID=$(echo "$CLAR" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['items'][0]['clarification_id'])")

echo "▶ 11. Answer the clarification (learning loop, FR-5.2):"
curl -s -X POST "$B/v1/agent/clarifications/$CLR_ID/respond" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"answer":"Rent"}'; echo

echo "▶ 12. Dashboard summary:"
curl -s "$B/v1/dashboard/summary" -H "Authorization: Bearer $TOKEN"; echo

echo "▶ 13. Transactions list:"
curl -s "$B/v1/transactions" -H "Authorization: Bearer $TOKEN" | head -c 600; echo

echo
echo "✅ Demo complete. The dashboard UI is served at $B/login (same server)."
echo "   (Log in using email: abel@example.com / password: s3curepass!)"
echo
read -p "Press [Enter] to shut down the server and exit..."
