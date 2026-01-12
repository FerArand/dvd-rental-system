#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080}"

echo "Health:"
curl -s "$API_BASE/health" | jq .

echo "Login as staff:"
STAFF_JSON=$(curl -s -X POST "$API_BASE/api/auth/login" -H "Content-Type: application/json" \
  -d '{"email":"Mike.Hillyer@sakilastaff.com", "role":"staff"}')
echo "$STAFF_JSON" | jq .
STAFF_ID=$(echo "$STAFF_JSON" | jq -r .id)

echo "Login as customer:"
CUST_JSON=$(curl -s -X POST "$API_BASE/api/auth/login" -H "Content-Type: application/json" \
  -d '{"email":"mary.smith@sakilacustomer.org", "role":"customer"}')
echo "$CUST_JSON" | jq .
CUSTOMER_ID=$(echo "$CUST_JSON" | jq -r .id)

echo "Find available inventory for film_id=1"
INV_IDS=$(curl -s "$API_BASE/api/inventory/available?film_id=1" | jq -r '.inventory_ids[0]')
echo "Inventory chosen: $INV_IDS"

echo "Create rental:"
RENT_JSON=$(curl -s -X POST "$API_BASE/api/rentals" -H "Content-Type: application/json" \
  -d "{\"customer_id\":$CUSTOMER_ID, \"inventory_id\":$INV_IDS, \"staff_id\":$STAFF_ID}")
echo "$RENT_JSON" | jq .
RENTAL_ID=$(echo "$RENT_JSON" | jq -r .rental_id)

echo "Report: customer rentals"
curl -s "$API_BASE/api/reports/customer/$CUSTOMER_ID/rentals" | jq '.[0]'

echo "Report: not returned (should include our rental)"
curl -s "$API_BASE/api/reports/not-returned" | jq '.[0]'

echo "Return rental:"
curl -s -X POST "$API_BASE/api/returns/$RENTAL_ID" | jq .

echo "Top rented:"
curl -s "$API_BASE/api/reports/top-rented?limit=5" | jq .

echo "Revenue by staff:"
curl -s "$API_BASE/api/reports/revenue-by-staff" | jq .

echo "Try cancel (should fail because already returned):"
curl -s -X POST "$API_BASE/api/rentals/$RENTAL_ID/cancel" | jq .
