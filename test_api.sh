#!/usr/bin/env zsh

# Robust API test script for the Gin backend
BASE_URL="http://localhost:3000"

echo "Running API tests against $BASE_URL"

# Fail on unset variables, but don't exit on failing commands; we'll check return codes.
set -uo pipefail

tmpfile=$(mktemp)
cleanup() { rm -f "$tmpfile" }
trap cleanup EXIT

echo "1) GET /users (expect empty array or list)"
code=$(curl -sS -o "$tmpfile" -w "%{http_code}" ${BASE_URL}/users || true)
echo "Body:"; cat "$tmpfile"; echo "HTTP $code"

echo "\n2) POST /users (create user)"
read -r -d '' CREATE_PAYLOAD <<'JSON'
{
  "member_code":"LBK001234",
  "membership_level":"Gold",
  "name":"สมชาย",
  "surname":"ใจดี",
  "phone":"081-234-5678",
  "email":"somchai@example.com",
  "registration_date":"2023-06-15",
  "remaining_points":15420
}
JSON

code=$(curl -sS -o "$tmpfile" -w "%{http_code}" -X POST -H "Content-Type: application/json" -d "$CREATE_PAYLOAD" ${BASE_URL}/users || true)
echo "Response:"; cat "$tmpfile"; echo "HTTP $code"

if [[ "$code" -eq 201 || "$code" -eq 200 ]]; then
  ID=$(jq -r '.id' "$tmpfile")
else
  echo "Create returned HTTP $code; searching for existing user by member_code"
  # Get users list and find the one with the member_code
  curl -sS -o "$tmpfile" ${BASE_URL}/users || true
  ID=$(jq -r --arg mc "LBK001234" '.[] | select(.member_code==$mc) | .id' "$tmpfile" 2>/dev/null || echo "")
  if [[ -z "$ID" || "$ID" == "null" ]]; then
    echo "Could not create or find existing user. Exiting."
    exit 1
  fi
fi

echo "Created/Found user id: $ID"

echo "\n3) GET /users (expect list with created user)"
code=$(curl -sS -o "$tmpfile" -w "%{http_code}" ${BASE_URL}/users || true)
echo "Body:"; cat "$tmpfile"; echo "HTTP $code"

echo "\n4) GET /users/$ID (expect created user)"
code=$(curl -sS -o "$tmpfile" -w "%{http_code}" ${BASE_URL}/users/$ID || true)
echo "Body:"; cat "$tmpfile"; echo "HTTP $code"

echo "\n5) PUT /users/$ID (update membership_level and remaining_points)"
UPDATE_PAYLOAD=$(jq '.membership_level="Platinum" | .remaining_points=20000' "$tmpfile" 2>/dev/null || echo "$CREATE_PAYLOAD")
code=$(curl -sS -o "$tmpfile" -w "%{http_code}" -X PUT -H "Content-Type: application/json" -d "$UPDATE_PAYLOAD" ${BASE_URL}/users/$ID || true)
echo "Body:"; cat "$tmpfile"; echo "HTTP $code"

echo "\n6) DELETE /users/$ID"
code=$(curl -sS -o "$tmpfile" -w "%{http_code}" -X DELETE ${BASE_URL}/users/$ID || true)
echo "HTTP $code"

echo "\n7) GET /users (expect empty or no user)"
code=$(curl -sS -o "$tmpfile" -w "%{http_code}" ${BASE_URL}/users || true)
echo "Body:"; cat "$tmpfile"; echo "HTTP $code"

echo "\nAll tests completed"

echo "\n8) Transfer tests"

# helpers
find_user_by_member() {
  local mc="$1"
  curl -sS ${BASE_URL}/users | jq -r --arg mc "$mc" '.[] | select(.member_code==$mc) | @json' | head -n1
}

create_user_payload() {
  local mc="$1"; shift
  cat <<JSON
{
  "member_code":"$mc",
  "membership_level":"Silver",
  "name":"Test",
  "surname":"User",
  "phone":"0810000000",
  "email":"${mc}@example.com",
  "registration_date":"2025-01-01",
  "remaining_points":100
}
JSON
}

ensure_user() {
  local mc="$1"
  # try find
  local found=$(find_user_by_member "$mc")
  if [[ -n "$found" ]]; then
    echo "$found"
    return 0
  fi
  # create
  local payload=$(create_user_payload "$mc")
  code=$(curl -sS -o "$tmpfile" -w "%{http_code}" -X POST -H "Content-Type: application/json" -d "$payload" ${BASE_URL}/users || true)
  if [[ "$code" -ge 200 && "$code" -lt 300 ]]; then
    cat "$tmpfile"
    echo
    jq -c . "$tmpfile"
    return 0
  fi
  # fallback: list and find
  curl -sS ${BASE_URL}/users -o "$tmpfile" || true
  jq -r --arg mc "$mc" '.[] | select(.member_code==$mc) | @json' "$tmpfile" | head -n1
}

MC_A="TEST_A_TRANSFER"
MC_B="TEST_B_TRANSFER"

echo "Ensure test users $MC_A and $MC_B exist"
UA_JSON=$(ensure_user "$MC_A")
UB_JSON=$(ensure_user "$MC_B")

UA_ID=$(echo "$UA_JSON" | jq -r '.id')
UB_ID=$(echo "$UB_JSON" | jq -r '.id')
UA_BAL=$(echo "$UA_JSON" | jq -r '.remaining_points')
UB_BAL=$(echo "$UB_JSON" | jq -r '.remaining_points')

echo "User A id=$UA_ID balance=$UA_BAL"
echo "User B id=$UB_ID balance=$UB_BAL"

AMOUNT=30
echo "Perform transfer $AMOUNT from A($UA_ID) -> B($UB_ID)"
payload=$(cat <<JSON
{
  "fromUserId": $UA_ID,
  "toUserId": $UB_ID,
  "amount": $AMOUNT,
  "note": "test transfer"
}
JSON
)
code=$(curl -sS -o "$tmpfile" -w "%{http_code}" -X POST -H "Content-Type: application/json" -d "$payload" ${BASE_URL}/transfers || true)
echo "Response:"; cat "$tmpfile"; echo "HTTP $code"
if [[ "$code" -ne 201 && "$code" -ne 200 ]]; then
  echo "Transfer failed (HTTP $code)"
  exit 1
fi

IDEM=$(jq -r '.transfer.idemKey // .transfer.idempotency_key // .transfer.id' "$tmpfile")
TID=$(jq -r '.transfer.transferId // .transfer.transfer_id // .transfer.transferId' "$tmpfile")
echo "Created transfer idemKey=$IDEM transferId=$TID"

echo "Verifying balances"
UA_NEW=$(curl -sS ${BASE_URL}/users/$UA_ID)
UB_NEW=$(curl -sS ${BASE_URL}/users/$UB_ID)
UA_NEW_BAL=$(echo "$UA_NEW" | jq -r '.remaining_points')
UB_NEW_BAL=$(echo "$UB_NEW" | jq -r '.remaining_points')

expect_A=$((UA_BAL - AMOUNT))
expect_B=$((UB_BAL + AMOUNT))
echo "Expected A balance: $expect_A got: $UA_NEW_BAL"
echo "Expected B balance: $expect_B got: $UB_NEW_BAL"
if [[ "$UA_NEW_BAL" -ne "$expect_A" || "$UB_NEW_BAL" -ne "$expect_B" ]]; then
  echo "Balance verification failed"
  exit 1
fi

echo "Checking transfer listing for user A"
curl -sS "${BASE_URL}/transfers?userId=$UA_ID" | jq .

echo "Checking GET /transfers/$IDEM"
curl -sS ${BASE_URL}/transfers/$IDEM | jq .

echo "Transfer test completed successfully"