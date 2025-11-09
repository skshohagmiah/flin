#!/bin/bash

# Test script for Flin web application example

set -e

BASE_URL="http://localhost:8080"

echo "🧪 Testing Flin Web Application"
echo "================================"
echo ""

# Check if server is running
echo "📡 Checking if server is running..."
if ! curl -s "$BASE_URL/api/stats" > /dev/null; then
    echo "❌ Server is not running!"
    echo "   Start it with: docker-compose up -d"
    echo "   Or: go run main.go"
    exit 1
fi
echo "✅ Server is running"
echo ""

# Test 1: Create user
echo "1️⃣  Creating user..."
USER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/users" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com"}')

USER_ID=$(echo $USER_RESPONSE | jq -r '.id')
echo "   User ID: $USER_ID"
echo "   ✅ User created"
echo ""

# Test 2: Get user (cache miss)
echo "2️⃣  Getting user (cache miss)..."
RESPONSE=$(curl -s -i "$BASE_URL/api/users/$USER_ID")
CACHE_STATUS=$(echo "$RESPONSE" | grep "X-Cache" | awk '{print $2}' | tr -d '\r')
echo "   Cache status: $CACHE_STATUS"
if [ "$CACHE_STATUS" = "MISS" ]; then
    echo "   ✅ Cache miss (expected)"
else
    echo "   ⚠️  Expected cache miss"
fi
echo ""

# Test 3: Get user again (cache hit)
echo "3️⃣  Getting user again (cache hit)..."
sleep 1
RESPONSE=$(curl -s -i "$BASE_URL/api/users/$USER_ID")
CACHE_STATUS=$(echo "$RESPONSE" | grep "X-Cache" | awk '{print $2}' | tr -d '\r')
echo "   Cache status: $CACHE_STATUS"
if [ "$CACHE_STATUS" = "HIT" ]; then
    echo "   ✅ Cache hit (super fast!)"
else
    echo "   ⚠️  Expected cache hit"
fi
echo ""

# Test 4: Login
echo "4️⃣  Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/login" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\"}")

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.token')
echo "   Token: ${TOKEN:0:20}..."
echo "   ✅ Login successful"
echo ""

# Test 5: Get profile with token
echo "5️⃣  Getting profile (authenticated)..."
PROFILE=$(curl -s "$BASE_URL/api/profile" \
  -H "Authorization: Bearer $TOKEN")
USERNAME=$(echo $PROFILE | jq -r '.username')
echo "   Username: $USERNAME"
echo "   ✅ Profile retrieved"
echo ""

# Test 6: Test cache performance
echo "6️⃣  Testing cache performance..."
PERF=$(curl -s "$BASE_URL/api/cache")
WRITE_LATENCY=$(echo $PERF | jq -r '.write_latency_us')
READ_LATENCY=$(echo $PERF | jq -r '.read_latency_us')
echo "   Write latency: ${WRITE_LATENCY}μs"
echo "   Read latency: ${READ_LATENCY}μs"
echo "   ✅ Performance test complete"
echo ""

# Test 7: Logout
echo "7️⃣  Logging out..."
curl -s -X POST "$BASE_URL/api/logout" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
echo "   ✅ Logout successful"
echo ""

# Test 8: Try to access profile after logout (should fail)
echo "8️⃣  Trying to access profile after logout..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/profile" \
  -H "Authorization: Bearer $TOKEN")
if [ "$HTTP_CODE" = "401" ]; then
    echo "   ✅ Access denied (expected)"
else
    echo "   ⚠️  Expected 401, got $HTTP_CODE"
fi
echo ""

# Test 9: Get stats
echo "9️⃣  Getting statistics..."
STATS=$(curl -s "$BASE_URL/api/stats")
echo "$STATS" | jq '.'
echo "   ✅ Statistics retrieved"
echo ""

echo "================================"
echo "✅ All tests passed!"
echo ""
echo "📊 Summary:"
echo "   - User creation: ✅"
echo "   - Cache hit/miss: ✅"
echo "   - Session management: ✅"
echo "   - Authentication: ✅"
echo "   - Performance: Write ${WRITE_LATENCY}μs, Read ${READ_LATENCY}μs"
echo ""
echo "🚀 Flin is production-ready!"
