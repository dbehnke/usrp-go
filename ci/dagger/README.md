Integration Tests Dagger Module

This Dagger module provides functions to run integration tests inside containers for consistent, reproducible test execution.

Prerequisites:
- Dagger CLI installed. See https://dagger.io for details.

Usage:

```bash
# Run integration tests from repository root
dagger -m ci/dagger call test --source=.

# Get a configured test container for interactive debugging
dagger -m ci/dagger call test-container --source=. terminal
```

The module mounts the repository into an Ubuntu container and executes `test/containers/test-validator/run-integration-tests.sh`.
