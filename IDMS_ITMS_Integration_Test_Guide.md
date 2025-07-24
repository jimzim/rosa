# 🧪 **IDMS/ITMS Integration Test Guide**

## 📋 **Overview**

This guide provides step-by-step instructions for testing the complete IDMS/ITMS integration across all three repositories. Follow this guide to verify the implementation works end-to-end.

## 🛠️ **Prerequisites**

- **Go**: Version 1.23 or later
- **Make**: For building projects
- **Git**: For cloning repositories
- **Network Access**: To download dependencies

## 🚀 **Quick Setup for Testing**

### **Step 1: Clone All Three Repositories**

```bash
# Create a workspace directory
mkdir -p ~/rosa-idms-testing
cd ~/rosa-idms-testing

# Clone all three repositories with feature branches
git clone -b feat/IDMS-ITMS-support https://github.com/YOUR_USERNAME/ocm-api-model.git
git clone -b feat/IDMS-ITMS-support https://github.com/YOUR_USERNAME/ocm-sdk-go.git  
git clone -b feat/IDMS-ITMS-support https://github.com/YOUR_USERNAME/rosa.git
```

### **Step 2: Set Up Local Development Environment**

```bash
# Generate API model code
cd ocm-api-model
make generate
cd ..

# Generate and test OCM SDK
cd ocm-sdk-go
make generate
make examples  # This should compile without errors
cd ..

# Build ROSA CLI
cd rosa
make rosa  # This should build the rosa binary
cd ..
```

---

## 🧪 **Test Suite 1: Basic Compilation**

### **Test 1.1: Verify All Repositories Compile**

```bash
# Test API Model
cd ocm-api-model
echo "Testing API Model compilation..."
make generate
if [ $? -eq 0 ]; then
    echo "✅ API Model: PASSED"
else
    echo "❌ API Model: FAILED"
    exit 1
fi
cd ..

# Test OCM SDK
cd ocm-sdk-go
echo "Testing OCM SDK compilation..."
make examples
if [ $? -eq 0 ]; then
    echo "✅ OCM SDK: PASSED"
else
    echo "❌ OCM SDK: FAILED"
    exit 1
fi
cd ..

# Test ROSA CLI
cd rosa
echo "Testing ROSA CLI compilation..."
make rosa
if [ $? -eq 0 ]; then
    echo "✅ ROSA CLI: PASSED"
else
    echo "❌ ROSA CLI: FAILED"
    exit 1
fi
cd ..
```

### **Test 1.2: Verify New Types Exist in OCM SDK**

```bash
cd ocm-sdk-go

echo "Checking for IDMS/ITMS types in OCM SDK..."

# Check for ImageDigestMirrorSet
if grep -r "ImageDigestMirrorSet" clustersmgmt/v1/ > /dev/null; then
    echo "✅ ImageDigestMirrorSet type found"
else
    echo "❌ ImageDigestMirrorSet type NOT found"
fi

# Check for ImageTagMirrorSet
if grep -r "ImageTagMirrorSet" clustersmgmt/v1/ > /dev/null; then
    echo "✅ ImageTagMirrorSet type found"
else
    echo "❌ ImageTagMirrorSet type NOT found"
fi

# Check for ImageMirror
if grep -r "ImageMirror" clustersmgmt/v1/ > /dev/null; then
    echo "✅ ImageMirror type found"
else
    echo "❌ ImageMirror type NOT found"
fi

cd ..
```

---

## 🧪 **Test Suite 2: CLI Flag Testing**

### **Test 2.1: Verify New CLI Flags Exist**

```bash
cd rosa

echo "Testing ROSA CLI flag availability..."

# Test IDMS flag exists
if ./rosa create cluster --help | grep -q "registry-config-image-digest-mirror-sets"; then
    echo "✅ IDMS flag found in help"
else
    echo "❌ IDMS flag NOT found in help"
fi

# Test ITMS flag exists
if ./rosa create cluster --help | grep -q "registry-config-image-tag-mirror-sets"; then
    echo "✅ ITMS flag found in help"
else
    echo "❌ ITMS flag NOT found in help"
fi

# Display the help for manual verification
echo ""
echo "🔍 Registry config flags in help:"
./rosa create cluster --help | grep -A1 -B1 "registry-config-image"

cd ..
```

### **Test 2.2: Test Flag Parsing**

```bash
cd rosa

echo "Testing CLI flag parsing..."

# Test IDMS flag parsing (dry-run to avoid actual cluster creation)
echo "Testing IDMS flag parsing..."
./rosa create cluster test-idms-cluster \
  --registry-config-image-digest-mirror-sets="production-mirrors:registry.redhat.io=mirror1.company.com,mirror2.company.com" \
  --dry-run \
  --mode=auto \
  --yes 2>&1 | head -20

echo ""
echo "Testing ITMS flag parsing..."
# Test ITMS flag parsing
./rosa create cluster test-itms-cluster \
  --registry-config-image-tag-mirror-sets="dev-mirrors:quay.io=dev-mirror.company.com" \
  --dry-run \
  --mode=auto \
  --yes 2>&1 | head -20

cd ..
```

---

## 🧪 **Test Suite 3: Validation Testing**

### **Test 3.1: Test Format Validation**

```bash
cd rosa

echo "Testing validation for incorrect formats..."

# Test invalid format (should fail)
echo "Testing invalid IDMS format (should show validation error):"
./rosa create cluster test-invalid \
  --registry-config-image-digest-mirror-sets="invalid-format-no-colon" \
  --dry-run \
  --mode=auto \
  --yes 2>&1 | grep -i error || echo "No validation error shown"

echo ""
echo "Testing invalid registry format (should show validation error):"
./rosa create cluster test-invalid2 \
  --registry-config-image-digest-mirror-sets="test:invalid..registry=mirror.com" \
  --dry-run \
  --mode=auto \
  --yes 2>&1 | grep -i error || echo "No validation error shown"

cd ..
```

### **Test 3.2: Test Platform Registry Protection**

```bash
cd rosa

echo "Testing platform registry protection..."

# Test blocking Red Hat registries (should show warning or error)
echo "Testing platform registry protection (should show warning):"
./rosa create cluster test-platform \
  --registry-config-image-digest-mirror-sets="risky:registry.redhat.io=external-mirror.com" \
  --dry-run \
  --mode=auto \
  --yes 2>&1 | grep -i "warning\|error" || echo "No protection warning shown"

cd ..
```

---

## 🧪 **Test Suite 4: Unit Test Verification**

### **Test 4.1: Run ROSA CLI Unit Tests**

```bash
cd rosa

echo "Running ROSA CLI unit tests..."

# Run validation tests specifically
echo "Testing validation functions..."
go test ./pkg/clusterregistryconfig/... -v

echo ""
echo "Testing OCM integration..."
go test ./pkg/ocm/... -v

echo ""
echo "Running all CLI tests..."
go test ./cmd/create/cluster/... -v

cd ..
```

### **Test 4.2: Test Coverage Report**

```bash
cd rosa

echo "Generating test coverage report for new functionality..."

# Generate coverage for registry config package
go test ./pkg/clusterregistryconfig/... -coverprofile=coverage-registry.out
go tool cover -html=coverage-registry.out -o coverage-registry.html

echo "✅ Coverage report generated: coverage-registry.html"

cd ..
```

---

## 🧪 **Test Suite 5: Integration Scenarios**

### **Test 5.1: Multiple Mirror Sets**

```bash
cd rosa

echo "Testing multiple IDMS/ITMS configurations..."

# Test multiple IDMS entries
./rosa create cluster test-multiple \
  --registry-config-image-digest-mirror-sets="prod-mirrors:registry.redhat.io=mirror1.com,mirror2.com" \
  --registry-config-image-digest-mirror-sets="dev-mirrors:quay.io=dev-mirror.com" \
  --registry-config-image-tag-mirror-sets="tag-mirrors:docker.io=local-mirror.com" \
  --dry-run \
  --mode=auto \
  --yes 2>&1 | head -30

cd ..
```

### **Test 5.2: Mixed Registry Configurations**

```bash
cd rosa

echo "Testing IDMS/ITMS with other registry configurations..."

# Test IDMS/ITMS with blocked registries
./rosa create cluster test-mixed \
  --registry-config-image-digest-mirror-sets="mirrors:external.registry.com=internal.mirror.com" \
  --registry-config-blocked-registries="badregistry.io" \
  --registry-config-insecure-registries="insecure.registry.com" \
  --dry-run \
  --mode=auto \
  --yes 2>&1 | head -30

cd ..
```

---

## 🧪 **Test Suite 6: Error Scenarios**

### **Test 6.1: Version Compatibility**

```bash
cd rosa

echo "Testing OpenShift version compatibility..."

# Test with older OpenShift version (should show warning for IDMS/ITMS)
./rosa create cluster test-version \
  --version="4.12.30" \
  --registry-config-image-digest-mirror-sets="test:registry.com=mirror.com" \
  --dry-run \
  --mode=auto \
  --yes 2>&1 | grep -i "version\|warning\|error" || echo "No version compatibility check shown"

cd ..
```

### **Test 6.2: Limit Testing**

```bash
cd rosa

echo "Testing limits and boundaries..."

# Test with maximum number of mirror sets (implementation dependent)
# This tests the limits validation
./rosa create cluster test-limits \
  --registry-config-image-digest-mirror-sets="set1:reg1.com=mirror1.com" \
  --registry-config-image-digest-mirror-sets="set2:reg2.com=mirror2.com" \
  --registry-config-image-digest-mirror-sets="set3:reg3.com=mirror3.com" \
  --registry-config-image-digest-mirror-sets="set4:reg4.com=mirror4.com" \
  --registry-config-image-digest-mirror-sets="set5:reg5.com=mirror5.com" \
  --dry-run \
  --mode=auto \
  --yes 2>&1 | grep -i "limit\|maximum\|error" || echo "No limit validation shown"

cd ..
```

---

## 📊 **Test Results Summary**

After running all test suites, you should see:

### **✅ Expected PASSED Results**
- All three repositories compile successfully
- IDMS/ITMS types exist in OCM SDK
- CLI flags are recognized and parsed
- Validation functions work correctly
- Unit tests pass
- Integration scenarios work

### **⚠️ Expected Warnings**
- Platform registry protection warnings
- OpenShift version compatibility notices
- Configuration complexity warnings

### **❌ Expected FAILED Results (Should Show Errors)**
- Invalid format inputs
- Malformed registry names
- Exceeding configuration limits

---

## 🐛 **Troubleshooting Common Issues**

### **Issue 1: Compilation Errors**

```bash
# If you see "undefined: ImageDigestMirrorSet" errors:
cd ocm-api-model && make generate
cd ../ocm-sdk-go && make generate
cd ../rosa && go mod tidy && go mod vendor
```

### **Issue 2: Missing Dependencies**

```bash
# If modules are missing:
cd rosa
go mod download
go mod tidy
go mod vendor
```

### **Issue 3: Outdated Vendor Cache**

```bash
# If vendor directory is out of sync:
cd rosa
rm -rf vendor/
go mod vendor
```

---

## 🚀 **Advanced Testing Scenarios**

### **Performance Testing**

```bash
cd rosa

# Time the parsing of complex configurations
time ./rosa create cluster perf-test \
  --registry-config-image-digest-mirror-sets="test1:registry1.com=m1.com,m2.com,m3.com" \
  --registry-config-image-digest-mirror-sets="test2:registry2.com=m4.com,m5.com,m6.com" \
  --registry-config-image-tag-mirror-sets="test3:registry3.com=m7.com,m8.com,m9.com" \
  --dry-run --mode=auto --yes
```

### **Memory Usage Testing**

```bash
cd rosa

# Monitor memory usage during parsing
/usr/bin/time -v ./rosa create cluster memory-test \
  --registry-config-image-digest-mirror-sets="heavy-config:registry.com=mirror1.com,mirror2.com,mirror3.com,mirror4.com,mirror5.com" \
  --dry-run --mode=auto --yes 2>&1 | grep -i "memory\|resident"
```

---

## 📝 **Manual Verification Checklist**

- [ ] All repositories compile without errors
- [ ] New CLI flags appear in help output
- [ ] Flag parsing accepts valid configurations
- [ ] Validation rejects invalid configurations
- [ ] Platform registries show protection warnings
- [ ] Version compatibility is checked
- [ ] Unit tests pass
- [ ] Integration scenarios work
- [ ] Error messages are clear and helpful
- [ ] Performance is acceptable

---

## 📞 **Support**

If you encounter issues during testing:

1. **Check Logs**: Look for specific error messages
2. **Verify Versions**: Ensure all repos are on the correct branch
3. **Clean Build**: Try `make clean && make` in each repository
4. **Compare Working**: Compare with the reference implementation
5. **Report Issues**: Document and report any unexpected behavior

---

**Happy Testing!** 🎉

This comprehensive test guide ensures the IDMS/ITMS implementation works correctly across all scenarios and repositories. 