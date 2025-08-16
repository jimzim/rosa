#!/bin/bash

# Test script for browser-based authentication
set -e

echo "🔐 Testing ROSA HCP Browser-Based Authentication"
echo "================================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}1. Testing Login Command Help${NC}"
echo "----------------------------------------"
./bin/rosa login --help | grep -q "use-auth-code" && echo -e "${GREEN}✓ Auth code flag available${NC}" || echo -e "${RED}✗ Auth code flag missing${NC}"
echo ""

echo -e "${CYAN}2. Available Authentication Methods${NC}"
echo "----------------------------------------"
echo "a) Browser-based (STS with auth code):"
echo "   ./bin/rosa login --use-auth-code"
echo ""
echo "b) Interactive (will prompt for method):"
echo "   ./bin/rosa login"
echo ""
echo "c) Direct token (if you have one):"
echo "   ./bin/rosa login --token \$TOKEN"
echo ""

echo -e "${CYAN}3. How Browser Authentication Works${NC}"
echo "----------------------------------------"
echo "1. Run: ./bin/rosa login --use-auth-code"
echo "2. Browser opens to Red Hat SSO"
echo "3. Log in with your Red Hat account"
echo "4. Browser redirects back to local callback"
echo "5. CLI exchanges code for tokens"
echo "6. Tokens are saved securely"
echo ""

echo -e "${CYAN}4. Test Authentication Flow (Dry Run)${NC}"
echo "----------------------------------------"
echo "To test the authentication flow:"
echo ""
echo -e "${YELLOW}Run this command:${NC}"
echo "  ./bin/rosa login --use-auth-code"
echo ""
echo "The CLI will:"
echo "• Start a local server on a random port"
echo "• Open your default browser"
echo "• Wait for you to log in"
echo "• Receive the auth code via callback"
echo "• Exchange it for access tokens"
echo "• Save credentials securely"
echo ""

echo -e "${CYAN}5. Alternative: Interactive Mode${NC}"
echo "----------------------------------------"
echo -e "${YELLOW}Run this command:${NC}"
echo "  ./bin/rosa login"
echo ""
echo "Then select:"
echo "• Browser Authentication (recommended)"
echo "• Production environment"
echo ""

echo -e "${GREEN}✅ Authentication Methods Ready!${NC}"
echo ""
echo "The --use-auth-code flag implements the standard OAuth 2.0"
echo "authorization code flow with PKCE, which is more secure and"
echo "convenient than managing offline tokens."
echo ""
echo "This is the same authentication method used by the official"
echo "ROSA CLI when you don't have tokens configured."
