# Branch Protection Rules

## Main Branch Protection

The `main` branch is protected with the following rules:

### Required Status Checks ✅
- **Dagger Integration Tests** - Must pass before merging
- **Strict status checks** - Branch must be up-to-date before merging

### Pull Request Reviews 👥
- **1 approving review required** - At least one reviewer must approve
- **Dismiss stale reviews** - New commits invalidate previous approvals
- **Code owner reviews** - Not required (disabled for flexibility)

### Admin Enforcement 🔒
- **Enforce for admins** - Even repository administrators must follow these rules

## Workflow Integration

The protection rules are integrated with our GitHub Actions workflow:

- **Workflow**: `.github/workflows/test-integration.yml`
- **Job Name**: `Dagger Integration Tests` 
- **Purpose**: Runs comprehensive USRP protocol integration tests
- **Requirements**: All 23+ test cases must pass

## Testing Locally

Before creating a PR, ensure tests pass locally:

```bash
# Run integration tests via Dagger
just dagger-test

# Alternative: Direct Dagger call
dagger -m ci/dagger call test --source=.
```

## Bypass Procedures

In emergency situations, repository administrators can:

1. Temporarily disable branch protection
2. Make emergency changes
3. Re-enable protection immediately

However, this should be avoided as it bypasses quality gates.

## Benefits

- **Quality Assurance**: All changes must pass comprehensive testing
- **Code Review**: All changes require peer review
- **Amateur Radio Compliance**: USRP protocol changes are validated
- **Continuous Integration**: Automated testing prevents broken builds

---

*These rules ensure the USRP Go library maintains high quality and amateur radio protocol compliance.* 📻