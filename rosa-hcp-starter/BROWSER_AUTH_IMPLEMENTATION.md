# Browser-Based Authentication (STS) Implementation

## ✅ Feature Complete: `rosa login --use-auth-code`

### Overview
I've implemented browser-based authentication using the OAuth 2.0 authorization code flow with PKCE (Proof Key for Code Exchange). This is the standard method used by ROSA and other Red Hat CLIs for secure authentication without requiring users to manage offline tokens.

## 🚀 How to Use

### Method 1: Direct Browser Authentication
```bash
./bin/rosa login --use-auth-code
```

This will:
1. Start a local HTTP server on a random port
2. Open your default browser to Red Hat SSO
3. You log in with your Red Hat account credentials
4. Browser redirects back to `http://127.0.0.1:<port>/callback`
5. CLI receives the authorization code
6. Exchanges code for access and refresh tokens
7. Saves tokens securely for future use

### Method 2: Interactive Mode
```bash
./bin/rosa login
```

When prompted, select:
- **Browser Authentication (recommended)**
- **Production** environment

### Method 3: If You Still Have a Token
```bash
# Still supported for compatibility
./bin/rosa login --token $OFFLINE_ACCESS_TOKEN
```

## 🔒 Security Features

### PKCE (Proof Key for Code Exchange)
- Generates a cryptographic code verifier
- Creates SHA256 challenge for the authorization request
- Prevents authorization code interception attacks

### CSRF Protection
- Generates random state parameter
- Validates state on callback to prevent CSRF attacks

### Local-Only Callback
- Uses `127.0.0.1` (localhost) for callback
- Random port selection to avoid conflicts
- No external network exposure

## 🏗️ Implementation Details

### OAuth Flow
```
User                    CLI                     Red Hat SSO
 |                       |                           |
 |---> rosa login        |                           |
 |     --use-auth-code   |                           |
 |                       |                           |
 |                 [Start local server]              |
 |                 [Generate PKCE verifier]          |
 |                 [Generate state]                  |
 |                       |                           |
 |<--- Browser opens ----|                           |
 |                       |                           |
 |-----------------------------------------> Login   |
 |                       |                           |
 |<------- Redirect with auth code ------------------|
 |                       |                           |
 |---> Callback -------->|                           |
 |                       |                           |
 |                       |---> Exchange code ------->|
 |                       |                           |
 |                       |<--- Access tokens --------|
 |                       |                           |
 |<--- Success -----------|                           |
```

### Supported Environments
- **Production** (default): `https://sso.redhat.com`
- **Staging**: `https://sso.stage.redhat.com`
- **Integration**: `https://sso.integration.redhat.com`

### Token Storage
```yaml
# Tokens saved to ~/.rosa/config.yaml
api_url: https://api.openshift.com
refresh_token: <refresh_token>
# Access token stored separately in ~/.rosa/token
```

## 🎯 Benefits Over Token-Based Auth

1. **No Token Management**: Users don't need to obtain offline tokens
2. **Familiar Flow**: Same as logging into any web service
3. **Automatic Refresh**: Refresh tokens handle expiration
4. **Better Security**: PKCE prevents code interception
5. **SSO Integration**: Works with Red Hat SSO/Keycloak

## 🧪 Testing

### Test the Flow
```bash
# Test browser authentication
./bin/rosa login --use-auth-code

# Verify login succeeded
./bin/rosa whoami

# Now you can use all commands
./bin/rosa cluster list
./bin/rosa nodepool list --cluster my-cluster
```

### Troubleshooting

#### Browser Doesn't Open
The CLI will display the URL - copy and paste it manually:
```
Opening browser for authentication...
If the browser doesn't open automatically, visit:
https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/auth?client_id=...
```

#### Port Already in Use
The CLI automatically selects a random available port. If issues persist, try again.

#### Authentication Timeout
The CLI waits 5 minutes for authentication. If it times out, run the command again.

## 📝 Configuration

### OAuth Client ID
Default: `ocm-cli` (standard ROSA/OCM client)

To use a custom client:
```bash
./bin/rosa login --use-auth-code --client-id my-custom-client
```

### Environment Selection
```bash
# Production (default)
./bin/rosa login --use-auth-code

# Staging
./bin/rosa login --use-auth-code --env staging

# Integration
./bin/rosa login --use-auth-code --env integration
```

## 🔄 Token Refresh

The CLI automatically handles token refresh using the stored refresh token. You don't need to re-authenticate unless:
- The refresh token expires (typically 30 days)
- You explicitly log out
- You switch environments

## 🎉 Summary

The `--use-auth-code` flag provides a seamless, secure authentication experience that:
- **Works with your Red Hat account** (no special tokens needed)
- **Opens your browser** for familiar web-based login
- **Handles everything automatically** (token exchange, storage, refresh)
- **Follows OAuth 2.0 best practices** (PKCE, state validation)

This is the recommended way to authenticate with ROSA going forward!
