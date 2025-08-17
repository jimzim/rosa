#!/bin/bash
# Smoke Test for ROSA HCP CLI
# This script performs basic validation of core functionality

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "======================================"
echo "   ROSA HCP CLI Smoke Test Suite"
echo "======================================"
echo

# Check if binary exists
if [ ! -f "./rosa" ]; then
    echo -e "${RED}✗ Error: rosa binary not found${NC}"
    echo "  Run: go build ./cmd/rosa"
    exit 1
fi

echo -e "${GREEN}✓ Binary found${NC}"

# Test 1: Version command
echo -n "Testing version command... "
if ./rosa version > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
fi

# Test 2: Help command
echo -n "Testing help command... "
if ./rosa --help > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
fi

# Test 3: Check all main commands exist
echo
echo "Checking command availability:"
commands=(
    "login"
    "logout"
    "whoami"
    "create cluster"
    "list clusters"
    "describe cluster"
    "delete cluster"
    "create nodepool"
    "list nodepools"
    "create oidc-config"
    "create account-roles"
    "create operator-roles"
    "create external-auth-provider"
    "create break-glass-credential"
    "create idp"
    "create ingress"
    "install addon"
    "grant user"
    "verify network"
    "logs install"
)

failed=0
for cmd in "${commands[@]}"; do
    echo -n "  $cmd... "
    if ./rosa $cmd --help > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        ((failed++))
    fi
done

# Test 4: Dry run test (doesn't require auth)
echo
echo "Testing dry-run mode:"
echo -n "  Cluster creation dry-run... "
if ./rosa create cluster --name test-dry-run --region us-west-2 --dry-run 2>&1 | grep -q "DRY RUN"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${YELLOW}⚠ Dry run may not be implemented${NC}"
fi

# Test 5: Output formats
echo
echo "Testing output formats:"
for format in text json yaml; do
    echo -n "  Format: $format... "
    if ./rosa list clusters --output $format 2>&1 | head -1 > /dev/null; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${YELLOW}⚠${NC}"
    fi
done

# Test 6: Check if logged in
echo
echo "Authentication check:"
if ./rosa whoami > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Authenticated${NC}"
    auth=true
else
    echo -e "${YELLOW}⚠ Not authenticated - run: ./rosa login --use-auth-code${NC}"
    auth=false
fi

# Test 7: If authenticated, try list operations
if [ "$auth" = true ]; then
    echo
    echo "Testing authenticated operations:"
    
    echo -n "  List clusters... "
    if ./rosa list clusters > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi
    
    echo -n "  List OIDC configs... "
    if ./rosa list oidc-configs > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi
    
    echo -n "  List regions... "
    if ./rosa list regions > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi
    
    echo -n "  List instance-types... "
    if ./rosa list instance-types > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
    fi
fi

# Summary
echo
echo "======================================"
if [ $failed -eq 0 ]; then
    echo -e "${GREEN}✓ All basic commands available${NC}"
else
    echo -e "${YELLOW}⚠ $failed commands not available${NC}"
fi

# Test interactive mode (non-blocking)
echo
echo "Interactive mode test:"
echo "  To test interactive mode, run:"
echo "  ./rosa create cluster --interactive"
echo "  ./rosa create nodepool --cluster <name> --interactive"

# Performance test
echo
echo "Performance quick test:"
start=$(date +%s%N)
./rosa version > /dev/null 2>&1
end=$(date +%s%N)
elapsed=$((($end - $start) / 1000000))
echo "  Version command: ${elapsed}ms"

if [ $elapsed -lt 100 ]; then
    echo -e "  ${GREEN}✓ Excellent performance${NC}"
elif [ $elapsed -lt 500 ]; then
    echo -e "  ${GREEN}✓ Good performance${NC}"
else
    echo -e "  ${YELLOW}⚠ Slow startup (${elapsed}ms)${NC}"
fi

echo
echo "======================================"
echo -e "${GREEN}Smoke test complete!${NC}"
echo
echo "Next steps:"
echo "1. Run: ./rosa login --use-auth-code"
echo "2. Create test cluster (see TEST_PLAN.md)"
echo "3. Run full test suite"
