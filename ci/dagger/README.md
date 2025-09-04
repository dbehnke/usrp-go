Dagger Go integration runner

This small CLI uses the Dagger Go SDK to run the `test-validator` integration script inside a container.

Prerequisites:
- Dagger engine (desktop or remote) available. See https://dagger.io for details.
- Go 1.20+

Build and run locally:

```bash
cd ci/dagger
go build -o run
./run
```

The program mounts the repository into the container and executes `test/containers/test-validator/run-integration-tests.sh`.
