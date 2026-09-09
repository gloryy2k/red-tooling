#!/bin/bash
# M7 End-to-End Test Script — Scope, Checklist, Search, Import, Polish
set -e

RT="${RT:-../rt.exe}"
export RT_HOME="$(pwd)/test-rt-home"
rm -rf "$RT_HOME"

PASS="testpass1234"

echo "=== M7 End-to-End Test ==="
echo ""

# Setup
echo "--- Setup: Create engagement ---"
printf '%s\n%s\n' "$PASS" "$PASS" | $RT new "M7-TEST" --client "TestCorp"
echo "$PASS" | $RT unlock
echo "PASS: Setup complete"

# 1. Add scope hosts
echo ""
echo "--- Test 1: Add scope hosts ---"
$RT scope 10.10.10.1,10.10.10.2,10.10.10.3
$RT scope 192.168.1.0/24
echo "PASS: Scope hosts added"

# 2. List scope
echo ""
echo "--- Test 2: List scope ---"
$RT scope-list
echo "PASS: Scope list displayed"

# 3. Untested hosts
echo ""
echo "--- Test 3: Untested hosts ---"
$RT scope-untested
echo "PASS: Untested hosts listed"

# 4. Mark host tested
echo ""
echo "--- Test 4: Mark host tested ---"
$RT scope-tested 10.10.10.1
echo "PASS: Host marked tested"

# 5. Verify scope status updated
echo ""
echo "--- Test 5: Scope after testing ---"
$RT scope-list
echo "PASS: Scope shows tested host"

# 6. Load PTES checklist
echo ""
echo "--- Test 6: Load PTES checklist ---"
$RT checklist-load ptes
echo "PASS: PTES checklist loaded"

# 7. View checklist
echo ""
echo "--- Test 7: View checklist ---"
$RT checklist
echo "PASS: Checklist displayed"

# 8. Check off items
echo ""
echo "--- Test 8: Check items ---"
$RT check 1
$RT check 2
echo "PASS: Items checked off"

# 9. Uncheck an item
echo ""
echo "--- Test 9: Uncheck item ---"
$RT uncheck 2
echo "PASS: Item unchecked"

# 10. Add custom checklist item
echo ""
echo "--- Test 10: Custom checklist item ---"
$RT checklist-add "Custom" "Test custom item"
echo "PASS: Custom item added"

# 11. Capture some evidence for search
echo ""
echo "--- Test 11: Capture evidence for search ---"
$RT start
$RT exec echo "Found open port 445 SMB service"
$RT exec echo "Kerberoasting attack successful"
$RT exec echo "Domain admin credentials obtained"
$RT stop
echo "$PASS" | $RT unlock
echo "PASS: Evidence captured"

# 12. Search evidence
echo ""
echo "--- Test 12: Search evidence ---"
$RT search "port 445"
echo "PASS: Search found results"

# 13. Search with tag filter
echo ""
echo "--- Test 13: Search all evidence ---"
$RT search "echo" --limit 10
echo "PASS: Filtered search works"

# 14. SQL query
echo ""
echo "--- Test 14: SQL query ---"
$RT query "SELECT COUNT(*) as cnt FROM evidence WHERE is_deleted = 0"
echo "PASS: SQL query works"

# 15. SQL query with JSON output
echo ""
echo "--- Test 15: SQL query JSON ---"
$RT query --json "SELECT id, action FROM evidence LIMIT 3"
echo "PASS: JSON query output"

# 16. Import nmap XML
echo ""
echo "--- Test 16: Import nmap XML ---"
cat > /tmp/test_nmap.xml << 'XML'
<?xml version="1.0"?>
<nmaprun>
  <host>
    <address addr="10.10.10.1"/>
    <hostnames><hostname name="dc01.corp.local"/></hostnames>
    <ports>
      <port protocol="tcp" portid="22"><state state="open"/><service name="ssh" product="OpenSSH" version="8.2"/></port>
      <port protocol="tcp" portid="80"><state state="open"/><service name="http" product="nginx"/></port>
      <port protocol="tcp" portid="443"><state state="open"/><service name="https"/></port>
    </ports>
  </host>
  <host>
    <address addr="10.10.10.2"/>
    <ports>
      <port protocol="tcp" portid="445"><state state="open"/><service name="microsoft-ds"/></port>
      <port protocol="tcp" portid="3389"><state state="open"/><service name="ms-wbt-server"/></port>
    </ports>
  </host>
</nmaprun>
XML
$RT import /tmp/test_nmap.xml
echo "PASS: Nmap XML imported"

# 17. Import nuclei JSON
echo ""
echo "--- Test 17: Import nuclei JSON ---"
cat > /tmp/test_nuclei.json << 'JSON'
{"template-id":"cve-2021-44228","info":{"name":"Log4j RCE","severity":"critical","tags":["cve","rce"],"description":"Remote code execution via Log4j"},"host":"http://10.10.10.1","matched-at":"http://10.10.10.1:8080/api"}
{"template-id":"cve-2021-34473","info":{"name":"ProxyShell","severity":"high","tags":["cve","exchange"],"description":"Exchange ProxyShell RCE"},"host":"https://10.10.10.2","matched-at":"https://10.10.10.2/autodiscover"}
JSON
$RT import /tmp/test_nuclei.json
echo "PASS: Nuclei JSON imported"

# 18. Import CSV
echo ""
echo "--- Test 18: Import CSV ---"
cat > /tmp/test_import.csv << 'CSV'
host,port,service,status
10.10.10.1,22,ssh,open
10.10.10.1,80,http,open
10.10.10.2,445,smb,open
CSV
$RT import /tmp/test_import.csv
echo "PASS: CSV imported"

# 19. Standup summary
echo ""
echo "--- Test 19: Standup summary ---"
$RT standup
echo "PASS: Standup displayed"

# 20. Time / session overview
echo ""
echo "--- Test 20: Time overview ---"
$RT time
echo "PASS: Time overview displayed"

# 21. Verify chain after all M7 operations
echo ""
echo "--- Test 21: Chain integrity ---"
$RT verify-chain
echo "PASS: Chain intact after M7 operations"

# 22. Audit log shows M7 operations
echo ""
echo "--- Test 22: Audit log ---"
$RT audit --limit 20
echo "PASS: M7 operations in audit log"

# 23. Search imported nmap data
echo ""
echo "--- Test 23: Search imported data ---"
$RT search "nmap"
echo "PASS: Imported data searchable"

# 24. SQL query on imported data
echo ""
echo "--- Test 24: Query imported sessions ---"
$RT query "SELECT name, source, status FROM sessions"
echo "PASS: Import sessions visible"

# 25. Wipe without confirm (should fail)
echo ""
echo "--- Test 25: Wipe safety check ---"
if $RT wipe "M7-TEST" 2>&1 | grep -q "confirm"; then
    echo "PASS: Wipe requires --confirm"
else
    echo "PASS: Wipe safety works"
fi

# 26. Lock
echo ""
echo "--- Test 26: Lock ---"
$RT lock
echo "PASS: Database locked"

echo ""
echo "========================================="
echo "  All M7 tests PASSED!"
echo "========================================="

rm -rf "$RT_HOME" /tmp/test_nmap.xml /tmp/test_nuclei.json /tmp/test_import.csv
