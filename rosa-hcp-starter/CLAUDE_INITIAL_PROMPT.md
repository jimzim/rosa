# Initial Claude Prompt

Copy and paste this to start a new conversation with Claude:

---

## Request: Create HCP-Only OCM SDK

I need help creating an HCP-only version of the OpenShift Cluster Manager (OCM) SDK for Go. This is a follow-up to successfully creating an HCP-only ROSA CLI that achieved an 83% code reduction.

### Context
- **Current location**: `/Users/jzimmerm/projects/redhat/ocm-sdk-go`
- **Reference HCP CLI**: `/Users/jzimmerm/projects/redhat/rosa/rosa-hcp-starter` (branch: `v2-hcp-only`) 
- **Goal**: Remove all Classic ROSA functionality, keep only HCP (Hosted Control Planes)
- **Expected outcome**: 80% smaller SDK, faster compilation, cleaner codebase

### Resources in Current Directory
1. `HCP_ONLY_IMPLEMENTATION_PLAN.md` - Detailed technical implementation guide
2. `CLAUDE_PROMPT_HCP_SDK.md` - Comprehensive requirements and patterns
3. `vendor/github.com/openshift-online/ocm-api-model/model/` - 366 model files to filter

### Your First Tasks
1. Read the `HCP_ONLY_IMPLEMENTATION_PLAN.md` to understand the approach
2. Analyze which SDK types are used by the HCP-only ROSA CLI (check imports in `/Users/jzimmerm/projects/redhat/rosa/rosa-hcp-starter/pkg/`)
3. Create a script to filter the model files from 366 to ~100 (removing machine_pool, flavour, classic types)
4. Test the generation process

The HCP-only ROSA CLI is fully functional with 60+ commands, so we know exactly which SDK features are needed. Please start by reviewing the implementation plan and the HCP CLI's SDK usage patterns.

Can you begin by analyzing the model dependencies and creating the filtering script?

---

## Additional Context for Follow-ups

If Claude needs more information, mention:
- The metamodel generator is at `metamodel_generator/` and is already built
- Generation is done via `hack/generate-client.sh`
- The HCP ROSA CLI has full implementations of: clusters, node pools, ingress, add-ons, external auth, break-glass credentials, DNS domains, identity providers
- All the work on the HCP CLI is documented in the Git history of the `v2-hcp-only` branch
