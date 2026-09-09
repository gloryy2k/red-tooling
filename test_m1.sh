#!/bin/bash
# M1 End-to-End Test Script
set -e

RT="./rt.exe"
export RT_HOME="$(pwd)/test-rt-home"
rm -rf "$RT_HOME"

PASS="testpass1234"

echo "=== M1 End-to-End Test ==="
echo ""

# 1. Create engagement
echo "--- Test 1: Create engagement ---"
printf '%s\n%s\n' "$PASS" "$PASS" | $RT new "TEST-ENG" --client "TestCorp"
echo "PASS: Engagement created"

# 2. List engagements
echo ""
echo "--- Test 2: List engagements ---"
$RT ls
echo "PASS: List works"

# 3. Status (locked — new engagement is locked after creation)
echo ""
echo "--- Test 3: Status (locked) ---"
$RT status
echo "PASS: Status works"

# 4. Verify encrypted file exists
echo ""
echo "--- Test 4: Encrypted file check ---"
ENC_FILE="$RT_HOME/data/TEST-ENG.db.enc"
if [ -f "$ENC_FILE" ]; then
    echo "PASS: Encrypted .db.enc file exists"
else
    echo "FAIL: No encrypted file"
    exit 1
fi

# 5. Unlock
echo ""
echo "--- Test 5: Unlock ---"
echo "$PASS" | $RT unlock
echo "PASS: Unlocked"

# 6. Status (unlocked)
echo ""
echo "--- Test 6: Status (unlocked) ---"
$RT status
echo "PASS: Status shows unlocked"

# 7. Start session (db already unlocked, no passphrase needed)
echo ""
echo "--- Test 7: Start session + capture commands ---"
# We can't run interactive start in a script, so use 'exec' to capture commands
$RT exec echo "hello from rt"
echo "PASS: Exec captured"

# 8. Tag/milestone/bookmark/note
echo ""
echo "--- Test 8: Annotations ---"
$RT tag "Found interesting service on port 445"
$RT milestone "Initial foothold via SMB"
$RT bookmark "Check ADCS later"
$RT note "Client confirmed scope includes 10.10.10.0/24"
echo "PASS: All annotations created"

# 9. Timeline
echo ""
echo "--- Test 9: Timeline ---"
$RT timeline
echo "PASS: Timeline works"

# 10. Verify chain
echo ""
echo "--- Test 10: Verify chain ---"
$RT verify-chain
echo "PASS: Chain verification works"

# 11. Status with data
echo ""
echo "--- Test 11: Status with data ---"
$RT status

# 12. Lock
echo ""
echo "--- Test 12: Lock ---"
$RT lock
echo "PASS: Database locked"

# 13. Verify locked
echo ""
echo "--- Test 13: Verify locked state ---"
$RT status
PLAIN_FILE="$RT_HOME/data/TEST-ENG.db"
if [ ! -f "$PLAIN_FILE" ]; then
    echo "PASS: Plain .db file removed after lock"
else
    echo "FAIL: Plain .db file still exists"
    exit 1
fi

# 14. Encrypted file not readable as SQLite
echo ""
echo "--- Test 14: Encrypted file integrity ---"
HEADER=$(xxd -l 6 -p "$ENC_FILE")
echo "Encrypted file header: $HEADER"
if [[ "$HEADER" == "5254454e43"* ]]; then
    echo "PASS: RTENC magic header present"
else
    echo "FAIL: Missing RT encryption header"
    exit 1
fi

echo ""
echo "========================================="
echo "  All M1 tests PASSED!"
echo "========================================="

# Cleanup
rm -rf "$RT_HOME"
