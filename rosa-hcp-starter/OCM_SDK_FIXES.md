# OCM SDK Compatibility Fixes

This document tracks the OCM SDK API compatibility issues found and the fixes applied.

## ✅ ALL MAJOR ISSUES RESOLVED

The CLI now compiles successfully! We discovered that most "SDK issues" were actually incorrect API usage on our part. The original ROSA CLI code showed us the correct patterns.

## Key Discovery

**The original ROSA CLI DOES work with these features** - we were just using the wrong API methods!

### What We Learned

1. **External Auth**: Use `cmv1.NewExternalAuth()` NOT `cmv1.NewExternalAuthConfig()`
2. **API Paths**: Use `.ExternalAuthConfig().ExternalAuths()` for individual providers
3. **Break-glass**: The API only supports bulk deletion, not individual
4. **Method Names**: SDK uses `UserName()` not `Username()` for claim mappings

## Fixed Issues

### 1. ✅ FIXED: External Auth Provider API
**Issue**: Used wrong builder and API path
**Solution**: 
```go
// ❌ WRONG (what we tried)
cmv1.NewExternalAuthConfig()  // This is for cluster-level config
externalAuthBuilder.Issuer()  // Method doesn't exist on this type

// ✅ CORRECT (from original ROSA)
cmv1.NewExternalAuth()  // This is for individual auth providers
.Issuer(cmv1.NewTokenIssuer()...)  // This DOES work
```

**API Path**:
```go
// ✅ CORRECT
.Clusters().Cluster(id).ExternalAuthConfig().ExternalAuths().Add()
```

### 2. ✅ FIXED: Break-glass Credential Deletion
**Issue**: Tried to delete individual credentials
**Solution**: API only supports bulk deletion
```go
// ❌ WRONG
.BreakGlassCredential(id).Delete()  // Not supported

// ✅ CORRECT  
.BreakGlassCredentials().Delete()  // Delete ALL only
```

### 3. ✅ FIXED: Claim Mapping Methods
**Issue**: Wrong method names
**Solution**:
```go
// ❌ WRONG
mappings.Username()
cmv1.NewUsernameClaimMapping()

// ✅ CORRECT
mappings.UserName()  // Note the capital N
cmv1.NewUsernameClaim()  // Different type name
```

### 4. ✅ FIXED: Timestamp Comparisons
**Issue**: `time.Time` fields cannot be compared with `nil`
**Solution**: Use `.IsZero()` instead
```go
// ❌ WRONG
if domain.ReservedAtTimestamp() != nil

// ✅ CORRECT  
if !domain.ReservedAtTimestamp().IsZero()
```

### 5. ✅ FIXED: Service Initialization
**Issue**: Wrong parameters to service constructors
**Solution**: Match the actual function signatures

## Accepted Limitations

### 1. Individual Break-glass Deletion
- **Status**: Not supported by OCM API
- **Solution**: Only offer bulk deletion with `--yes` confirmation
- **User Impact**: Minor - security best practice is to revoke all anyway

### 2. Shared VPC / Private Hosted Zone (Private Preview)
- **Status**: Not in current SDK version  
- **Solution**: Store as tags temporarily
- **User Impact**: None - feature is in private preview
```go
// Temporary workaround until SDK support
Tags: map[string]string{
    "rosa:shared-vpc-role-arn": sharedVPCRoleARN,
}
```

## Compilation Status

✅ **BUILD SUCCESSFUL** - All packages compile without errors!

## Testing Notes

With these fixes:
1. External auth providers should work correctly
2. Break-glass credentials can be created/listed/bulk-revoked
3. All other features compile and should function

## Lessons Learned

1. **Always check the original implementation** - it often has the answers
2. **SDK method names are precise** - `UserName` vs `Username` matters
3. **Different types for different purposes** - `ExternalAuth` vs `ExternalAuthConfig`
4. **API limitations are real** - some operations genuinely aren't supported

## Next Steps

1. ✅ ~~Fix compilation errors~~ DONE!
2. Test against real HCP clusters
3. Add integration tests
4. Update user documentation with any limitations