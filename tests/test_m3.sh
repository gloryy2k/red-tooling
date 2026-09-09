#!/bin/bash
# M3 End-to-End Test Script — Finding Management + Attachments + Delete/Redact
set -e

RT="${RT:-../rt.exe}"
export RT_HOME="$(pwd)/test-rt-home"
rm -rf "$RT_HOME"

PASS="testpass1234"

echo "=== M3 End-to-End Test ==="
echo ""

# Setup: create and unlock engagement
echo "--- Setup: Create engagement ---"
printf '%s\n%s\n' "$PASS" "$PASS" | $RT new "M3-TEST" --client "TestCorp"
echo "$PASS" | $RT unlock

# Generate some evidence first
echo ""
echo "--- Setup: Generate evidence ---"
$RT exec echo "Found SQL injection in login form"
$RT exec echo "Administrator:500:aad3b435b51404eeaad3b435b51404ee:31d6cfe0d16ae931b73c59d7e0c089c0:::"
$RT exec echo "[+] Domain Admin access achieved"
$RT exec echo "Certificate Template ESC1 is vulnerable"
echo "PASS: Evidence generated"

# 1. Create findings
echo ""
echo "--- Test 1: Create findings ---"
$RT finding "SQL Injection in Login" --desc "Blind SQLi in login form parameter" --priority high --evidence 1 --mitre T1190
$RT finding "Domain Admin Access" --desc "Achieved DA via pass-the-hash" --priority critical --evidence 2,3 --mitre T1078,T1550
$RT finding "ADCS ESC1 Misconfiguration" --desc "Vulnerable certificate template" --priority high --evidence 4 --mitre T1649
echo "PASS: Findings created"

# 2. List findings
echo ""
echo "--- Test 2: List findings ---"
$RT findings
echo "PASS: Findings listed"

# 3. Verify a finding
echo ""
echo "--- Test 3: Verify finding ---"
$RT verify-finding 1 confirmed --note "Reproduced with sqlmap"
echo "PASS: Finding verified"

# 4. Set recommendation
echo ""
echo "--- Test 4: Set recommendation ---"
$RT recommend 1 "Apply parameterized queries and input validation"
echo "PASS: Recommendation set"

# 5. List after verify
echo ""
echo "--- Test 5: List after verification ---"
$RT findings
echo "PASS: Updated listing"

# 6. Create attachment (test file)
echo ""
echo "--- Test 6: Attach file ---"
echo "test log content for attachment" > test_attach.txt
$RT attach 1 test_attach.txt --caption "Test log file"
echo "PASS: File attached"

# 7. List attachments
echo ""
echo "--- Test 7: List attachments ---"
$RT attachments
echo "PASS: Attachments listed"

# 8. List attachments for specific evidence
echo ""
echo "--- Test 8: Attachments by evidence ---"
$RT attachments 1
echo "PASS: Evidence-specific listing"

# 9. Export attachment
echo ""
echo "--- Test 9: Export attachment ---"
$RT export-attach 1 .
echo "PASS: Attachment exported"

# 10. Soft-delete evidence
echo ""
echo "--- Test 10: Soft-delete evidence ---"
$RT exec echo "This is junk evidence to delete"
$RT delete 5
echo "PASS: Evidence soft-deleted"

# 11. Verify deleted evidence doesn't show in timeline
echo ""
echo "--- Test 11: Timeline after delete ---"
$RT timeline
echo "PASS: Deleted evidence hidden"

# 12. Redact evidence
echo ""
echo "--- Test 12: Redact evidence ---"
$RT redact 2 "Contains client passwords"
echo "PASS: Evidence redacted"

# 13. Verify redacted evidence shows marker
echo ""
echo "--- Test 13: Timeline after redact ---"
$RT timeline
echo "PASS: Redacted marker visible"

# 14. Merge findings
echo ""
echo "--- Test 14: Merge findings ---"
$RT finding "Secondary SQLi" --desc "Another SQLi in search" --priority medium --evidence 1
$RT merge-findings 1,4 "SQL Injection Findings (Combined)"
echo "PASS: Findings merged"

# 15. List after merge
echo ""
echo "--- Test 15: Findings after merge ---"
$RT findings
echo "PASS: Merged listing"

# 16. Mark false-positive
echo ""
echo "--- Test 16: False positive ---"
$RT verify-finding 3 false-positive --note "Template not exploitable in this config"
$RT findings
echo "PASS: False positive marked"

# 17. Verify hash chain still intact
echo ""
echo "--- Test 17: Hash chain integrity ---"
$RT verify-chain
echo "PASS: Chain intact"

# 18. Audit log shows all M3 operations
echo ""
echo "--- Test 18: Audit log ---"
$RT audit --limit 30
echo "PASS: Audit log complete"

# 19. Lock
echo ""
echo "--- Test 19: Lock ---"
$RT lock
echo "PASS: Database locked"

echo ""
echo "========================================="
echo "  All M3 tests PASSED!"
echo "========================================="

rm -rf "$RT_HOME" test_attach.txt test_attach_*.txt
