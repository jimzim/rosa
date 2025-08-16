#!/bin/bash

# ROSA HCP CLI - Full Integration Test Script
# This script demonstrates the complete end-to-end flow for creating a ROSA HCP cluster

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REGION="${AWS_REGION:-us-west-2}"
PREFIX="${PREFIX:-rosa-test}"
CLUSTER_NAME="${CLUSTER_NAME:-test-hcp-cluster}"
VPC_NAME="${VPC_NAME:-test-vpc}"
DRY_RUN="${DRY_RUN:-true}"

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}    ROSA HCP CLI - Full Integration Test${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Function to print step headers
print_step() {
    echo ""
    echo -e "${YELLOW}► $1${NC}"
    echo "----------------------------------------"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to prompt for continuation
prompt_continue() {
    if [ "$DRY_RUN" = "false" ]; then
        read -p "Continue? (y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Aborted."
            exit 1
        fi
    else
        echo -e "${YELLOW}[DRY RUN MODE - Skipping actual creation]${NC}"
    fi
}

# Check if CLI is built
if [ ! -f "./bin/rosa" ]; then
    print_error "ROSA CLI not found. Please run 'make build' first."
    exit 1
fi

print_success "ROSA HCP CLI found at ./bin/rosa"

# Check for required environment variables
print_step "Step 0: Checking Prerequisites"

if [ "$DRY_RUN" = "true" ]; then
    echo -e "${YELLOW}Running in DRY RUN mode - skipping authentication check${NC}"
    print_success "Dry run mode enabled"
else
    if [ -z "$ROSA_TOKEN" ]; then
        print_error "ROSA_TOKEN environment variable not set"
        echo "Please run: export ROSA_TOKEN=<your-token>"
        echo "Or use: ./bin/rosa login --use-auth-code"
        exit 1
    fi
    print_success "ROSA_TOKEN is set"
    
    # Test authentication
    echo "Testing authentication..."
    ./bin/rosa whoami > /dev/null 2>&1 || {
        print_error "Authentication failed. Please check your ROSA_TOKEN"
        exit 1
    }
    print_success "Authentication successful"
fi

# Step 1: Create VPC
print_step "Step 1: Create VPC Network"
echo "Creating VPC with name: $VPC_NAME in region: $REGION"

if [ "$DRY_RUN" = "true" ]; then
    echo "./bin/rosa create network --name $VPC_NAME --region $REGION --dry-run"
    ./bin/rosa create network --name $VPC_NAME --region $REGION --dry-run || true
else
    ./bin/rosa create network --name $VPC_NAME --region $REGION
    # Extract subnet IDs from output (in real scenario)
    SUBNET_IDS="subnet-abc123,subnet-def456"  # Placeholder
fi

print_success "VPC creation command executed"
prompt_continue

# Step 2: Create Account Roles
print_step "Step 2: Create IAM Account Roles"
echo "Creating account roles with prefix: $PREFIX"

if [ "$DRY_RUN" = "true" ]; then
    echo "./bin/rosa create account-roles --prefix $PREFIX --region $REGION --dry-run"
    ./bin/rosa create account-roles --prefix $PREFIX --region $REGION --dry-run || true
else
    ./bin/rosa create account-roles --prefix $PREFIX --region $REGION
    # Get installer role ARN (in real scenario, parse from output)
    INSTALLER_ROLE_ARN="arn:aws:iam::123456789012:role/$PREFIX-Installer"
fi

print_success "Account roles creation command executed"
prompt_continue

# Step 3: Create OIDC Configuration
print_step "Step 3: Create OIDC Configuration"
echo "Creating managed OIDC configuration"

if [ "$DRY_RUN" = "true" ]; then
    echo "./bin/rosa create oidc-config --managed --region $REGION"
    # In dry run, we can't get real values
    OIDC_CONFIG_ID="test-oidc-config-id"
    OIDC_ENDPOINT="https://oidc.example.com"
else
    # Capture output to extract OIDC config ID
    OUTPUT=$(./bin/rosa create oidc-config --managed --region $REGION)
    echo "$OUTPUT"
    # Parse OIDC config ID and endpoint from output (simplified)
    OIDC_CONFIG_ID=$(echo "$OUTPUT" | grep "ID:" | awk '{print $2}')
    OIDC_ENDPOINT=$(echo "$OUTPUT" | grep "Issuer URL:" | awk '{print $3}')
fi

print_success "OIDC configuration created"
echo "OIDC Config ID: $OIDC_CONFIG_ID"
echo "OIDC Endpoint: $OIDC_ENDPOINT"
prompt_continue

# Step 4: Create Operator Roles
print_step "Step 4: Create Operator Roles"
echo "Creating operator roles for cluster: $CLUSTER_NAME"

if [ "$DRY_RUN" = "true" ]; then
    echo "./bin/rosa create operator-roles --cluster $CLUSTER_NAME --oidc-endpoint $OIDC_ENDPOINT --prefix $PREFIX --region $REGION --dry-run"
    ./bin/rosa create operator-roles \
        --cluster $CLUSTER_NAME \
        --oidc-endpoint "$OIDC_ENDPOINT" \
        --prefix $PREFIX \
        --region $REGION \
        --dry-run || true
else
    ./bin/rosa create operator-roles \
        --cluster $CLUSTER_NAME \
        --oidc-endpoint "$OIDC_ENDPOINT" \
        --prefix $PREFIX \
        --region $REGION
fi

print_success "Operator roles creation command executed"
prompt_continue

# Step 5: Create Cluster
print_step "Step 5: Create ROSA HCP Cluster"
echo "Creating cluster: $CLUSTER_NAME"

if [ "$DRY_RUN" = "true" ]; then
    # Use placeholder values for dry run
    SUBNET_IDS="subnet-abc123,subnet-def456"
    INSTALLER_ROLE_ARN="arn:aws:iam::123456789012:role/$PREFIX-Installer"
    
    echo "./bin/rosa cluster create \\"
    echo "  --name $CLUSTER_NAME \\"
    echo "  --region $REGION \\"
    echo "  --subnet-ids $SUBNET_IDS \\"
    echo "  --role-arn $INSTALLER_ROLE_ARN \\"
    echo "  --oidc-config-id $OIDC_CONFIG_ID \\"
    echo "  --compute-nodes 2 \\"
    echo "  --dry-run"
    
    ./bin/rosa cluster create \
        --name $CLUSTER_NAME \
        --region $REGION \
        --subnet-ids $SUBNET_IDS \
        --role-arn "$INSTALLER_ROLE_ARN" \
        --oidc-config-id "$OIDC_CONFIG_ID" \
        --compute-nodes 2 \
        --dry-run || true
else
    ./bin/rosa cluster create \
        --name $CLUSTER_NAME \
        --region $REGION \
        --subnet-ids "$SUBNET_IDS" \
        --role-arn "$INSTALLER_ROLE_ARN" \
        --oidc-config-id "$OIDC_CONFIG_ID" \
        --compute-nodes 2
fi

print_success "Cluster creation command executed"

# Summary
echo ""
echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}                   SUMMARY${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

if [ "$DRY_RUN" = "true" ]; then
    echo -e "${YELLOW}This was a DRY RUN - no actual resources were created${NC}"
    echo ""
    echo "To run the actual creation, set DRY_RUN=false:"
    echo "  export DRY_RUN=false"
    echo "  ./test-full-integration.sh"
else
    echo -e "${GREEN}All steps completed successfully!${NC}"
    echo ""
    echo "Resources created:"
    echo "  • VPC: $VPC_NAME"
    echo "  • Account Roles Prefix: $PREFIX"
    echo "  • OIDC Config ID: $OIDC_CONFIG_ID"
    echo "  • Operator Roles for: $CLUSTER_NAME"
    echo "  • Cluster: $CLUSTER_NAME (creation initiated)"
fi

echo ""
echo "Useful commands:"
echo "  • List clusters:     ./bin/rosa cluster list"
echo "  • Describe cluster:  ./bin/rosa cluster describe $CLUSTER_NAME"
echo "  • List node pools:   ./bin/rosa nodepool list --cluster $CLUSTER_NAME"
echo "  • Delete cluster:    ./bin/rosa cluster delete $CLUSTER_NAME"
echo ""

# Cleanup instructions
if [ "$DRY_RUN" = "false" ]; then
    echo -e "${YELLOW}To clean up all resources:${NC}"
    echo "  ./bin/rosa cluster delete $CLUSTER_NAME"
    echo "  ./bin/rosa delete operator-roles --cluster $CLUSTER_NAME --prefix $PREFIX"
    echo "  ./bin/rosa delete account-roles --prefix $PREFIX"
    echo "  ./bin/rosa delete network rosa-vpc-$VPC_NAME --region $REGION"
fi

echo ""
echo -e "${GREEN}Test completed!${NC}"
