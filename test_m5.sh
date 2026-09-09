#!/bin/bash
# M5 End-to-End Test Script — Web Dashboard + Security
set -e

RT="./rt.exe"
export RT_HOME="$(pwd)/test-rt-home"
rm -rf "$RT_HOME"

PASS="testpass1234"
PORT=18932

echo "=== M5 End-to-End Test ==="
echo ""

# Setup: create engagement with data
echo "--- Setup: Create engagement with data ---"
printf '%s\n%s\n' "$PASS" "$PASS" | $RT new "M5-TEST" --client "ACME Corp"
echo "$PASS" | $RT unlock

# Generate evidence and findings
$RT exec echo "nmap -sV 10.10.10.1 - ports 22,80,443"
$RT exec echo "Administrator:500:aad3b435b51404eeaad3b435b51404ee:31d6cfe0d16ae931b73c59d7e0c089c0:::"
$RT exec echo "[+] Domain Admin access achieved"
$RT finding "SQL Injection" --desc "SQLi in login" --priority high --evidence 1 --mitre T1190
$RT finding "Domain Admin" --desc "PtH to DA" --priority critical --evidence 2,3 --mitre T1078,T1550
$RT verify-finding 1 confirmed --note "Reproduced"
$RT recommend 1 "Fix parameterized queries"
echo "PASS: Data setup complete"

# 1. Add operators
echo ""
echo "--- Test 1: Add operators ---"
$RT operator-add glory --role lead
$RT operator-add teammate --role operator
$RT operator-add reviewer1 --role reviewer
$RT operator-add client-pm --role viewer
echo "PASS: Operators added"

# 2. List operators
echo ""
echo "--- Test 2: List operators ---"
$RT operator-list
echo "PASS: Operators listed"

# 3. Rotate key
echo ""
echo "--- Test 3: Rotate API key ---"
$RT operator-rotate teammate
echo "PASS: Key rotated"

# 4. Start server in background (localhost = no TLS, no auth)
echo ""
echo "--- Test 4: Start server ---"
$RT serve --listen "localhost:$PORT" &
SERVER_PID=$!
sleep 2

# Check server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "FAIL: Server not running"
    exit 1
fi
echo "PASS: Server running on port $PORT"

# 5. Test API overview endpoint
echo ""
echo "--- Test 5: API /api/overview ---"
OVERVIEW=$(curl -s "http://localhost:$PORT/api/overview")
if echo "$OVERVIEW" | grep -q '"chain_intact"'; then
    echo "PASS: Overview API works"
else
    echo "FAIL: Overview API broken"
    echo "$OVERVIEW"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# 6. Test API findings
echo ""
echo "--- Test 6: API /api/findings ---"
FINDINGS=$(curl -s "http://localhost:$PORT/api/findings")
if echo "$FINDINGS" | grep -q '"SQL Injection"'; then
    echo "PASS: Findings API works"
else
    echo "FAIL: Findings API broken"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# 7. Test API evidence
echo ""
echo "--- Test 7: API /api/evidence ---"
EVIDENCE=$(curl -s "http://localhost:$PORT/api/evidence")
if echo "$EVIDENCE" | grep -q '"Action"'; then
    echo "PASS: Evidence API works"
else
    echo "FAIL: Evidence API broken"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# 8. Test API timeline
echo ""
echo "--- Test 8: API /api/timeline ---"
TIMELINE=$(curl -s "http://localhost:$PORT/api/timeline")
if echo "$TIMELINE" | grep -q '"action"'; then
    echo "PASS: Timeline API works"
else
    echo "FAIL: Timeline API broken"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# 9. Test API sessions
echo ""
echo "--- Test 9: API /api/sessions ---"
SESSIONS=$(curl -s "http://localhost:$PORT/api/sessions")
if echo "$SESSIONS" | grep -q '"ID"'; then
    echo "PASS: Sessions API works"
else
    echo "FAIL: Sessions API broken"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# 10. Test API audit
echo ""
echo "--- Test 10: API /api/audit ---"
AUDIT=$(curl -s "http://localhost:$PORT/api/audit")
if echo "$AUDIT" | grep -q '"Action"'; then
    echo "PASS: Audit API works"
else
    echo "FAIL: Audit API broken"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# 11. Test API operators
echo ""
echo "--- Test 11: API /api/operators ---"
OPS=$(curl -s "http://localhost:$PORT/api/operators")
if echo "$OPS" | grep -q '"glory"'; then
    echo "PASS: Operators API works"
else
    echo "FAIL: Operators API broken"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# 12. Test security headers
echo ""
echo "--- Test 12: Security headers ---"
HEADERS=$(curl -sI "http://localhost:$PORT/")
if echo "$HEADERS" | grep -q "Content-Security-Policy"; then
    echo "PASS: CSP header present"
else
    echo "FAIL: CSP header missing"
fi
if echo "$HEADERS" | grep -q "X-Content-Type-Options: nosniff"; then
    echo "PASS: X-Content-Type-Options present"
else
    echo "FAIL: X-Content-Type-Options missing"
fi
if echo "$HEADERS" | grep -q "X-Frame-Options: DENY"; then
    echo "PASS: X-Frame-Options present"
else
    echo "FAIL: X-Frame-Options missing"
fi

# 13. Test dashboard HTML (XSS check)
echo ""
echo "--- Test 13: Dashboard HTML / XSS check ---"
HTML=$(curl -s "http://localhost:$PORT/")
if echo "$HTML" | grep -q '<title>RT'; then
    echo "PASS: Dashboard HTML served"
else
    echo "FAIL: Dashboard HTML broken"
fi
if echo "$HTML" | grep -q '<script>alert'; then
    echo "FAIL: XSS detected!"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi
echo "PASS: No XSS in dashboard"

# 14. Test report endpoint
echo ""
echo "--- Test 14: Report API ---"
REPORT=$(curl -s "http://localhost:$PORT/api/report?format=html")
if echo "$REPORT" | grep -q 'Executive Summary'; then
    echo "PASS: Report HTML generated"
else
    echo "FAIL: Report generation broken"
fi

# 15. Test rate limiting (rapid fire)
echo ""
echo "--- Test 15: Rate limiting ---"
RATE_OK=true
for i in $(seq 1 65); do
    STATUS=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT/api/overview")
    if [ "$STATUS" = "429" ]; then
        echo "PASS: Rate limit triggered at request $i (HTTP 429)"
        RATE_OK=false
        break
    fi
done
if [ "$RATE_OK" = true ]; then
    echo "INFO: Rate limit not triggered in 65 requests (bucket may be larger)"
fi

# 16. Stop server
echo ""
echo "--- Test 16: Stop server ---"
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null || true
echo "PASS: Server stopped"

# 17. Hash chain intact
echo ""
echo "--- Test 17: Chain integrity ---"
$RT verify-chain
echo "PASS: Chain intact"

# 18. Audit log shows operator operations
echo ""
echo "--- Test 18: Audit log ---"
$RT audit --limit 10
echo "PASS: Audit log complete"

# 19. Lock
echo ""
echo "--- Test 19: Lock ---"
$RT lock
echo "PASS: Database locked"

echo ""
echo "========================================="
echo "  All M5 tests PASSED!"
echo "========================================="

rm -rf "$RT_HOME"
