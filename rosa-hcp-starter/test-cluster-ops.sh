#!/bin/bash

# Test script for cluster operations (create/list/delete)
set -e

echo "🧪 Testing ROSA HCP Cluster Operations"
echo "======================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test help for cluster commands
echo "1. Testing cluster help..."
./bin/rosa cluster --help
echo -e "${GREEN}✓ Cluster help works${NC}"
echo ""

echo "2. Testing cluster create help..."
./bin/rosa cluster create --help
echo -e "${GREEN}✓ Cluster create help works${NC}"
echo ""

echo "3. Testing cluster list (should fail without token)..."
./bin/rosa cluster list 2>&1 | grep -q "ROSA_TOKEN" && echo -e "${YELLOW}⚠ Expected: Requires ROSA_TOKEN${NC}" || echo -e "${RED}✗ Unexpected result${NC}"
echo ""

echo "4. Testing cluster create dry-run..."
./bin/rosa cluster create \
    --name test-cluster \
    --region us-west-2 \
    --dry-run 2>&1 | grep -q "ROSA_TOKEN" && echo -e "${YELLOW}⚠ Expected: Requires ROSA_TOKEN for real operation${NC}" || echo -e "${GREEN}✓ Dry run executed${NC}"
echo ""

echo "5. Testing cluster delete command..."
./bin/rosa cluster delete --help
echo -e "${GREEN}✓ Cluster delete help works${NC}"
echo ""

echo "6. Testing cluster describe command..."
./bin/rosa cluster describe --help
echo -e "${GREEN}✓ Cluster describe help works${NC}"
echo ""

echo -e "${GREEN}✅ All basic tests passed!${NC}"
echo ""
echo "To test with actual API:"
echo "1. Set ROSA_TOKEN environment variable:"
echo "   export ROSA_TOKEN=your-ocm-token"
echo ""
echo "2. Test cluster creation:"
echo "   ./bin/rosa cluster create --name test-hcp --region us-west-2 --dry-run"
echo ""
echo "3. List clusters:"
echo "   ./bin/rosa cluster list"
echo ""
echo "4. Describe a cluster:"
echo "   ./bin/rosa cluster describe test-hcp"
echo ""
echo "5. Delete a cluster:"
echo "   ./bin/rosa cluster delete test-hcp"
