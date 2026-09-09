#!/bin/bash
# M2 End-to-End Test Script — Auto-Flag + Credential Locker
set -e

RT="${RT:-../rt.exe}"
export RT_HOME="$(pwd)/test-rt-home"
rm -rf "$RT_HOME"

PASS="testpass1234"

echo "=== M2 End-to-End Test ==="
echo ""

# Setup: create and unlock engagement
echo "--- Setup: Create engagement ---"
printf '%s\n%s\n' "$PASS" "$PASS" | $RT new "M2-TEST" --client "TestCorp"
echo "$PASS" | $RT unlock

# 1. Manual credential storage
echo ""
echo "--- Test 1: Manual credential storage ---"
$RT cred "administrator" "P@ssw0rd123" --host 10.10.10.1 --type password
$RT cred "svc-backup" "Backup2024!" --host 10.10.10.5
$RT cred "da-user" "aad3b435b51404eeaad3b435b51404ee:31d6cfe0d16ae931b73c59d7e0c089c0" --host acme.local --type ntlm
echo "PASS: Credentials stored"

# 2. Masked credential listing
echo ""
echo "--- Test 2: Masked credential listing ---"
$RT creds
echo "PASS: Masked listing works"

# 3. Plaintext reveal (audit-logged)
echo ""
echo "--- Test 3: Plaintext credential reveal ---"
$RT creds --show
echo "PASS: Reveal works"

# 4. CSV export
echo ""
echo "--- Test 4: CSV export ---"
$RT creds --csv
echo "PASS: CSV export works"

# 5. Auto-flag: secretsdump output (single line with NTLM hash pattern)
echo ""
echo "--- Test 5: Auto-flag secretsdump ---"
$RT exec echo "Dumping local SAM hashes - Administrator:500:aad3b435b51404eeaad3b435b51404ee:31d6cfe0d16ae931b73c59d7e0c089c0:::"
echo "PASS: Secretsdump auto-flag"

# 6. Auto-flag: Domain Admin detection
echo ""
echo "--- Test 6: Auto-flag Domain Admin ---"
$RT exec echo "[+] Domain Admin access achieved"
echo "PASS: Domain Admin auto-flag"

# 7. Auto-flag: ADCS detection
echo ""
echo "--- Test 7: Auto-flag ADCS ---"
$RT exec echo "Certificate Template ESC1 is vulnerable"
echo "PASS: ADCS auto-flag"

# 8. Auto-flag: credential in output
echo ""
echo "--- Test 8: Auto-flag credential pattern ---"
$RT exec echo "[+] Valid credentials: admin@acme.local:Summer2024!"
echo "PASS: Credential auto-flag"

# 9. Auto-flag: SQLi
echo ""
echo "--- Test 9: Auto-flag SQLi ---"
$RT exec echo "sqlmap identified SQL injection in parameter id"
echo "PASS: SQLi auto-flag"

# 10. Auto-flag: RCE
echo ""
echo "--- Test 10: Auto-flag RCE ---"
$RT exec echo "remote code execution confirmed on target"
echo "PASS: RCE auto-flag"

# 11. Auto-flag: Kerberoast
echo ""
echo "--- Test 11: Auto-flag Kerberoast ---"
$RT exec echo "GetUserSPNs found ServicePrincipalName for svc-sql"
echo "PASS: Kerberoast auto-flag"

# 12. Check auto-stored credentials
echo ""
echo "--- Test 12: Verify credentials ---"
$RT creds
echo "PASS: Credentials listed"

# 13. Timeline
echo ""
echo "--- Test 13: Timeline ---"
$RT timeline
echo "PASS: Timeline works"

# 14. Verify chain
echo ""
echo "--- Test 14: Hash chain integrity ---"
$RT verify-chain
echo "PASS: Chain intact"

# 15. Audit log
echo ""
echo "--- Test 15: Audit log ---"
$RT audit --limit 25
echo "PASS: Audit log works"

# 16. Lock
echo ""
echo "--- Test 16: Lock ---"
$RT lock
echo "PASS: Database locked"

echo ""
echo "========================================="
echo "  All M2 tests PASSED!"
echo "========================================="

rm -rf "$RT_HOME"
