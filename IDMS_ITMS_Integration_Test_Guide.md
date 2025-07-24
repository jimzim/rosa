# 🧪 **IDMS/ITMS Integration Test Guide**

## 📋 **Overview**

This guide provides step-by-step instructions for testing the complete IDMS/ITMS implementation across all three repositories. It includes setup instructions, test scenarios, and expected outcomes.

## 🎯 **Test Objectives**

1. ✅ Verify CLI flag recognition and parsing
2. ✅ Test input validation and error handling  
3. ✅ Confirm help text and documentation
4. ✅ Validate data structure handling
5. ⏳ Test end-to-end cluster creation (pending SDK fix)

---

## 🏗️ **Environment Setup**

### **Prerequisites**
- Go 1.21+ installed
- Git configured for your repositories
- Terminal with bash/zsh support

### **Step 1: Clone All Repositories**

```bash
# Create a dedicated testing directory
mkdir -p ~/idms-itms-testing
cd ~/idms-itms-testing

# Clone the three repositories
git clone https://github.com/your-fork/ocm-api-model.git
git clone https://github.com/your-fork/ocm-sdk-go.git  
git clone https://github.com/your-fork/rosa.git
```

### **Step 2: Switch to Feature Branches**

```bash
# Switch to the IDMS/ITMS feature branch in all repos
cd ocm-api-model && git checkout feat/IDMS-ITMS-support && cd ..
cd ocm-sdk-go && git checkout feat/IDMS-ITMS-support && cd ..
cd rosa && git checkout feat/IDMS-ITMS-support && cd ..
```

### **Step 3: Verify Local Integration**

```bash
# Check that the repositories are properly linked
cd rosa

# Verify go.mod has the correct replace directives
grep -A2 "replace.*ocm-sdk-go" go.mod
grep -A2 "replace.*ocm-api-model" ../ocm-sdk-go/go.mod
```

---

## ✅ **Test Suite 1: CLI Flag Recognition**

### **Test 1.1: Help Text Verification**

```bash
cd rosa

# Test 1: Verify IDMS flag is recognized
echo "🧪 Testing IDMS flag recognition..."
./rosa create cluster --help | grep -A3 -B1 "image-digest-mirror-sets"

# Expected: Flag should be listed with description
# ✅ PASS: Flag appears in help text
# ❌ FAIL: Flag not found or missing description
```

```bash
# Test 2: Verify ITMS flag is recognized  
echo "🧪 Testing ITMS flag recognition..."
./rosa create cluster --help | grep -A3 -B1 "image-tag-mirror-sets"

# Expected: Flag should be listed with description
# ✅ PASS: Flag appears in help text
# ❌ FAIL: Flag not found or missing description
```

### **Test 1.2: Flag Format Documentation**

```bash
# Test 3: Check examples in help text
echo "🧪 Testing help text examples..."
./rosa create cluster --help | grep -A10 -B5 "registry-config.*mirror.*sets"

# Expected: Should show example usage format
# ✅ PASS: Examples show proper format: "name:source:mirror1,mirror2|name2:..."
# ❌ FAIL: No examples or incorrect format shown
```

---

## ✅ **Test Suite 2: Input Validation**

### **Test 2.1: Valid Input Acceptance**

```bash
echo "🧪 Testing valid IDMS input..."
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="production:registry.example.com:quay.io,docker.io" \
  --dry-run

# Expected: No validation errors, should proceed with dry-run
# ✅ PASS: Command accepts input and shows parsed configuration
# ❌ FAIL: Validation errors for valid input
```

```bash
echo "🧪 Testing valid ITMS input..."
./rosa create cluster test-cluster \
  --registry-config-image-tag-mirror-sets="staging:registry.internal.com:mirror1.io,mirror2.io" \
  --dry-run

# Expected: No validation errors, should proceed with dry-run
# ✅ PASS: Command accepts input and shows parsed configuration
# ❌ FAIL: Validation errors for valid input
```

### **Test 2.2: Invalid Input Rejection**

```bash
echo "🧪 Testing invalid format rejection..."
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="invalid-format-here" \
  --dry-run

# Expected: Clear validation error explaining correct format
# ✅ PASS: Helpful error message with format explanation
# ❌ FAIL: No validation or unclear error message
```

```bash
echo "🧪 Testing invalid registry format..."
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="test:not-a-valid-registry:mirror.io" \
  --dry-run

# Expected: Registry format validation error
# ✅ PASS: Error explains registry format requirements
# ❌ FAIL: Invalid registry format accepted
```

### **Test 2.3: Platform Registry Protection**

```bash
echo "🧪 Testing platform registry protection..."
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="test:registry.redhat.io:external-mirror.com" \
  --dry-run

# Expected: Warning or error about platform registry mirroring
# ✅ PASS: Clear warning about platform registry risks
# ❌ FAIL: Platform registries allowed without warning
```

---

## ✅ **Test Suite 3: Complex Configuration**

### **Test 3.1: Multiple Mirror Sets**

```bash
echo "🧪 Testing multiple IDMS configurations..."
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="prod:registry.example.com:mirror1.io,mirror2.io|staging:registry.internal.com:mirror3.io" \
  --dry-run

# Expected: Both mirror sets parsed correctly
# ✅ PASS: Multiple sets shown in output
# ❌ FAIL: Only one set parsed or parsing errors
```

### **Test 3.2: Mixed IDMS and ITMS**

```bash
echo "🧪 Testing IDMS and ITMS together..."
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="prod-digest:registry.example.com:mirror1.io" \
  --registry-config-image-tag-mirror-sets="prod-tag:registry.example.com:mirror2.io" \
  --dry-run

# Expected: Both types accepted and configured
# ✅ PASS: Both IDMS and ITMS configurations shown
# ❌ FAIL: Conflict or only one type accepted
```

### **Test 3.3: Integration with Other Registry Options**

```bash
echo "🧪 Testing with other registry configurations..."
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="test:registry.example.com:mirror.io" \
  --registry-config-blocked-registries="blocked.example.com" \
  --registry-config-insecure-registries="insecure.example.com" \
  --dry-run

# Expected: All registry configurations work together
# ✅ PASS: All registry options shown in configuration
# ❌ FAIL: Conflicts between different registry options
```

---

## ✅ **Test Suite 4: Validation Logic**

### **Test 4.1: Name Validation**

```bash
echo "🧪 Testing Kubernetes naming validation..."
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="Invalid_Name_With_Underscores:registry.io:mirror.io" \
  --dry-run

# Expected: Error about Kubernetes naming conventions
# ✅ PASS: Clear error about invalid name format
# ❌ FAIL: Invalid names accepted
```

### **Test 4.2: Configuration Limits**

```bash
echo "🧪 Testing configuration limits..."
# Create a very long configuration to test limits
LONG_CONFIG="set1:registry1.io:mirror1.io|set2:registry2.io:mirror2.io|set3:registry3.io:mirror3.io|set4:registry4.io:mirror4.io|set5:registry5.io:mirror5.io"
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="$LONG_CONFIG" \
  --dry-run

# Expected: May have limits on number of mirror sets
# ✅ PASS: Either accepts or gives clear limit error
# ❌ FAIL: Crashes or unclear error on large configurations
```

---

## ✅ **Test Suite 5: CLI User Experience**

### **Test 5.1: Error Message Quality**

```bash
echo "🧪 Testing error message helpfulness..."

# Test empty value
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="" \
  --dry-run

# Test malformed entry
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="name-only" \
  --dry-run

# Expected: Clear, actionable error messages with examples
# ✅ PASS: Error messages explain what's wrong and how to fix
# ❌ FAIL: Generic or unclear error messages
```

### **Test 5.2: Command Completion**

```bash
echo "🧪 Testing tab completion (if available)..."

# Test if flags auto-complete
./rosa create cluster --registry-config-image-<TAB>

# Expected: Should show both IDMS and ITMS flag options
# ✅ PASS: Both flags appear in completion
# ❌ FAIL: No completion or missing flags
```

---

## ⏳ **Test Suite 6: End-to-End Testing (Pending SDK Fix)**

> **Note**: These tests require the OCM SDK compilation issues to be resolved.

### **Test 6.1: Actual Cluster Creation**

```bash
echo "🧪 Testing real cluster creation..."
# This test will be enabled once SDK compilation is fixed

# ./rosa create cluster test-idms-cluster \
#   --cluster-name="test-idms-$(date +%s)" \
#   --registry-config-image-digest-mirror-sets="test:registry.example.com:mirror.io" \
#   --mode=auto \
#   --yes

# Expected: Cluster created with IDMS configuration
# ⏳ PENDING: Wait for OCM SDK compilation fix
```

### **Test 6.2: Cluster Description**

```bash
echo "🧪 Testing cluster description display..."
# This test will be enabled once SDK compilation is fixed

# ./rosa describe cluster test-idms-cluster

# Expected: Shows IDMS/ITMS configuration in cluster description
# ⏳ PENDING: Wait for OCM SDK compilation fix
```

---

## 📊 **Test Results Summary**

### **Automated Test Runner**

Create this test script for automated verification:

```bash
#!/bin/bash
# File: run_idms_itms_tests.sh

echo "🚀 Starting IDMS/ITMS Integration Tests..."

# Set test cluster name
CLUSTER_NAME="test-idms-$(date +%s)"

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

test_command() {
    local description="$1"
    local command="$2"
    local expected="$3"
    
    echo "🧪 $description"
    
    if eval "$command" &>/dev/null; then
        if [[ "$expected" == "success" ]]; then
            echo "✅ PASS"
            ((TESTS_PASSED++))
        else
            echo "❌ FAIL (Expected failure but command succeeded)"
            ((TESTS_FAILED++))
        fi
    else
        if [[ "$expected" == "failure" ]]; then
            echo "✅ PASS"
            ((TESTS_PASSED++))
        else
            echo "❌ FAIL (Expected success but command failed)"
            ((TESTS_FAILED++))
        fi
    fi
}

# Run tests
test_command "Help text includes IDMS flag" \
    "./rosa create cluster --help | grep -q 'image-digest-mirror-sets'" \
    "success"

test_command "Help text includes ITMS flag" \
    "./rosa create cluster --help | grep -q 'image-tag-mirror-sets'" \
    "success"

test_command "Valid IDMS configuration accepted" \
    "./rosa create cluster $CLUSTER_NAME --registry-config-image-digest-mirror-sets='test:registry.io:mirror.io' --dry-run" \
    "success"

test_command "Invalid format rejected" \
    "./rosa create cluster $CLUSTER_NAME --registry-config-image-digest-mirror-sets='invalid-format' --dry-run" \
    "failure"

# Summary
echo "📊 Test Results:"
echo "✅ Passed: $TESTS_PASSED"
echo "❌ Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo "🎉 All tests passed!"
    exit 0
else
    echo "❌ Some tests failed!"
    exit 1
fi
```

### **Run All Tests**

```bash
# Make the test script executable and run it
chmod +x run_idms_itms_tests.sh
./run_idms_itms_tests.sh
```

---

## 🔧 **Troubleshooting**

### **Common Issues and Solutions**

#### **Issue 1: Flags Not Recognized**
```
Error: unknown flag: --registry-config-image-digest-mirror-sets
```
**Solution**: Verify you're using the correct branch and build:
```bash
git branch  # Should show feat/IDMS-ITMS-support
make rosa   # Rebuild the binary
```

#### **Issue 2: Validation Not Working**
```
Invalid configurations are accepted without errors
```
**Solution**: Check that validation code is properly integrated:
```bash
grep -r "ValidateCompleteImageMirrorSetConfiguration" cmd/create/cluster/
```

#### **Issue 3: SDK Compilation Errors**
```
undefined: api_v1.ImageDigestMirrorSetBuilder
```
**Solution**: This is expected until OCM SDK alias generation is fixed. The CLI functionality still works for validation and parsing.

---

## 📈 **Expected Test Results**

### **✅ Should Pass (Current Implementation)**
- CLI flag recognition and help text
- Input parsing and validation
- Error message quality
- Integration with other registry options
- Dry-run cluster creation

### **⏳ Pending (After SDK Fix)**
- Actual cluster creation with IDMS/ITMS
- Backend integration and persistence
- Cluster description display

### **🎯 Success Criteria**
- 90%+ of CLI tests pass
- Clear, helpful error messages
- Comprehensive validation
- Good user experience

---

## 🚀 **Next Steps After Testing**

1. **Document Test Results**: Record which tests pass/fail
2. **Report Issues**: Create tickets for any failures
3. **Validate with Team**: Share results with other developers
4. **Plan SDK Integration**: Coordinate with OCM SDK maintainers
5. **E2E Testing**: Test against real OCM backend once SDK is ready

---

**This test guide ensures comprehensive validation of the IDMS/ITMS implementation while providing clear success criteria and troubleshooting guidance.** 