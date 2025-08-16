# Message for New Chat

Copy and paste this message to start your new chat:

---

I'm continuing work on rewriting the ROSA CLI from scratch for HCP-only support. The project is in the `rosa-hcp-starter/` directory on branch `v2-hcp-only`.

## What's Been Completed:
- Core MVP with cluster create/list/describe/delete commands
- Full NodePool CRUD operations  
- Browser-based authentication with PKCE (`rosa login --use-auth-code`)
- OIDC configuration creation
- Modern Go architecture with Result types, structured logging, and Charm libraries
- Fixed AWS field requirement - clusters need existing VPC with subnet IDs

## Current Status:
The basic CLI is working but requires:
1. Existing VPC with subnets (use `rosa create network` from original CLI)
2. Pre-created IAM roles
3. OIDC configuration

## What Needs to Be Done Next:
1. Implement `rosa create account-roles` and `rosa create operator-roles` for IAM setup
2. Add VPC creation helper or validation  
3. Implement `rosa edit cluster` and `rosa upgrade cluster`
4. Add more comprehensive validations and error messages
5. Support for addons and managed services

The full context is in `CONTEXT_SUMMARY.md`. All core commands are working but need the AWS/IAM integration completed for a full end-to-end experience without depending on the original ROSA CLI for setup tasks.

The main issue resolved in the last session was understanding that ROSA HCP requires an existing VPC and the AWS field is mandatory in cluster creation.

Please help me continue with [specify what you want to work on next].
