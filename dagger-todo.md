# Dagger pipeline — options & TODO

This document collects implementation options, trade-offs, and next actions for the Dagger-based CI pipeline (current working code in `ci/dagger`). Use it to decide how far to extend the minimal runner into a full in-Dagger orchestration that replaces docker-compose in CI.

Quick contract
- Input: repository at repo root, `test/containers/*` build contexts, `test/integration/docker-compose.yml` (reference)
- Output: run integration tests (validator) and return non-zero on failure, produce useful logs
- Error modes: missing files, image build failures, network/timeouts, Dagger engine unavailable

Options

1) Minimal runner (current) — mount repo + exec validator
   - What: mount the repo into a single container and run `test/containers/test-validator/run-integration-tests.sh`.
   - Pros: trivial, fast to implement, uses existing scripts unchanged.
   - Cons: doesn't build images in Dagger or validate service images; still relies on local tooling inside script.
   - When to use: smoke checks, quick CI where validator encapsulates orchestration.

2) Build images in Dagger & run validator against them (recommended)
   - What: use Dagger to build each `test/containers/*` image, produce images, then run the validator in a container that can reach those images. Optionally use `dagger.Container().From(image)` for each built image.
   - Pros: reproducible builds, caching, no docker-compose required in CI, faster iterative runs via cache.
   - Cons: more code to implement, need to map compose networking semantics if services must talk to each other.
   - Implementation hints:
     - Build images with `client.Container().Build()` or by calling `client.Container().From("docker.io/library/...")` depending on API; persist images to a registry or use Dagger's in-memory images and run containers with `From` on the built ref.
     - Expose ports via a Dagger network or run validator that references local host addresses if using an external engine.

3) Full orchestration (compose -> Dagger) — emulate docker-compose inside Dagger
   - What: implement the compose graph in Dagger: build images, create containers, wire volumes, configure environment, and run long-running services in background, then run validator.
   - Pros: CI matches local dev environment closely, full control of service lifecycle.
   - Cons: highest effort; Dagger is not a drop-in replacement for docker-compose networking and may need extra glue (service readiness checks, UDP endpoints, port mapping strategies).
   - When to choose: if tests require multi-service interaction and you want CI and dev parity.

4) Use Dagger Cloud / remote engine
   - What: run pipeline on Dagger Cloud to offload heavy builds, speed up parallelism, and use remote caching.
   - Pros: faster builds, persistent caches, easy scaling.
   - Cons: requires Dagger Cloud account, secrets and access tokens management, potential cost.

CI integration patterns
- Option A (simple): add a GitHub Actions job that builds and runs `ci/dagger` (already added as `dagger-run`).
- Option B (native): invoke `dagger do` or `dagger` CLI to run a pipeline file or script; may be simpler if using Dagger CLI workflows.
- Cache: add Go module cache and optionally Dagger engine cache (if using dagger cloud credentials) to speed up runs.

Testing & quality gates
- Happy path test: validator exits 0 -> success
- Failure cases: image build fails, validator exits non-zero, timeouts. Ensure non-zero exit propagates to workflow.
- Add smoke tests that build a single image and run a trivial command inside it.

Security & secrets
- Keep secrets out of logs. If using Dagger Cloud or private registries, pass credentials via GitHub secrets and mount them as environment variables only when needed.

Edge cases to handle
- Large uploads: `Host().Directory(".")` uploads entire repo; consider `.dockerignore`-like filters or mounting only `test/containers` when building images.
- UDP services & non-TCP readiness checks — Dagger focuses on container execution; test readiness may need custom probes.
- Local developer experience: ensure `ci/dagger` can be run locally (it already does) and add a `--dry-run` or `--verbose` flag.

Estimated effort
- Minimal runner (already done): 1-2 hours
- Build images in Dagger + run validator: 1-2 days (depends on complexity of service wiring)
- Full compose replacement + robust readiness handling: 3-5 days

Next recommended steps (short-term)
1. Implement image builds in `ci/dagger` for `test/containers/*` and add a simple smoke-run that `From` the built image and runs `echo ok`.
2. Add Go module caching to the workflow (`actions/cache@v4`) for `~/.cache/go-build` and `$GOPATH/pkg/mod`.
3. Convert `dagger-run` job to run in parallel (remove `needs: integration`) if desired.
4. Optionally enable Dagger Cloud for faster builds if CI time is a problem.

Files of interest
- `ci/dagger/main.go` — current Go runner (mounts repo and runs validator)
- `test/containers/*` — build contexts for test images
- `.github/workflows/test-integration.yml` — workflow now includes `dagger-run`

If you want I can start Step 1 and implement image builds + a smoke-run in `ci/dagger` next. Reply with "build-images" to start that work. 
