#!/bin/bash

# ROSA HCP CLI MVP Test Script
# This script helps test the basic functionality of the HCP-only ROSA CLI

set -e

echo "🚀 ROSA HCP CLI MVP Test"
echo "========================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    echo "Please install Go 1.23+ from https://golang.org/dl/"
    exit 1
fi

echo -e "${GREEN}✓ Go is installed:${NC} $(go version)"

# Build the CLI
echo ""
echo "Building ROSA HCP CLI..."
if go build -o bin/rosa ./cmd/rosa; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

# Check version
echo ""
echo "Checking version..."
./bin/rosa --version || true

# Test help command
echo ""
echo "Testing help command..."
./bin/rosa --help

echo ""
echo "Testing cluster create help..."
./bin/rosa cluster create --help

# Test dry-run cluster creation
echo ""
echo -e "${YELLOW}Testing cluster creation (dry-run)...${NC}"
./bin/rosa cluster create \
    --name test-hcp-cluster \
    --region us-west-2 \
    --dry-run || echo "Note: This requires ROSA_TOKEN to be set"

echo ""
echo -e "${GREEN}✅ MVP Test Complete!${NC}"
echo ""
echo "Next steps:"
echo "1. Set up your OCM token:"
echo "   export ROSA_TOKEN=your-ocm-token"
echo ""
echo "2. Configure AWS credentials:"
echo "   aws configure"
echo ""
echo "3. Create a real cluster:"
echo "   ./bin/rosa cluster create --interactive"
echo ""
echo "Or with specific options:"
echo "   ./bin/rosa cluster create \\"
echo "     --name my-cluster \\"
echo "     --region us-west-2 \\"
echo "     --role-arn arn:aws:iam::123:role/Installer \\"
echo "     --support-role-arn arn:aws:iam::123:role/Support \\"
echo "     --worker-iam-role arn:aws:iam::123:role/Worker"
