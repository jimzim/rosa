# Quick Reference: Claude + HCP SDK Setup

## ✅ What's Ready for Claude

### In the OCM SDK Repository (`/Users/jzimmerm/projects/redhat/ocm-sdk-go`)
1. **`HCP_ONLY_IMPLEMENTATION_PLAN.md`** - Technical implementation guide
2. **`CLAUDE_PROMPT_HCP_SDK.md`** - Comprehensive requirements document

### In the ROSA HCP Repository (`/Users/jzimmerm/projects/redhat/rosa/rosa-hcp-starter`)
1. **`CLAUDE_INITIAL_PROMPT.md`** - Ready-to-copy prompt for Claude
2. **Branch `v2-hcp-only`** - Complete HCP-only CLI implementation (reference)

## 🚀 How to Start

### Step 1: Start New Claude Session
```
Copy the content from: rosa-hcp-starter/CLAUDE_INITIAL_PROMPT.md
Paste as first message to Claude
```

### Step 2: Claude Will Access
```
/Users/jzimmerm/projects/redhat/ocm-sdk-go/
├── HCP_ONLY_IMPLEMENTATION_PLAN.md    # Implementation guide
├── CLAUDE_PROMPT_HCP_SDK.md           # Detailed requirements
├── vendor/.../model/                  # 366 model files to filter
└── hack/generate-client.sh            # Generation script
```

### Step 3: Claude Can Reference
```
/Users/jzimmerm/projects/redhat/rosa/rosa-hcp-starter/
├── pkg/*/service.go                   # Shows which SDK types are used
├── cmd/rosa/                          # Command implementations
└── go.mod                             # Current SDK version used
```

## 📝 Key Points for Claude Session

1. **Mention the v2-hcp-only branch** - It has the complete HCP CLI implementation
2. **Reference both repos** - SDK repo for changes, ROSA repo for validation
3. **Start with analysis** - Let Claude examine SDK usage before filtering
4. **Test incrementally** - Filter models → Generate → Test with HCP CLI

## 🎯 Success Metrics

Claude's implementation is successful when:
- [ ] Model files reduced from 366 to ~100
- [ ] Generated SDK is 80% smaller
- [ ] HCP-only ROSA CLI compiles with new SDK
- [ ] All 60+ HCP commands still work
- [ ] No Classic types remain in generated code

## 💡 Pro Tips

- Claude can run commands in both directories
- The metamodel generator is already built and ready
- Use `go mod replace` to test without changing imports
- The HCP CLI's service layers show exact SDK usage patterns

---

**Remember**: Everything Claude needs is already in place. The implementation plan, requirements, and reference CLI are all accessible on your local machine!
