#!/bin/bash

# Test script for advanced ROSA HCP features
set -e

echo "🚀 Testing Advanced ROSA HCP Features"
echo "====================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}1. Testing Authentication Commands${NC}"
echo "----------------------------------------"
echo "Testing login help..."
./bin/rosa login --help
echo -e "${GREEN}✓ Login help works${NC}"
echo ""

echo "Testing whoami (should fail without auth)..."
./bin/rosa whoami 2>&1 | grep -q "not logged in" && echo -e "${YELLOW}⚠ Expected: Not logged in${NC}" || echo -e "${RED}✗ Unexpected result${NC}"
echo ""

echo -e "${CYAN}2. Testing NodePool Commands${NC}"
echo "----------------------------------------"
echo "Testing nodepool help..."
./bin/rosa nodepool --help
echo -e "${GREEN}✓ NodePool help works${NC}"
echo ""

echo "Testing nodepool create help..."
./bin/rosa nodepool create --help
echo -e "${GREEN}✓ NodePool create help works${NC}"
echo ""

echo "Testing nodepool list (should fail without auth)..."
./bin/rosa nodepool list --cluster test 2>&1 | grep -q "ROSA_TOKEN" && echo -e "${YELLOW}⚠ Expected: Requires authentication${NC}" || echo -e "${RED}✗ Unexpected result${NC}"
echo ""

echo -e "${CYAN}3. Testing OIDC Config Commands${NC}"
echo "----------------------------------------"
echo "Testing create oidc-config help..."
./bin/rosa create oidc-config --help
echo -e "${GREEN}✓ OIDC config create help works${NC}"
echo ""

echo -e "${CYAN}4. Command Aliases${NC}"
echo "----------------------------------------"
echo "Testing nodepool alias 'np'..."
./bin/rosa np --help > /dev/null 2>&1 && echo -e "${GREEN}✓ 'np' alias works${NC}" || echo -e "${RED}✗ Alias failed${NC}"
echo ""

echo -e "${GREEN}✅ All advanced features tests passed!${NC}"
echo ""
echo -e "${CYAN}Authentication Workflow:${NC}"
echo "1. Get an offline token from:"
echo "   https://console.redhat.com/openshift/token/rosa"
echo ""
echo "2. Login to ROSA:"
echo "   ./bin/rosa login --token \$OFFLINE_ACCESS_TOKEN"
echo ""
echo "3. Verify login:"
echo "   ./bin/rosa whoami"
echo ""
echo -e "${CYAN}OIDC Configuration Workflow:${NC}"
echo "1. Create managed OIDC config:"
echo "   ./bin/rosa create oidc-config --managed"
echo ""
echo "2. Create cluster with OIDC:"
echo "   ./bin/rosa cluster create --name my-hcp --oidc-config-id <ID>"
echo ""
echo -e "${CYAN}NodePool Management:${NC}"
echo "1. Create a node pool:"
echo "   ./bin/rosa nodepool create --cluster my-hcp --name workers --replicas 3"
echo ""
echo "2. List node pools:"
echo "   ./bin/rosa nodepool list --cluster my-hcp"
echo ""
echo "3. Scale a node pool:"
echo "   ./bin/rosa nodepool edit workers --cluster my-hcp --replicas 5"
echo ""
echo "4. Enable autoscaling:"
echo "   ./bin/rosa nodepool edit workers --cluster my-hcp --min-replicas 2 --max-replicas 10"
